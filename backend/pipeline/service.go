package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"release-launcher/backend/build"
	"release-launcher/backend/models"
	"release-launcher/backend/packaging"
	"release-launcher/backend/transfer"
)

const EventRunFinished = "release_run_finished"

// Service runs build and package operations for each component in order.
type Service struct {
	builder  componentBuilder
	packager *packaging.Service
	sender   componentSender
}

type componentBuilder interface {
	BuildComponent(context.Context, models.BuildRun, models.Project, models.Component, build.EventSink) models.BuildResult
}

type componentSender interface {
	SendOne(context.Context, string, models.Project, models.Component, string, models.DaliConfig, transfer.EventSink) models.TransferResult
}

type metadataSender interface {
	SendOneWithMetadata(context.Context, string, models.Project, models.Component, string, string, string, string, models.DaliConfig, transfer.EventSink) models.TransferResult
}

func NewService(builder componentBuilder, packager *packaging.Service, senders ...componentSender) *Service {
	service := &Service{builder: builder, packager: packager}
	if len(senders) > 0 {
		service.sender = senders[0]
	}
	return service
}

// PrepareRun validates the selected project/components and creates initial state.
func (s *Service) PrepareRun(runID string, project models.Project, request models.PackageRequest) (models.ReleaseRun, error) {
	plan, err := s.packager.PlanRequest(project, request)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	selected := make(map[string]bool, len(request.ComponentIDs))
	for _, id := range request.ComponentIDs {
		selected[strings.TrimSpace(id)] = true
	}
	states := make([]models.ReleaseComponentState, 0, len(project.Components))
	for _, component := range project.Components {
		packageStatus := models.PackageStatusReady
		packageMessage := "Waiting"
		if !component.Package.Enabled {
			packageStatus = models.PackageStatusSkipped
			packageMessage = "Packaging disabled"
		}
		if !selected[component.ID] {
			packageStatus = models.PackageStatusSkipped
			packageMessage = "Not selected"
		}
		states = append(states, models.ReleaseComponentState{
			ComponentID:     component.ID,
			ComponentName:   component.Name,
			Selected:        selected[component.ID],
			BuildStatus:     buildStatusForSelection(selected[component.ID]),
			BuildMessage:    messageForSelection(selected[component.ID]),
			PackageStatus:   packageStatus,
			PackageMessage:  packageMessage,
			TransferStatus:  transferStatusFor(component, selected[component.ID]),
			TransferMessage: transferMessageFor(component, selected[component.ID]),
		})
	}
	if plan.Version == "" {
		return models.ReleaseRun{}, fmt.Errorf("package version could not be determined")
	}
	return models.ReleaseRun{
		ID:          runID,
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Version:     plan.Version,
		Status:      models.ReleaseRunStatusRunning,
		Components:  states,
		StartTime:   time.Now(),
	}, nil
}

// Execute builds and packages each selected component before moving forward.
func (s *Service) Execute(ctx context.Context, run models.ReleaseRun, project models.Project, request models.PackageRequest, emit func(models.BuildEvent)) models.ReleaseRun {
	return s.execute(ctx, run, project, request, nil, emit)
}

// ExecuteBuildPackageAndSend runs the complete build, package, and Dali transfer workflow.
func (s *Service) ExecuteBuildPackageAndSend(ctx context.Context, run models.ReleaseRun, project models.Project, request models.PackageRequest, settings models.DaliConfig, emit func(models.BuildEvent)) models.ReleaseRun {
	return s.execute(ctx, run, project, request, &settings, emit)
}

func (s *Service) execute(ctx context.Context, run models.ReleaseRun, project models.Project, request models.PackageRequest, settings *models.DaliConfig, emit func(models.BuildEvent)) models.ReleaseRun {
	request.Version = run.Version
	plan, err := s.packager.PlanRequest(project, request)
	if err != nil {
		run.Status = models.ReleaseRunStatusFailed
		run.Error = err.Error()
		return s.finish(run, emit)
	}
	components := make(map[string]models.Component, len(project.Components))
	items := make(map[string]models.PackagePlanItem, len(plan.Components))
	for _, component := range project.Components {
		components[component.ID] = component
	}
	for _, item := range plan.Components {
		items[item.ComponentID] = item
	}
	s.emit(emit, models.BuildEvent{Type: build.EventRunStarted, Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Version: run.Version, Timestamp: time.Now()})

	for i := range run.Components {
		state := &run.Components[i]
		if !state.Selected {
			continue
		}
		if err := ctx.Err(); err != nil {
			state.BuildStatus = models.BuildStatusCancelled
			state.BuildMessage = "Build cancelled before this component started"
			state.PackageStatus = models.PackageStatusSkipped
			state.PackageMessage = "Packaging cancelled"
			state.TransferStatus = models.TransferStatusSkipped
			state.TransferMessage = "Transfer cancelled"
			s.emitReleaseState(emit, run, *state)
			s.markRemainingSkipped(run.Components[i+1:], "Cancelled", emit, run)
			run.Status = models.ReleaseRunStatusCancelled
			run.Error = "Build and package cancelled"
			break
		}

		component := components[state.ComponentID]
		state.BuildStatus = models.BuildStatusBuilding
		state.BuildMessage = "Building..."
		s.emitReleaseState(emit, run, *state)
		buildRun := models.BuildRun{ID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName}
		buildResult := s.builder.BuildComponent(ctx, buildRun, project, component, func(event models.BuildEvent) {
			event.Phase = "build"
			event.Version = run.Version
			s.emit(emit, event)
		})
		state.BuildResult = &buildResult
		state.BuildStatus = buildResult.Status
		state.BuildMessage = buildResult.Error
		if state.BuildMessage == "" {
			state.BuildMessage = statusMessage(string(buildResult.Status))
		}
		if !buildResult.Success {
			state.PackageStatus = models.PackageStatusSkipped
			state.PackageMessage = "Build did not succeed"
			state.TransferStatus = models.TransferStatusSkipped
			state.TransferMessage = "Build did not succeed"
			s.emitReleaseState(emit, run, *state)
			if buildResult.Status == models.BuildStatusCancelled {
				run.Status = models.ReleaseRunStatusCancelled
				run.Error = "Build and package cancelled"
				s.markRemainingSkipped(run.Components[i+1:], "Cancelled", emit, run)
			} else {
				run.Status = models.ReleaseRunStatusFailed
				run.Error = fmt.Sprintf("%s failed to build.", component.Name)
				s.markRemainingSkipped(run.Components[i+1:], "Previous component failed", emit, run)
			}
			break
		}
		if !component.Package.Enabled {
			state.PackageStatus = models.PackageStatusSkipped
			state.PackageMessage = "Packaging disabled"
			state.TransferStatus = models.TransferStatusSkipped
			state.TransferMessage = "Packaging disabled"
			s.emitReleaseState(emit, run, *state)
			continue
		}

		state.PackageStatus = models.PackageStatusPackaging
		state.PackageMessage = "Packaging..."
		s.emitReleaseState(emit, run, *state)
		item := items[component.ID]
		packageResult := s.packager.PackageOneWithMetadata(ctx, run.ID, project, component, buildResult.OutputPath, item.PackagePath, request.Overwrite, item.FilenameTemplate, item.ResolvedFilename, run.Version)
		state.PackageResult = &packageResult
		state.PackageStatus = packageResult.Status
		state.PackageMessage = packageResult.Error
		if state.PackageMessage == "" {
			state.PackageMessage = statusMessage(string(packageResult.Status))
		}
		s.emitReleaseState(emit, run, *state)
		if !packageResult.Success {
			state.TransferStatus = models.TransferStatusSkipped
			state.TransferMessage = "Packaging did not succeed"
			s.emitReleaseState(emit, run, *state)
			if packageResult.Status == models.PackageStatusCancelled {
				run.Status = models.ReleaseRunStatusCancelled
				run.Error = "Build and package cancelled"
				s.markRemainingSkipped(run.Components[i+1:], "Cancelled", emit, run)
			} else {
				run.Status = models.ReleaseRunStatusFailed
				run.Error = fmt.Sprintf("%s failed to package.", component.Name)
				s.markRemainingSkipped(run.Components[i+1:], "Previous component failed", emit, run)
			}
			break
		}
		if settings == nil {
			continue
		}
		if s.sender == nil {
			state.TransferStatus = models.TransferStatusFailed
			state.TransferMessage = "Dali transfer service is unavailable"
			s.emitReleaseState(emit, run, *state)
			run.Status = models.ReleaseRunStatusFailed
			run.Error = "Dali transfer service is unavailable"
			s.markRemainingSkipped(run.Components[i+1:], "Previous transfer failed", emit, run)
			break
		}
		state.TransferStatus = models.TransferStatusSending
		state.TransferMessage = "Sending..."
		s.emitReleaseState(emit, run, *state)
		transferResult := s.sendOne(ctx, run.ID, project, component, packageResult.PackagePath, item.FilenameTemplate, item.ResolvedFilename, run.Version, *settings, func(event models.BuildEvent) {
			event.Phase = transfer.Phase
			event.Version = run.Version
			s.emit(emit, event)
		})
		state.TransferResult = &transferResult
		state.TransferStatus = transferResult.Status
		state.TransferMessage = transferResult.Error
		if state.TransferMessage == "" {
			state.TransferMessage = statusMessage(string(transferResult.Status))
		}
		s.emitReleaseState(emit, run, *state)
		if !transferResult.Success {
			if transferResult.Status == models.TransferStatusCancelled {
				run.Status = models.ReleaseRunStatusCancelled
				run.Error = "Build, package, and transfer cancelled"
				s.markRemainingSkipped(run.Components[i+1:], "Cancelled", emit, run)
			} else {
				run.Status = models.ReleaseRunStatusFailed
				run.Error = fmt.Sprintf("%s failed to send.", component.Name)
				s.markRemainingSkipped(run.Components[i+1:], "Previous transfer failed", emit, run)
			}
			break
		}
	}
	if run.Status == models.ReleaseRunStatusRunning {
		run.Status = models.ReleaseRunStatusCompleted
	}
	return s.finish(run, emit)
}

func (s *Service) sendOne(ctx context.Context, runID string, project models.Project, component models.Component, packagePath, filenameTemplate, resolvedFilename, version string, settings models.DaliConfig, emit transfer.EventSink) models.TransferResult {
	if sender, ok := s.sender.(metadataSender); ok {
		return sender.SendOneWithMetadata(ctx, runID, project, component, packagePath, filenameTemplate, resolvedFilename, version, settings, emit)
	}
	return s.sender.SendOne(ctx, runID, project, component, packagePath, settings, emit)
}

func (s *Service) finish(run models.ReleaseRun, emit func(models.BuildEvent)) models.ReleaseRun {
	run.EndTime = time.Now()
	s.emit(emit, models.BuildEvent{Type: EventRunFinished, Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Version: run.Version, RunStatus: models.BuildRunStatus(run.Status), Error: run.Error, Timestamp: run.EndTime})
	return run
}

func (s *Service) emit(sink func(models.BuildEvent), event models.BuildEvent) {
	if sink != nil {
		sink(event)
	}
}

func (s *Service) emitReleaseState(sink func(models.BuildEvent), run models.ReleaseRun, state models.ReleaseComponentState) {
	s.emit(sink, models.BuildEvent{Type: "release_component_state", Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, ComponentID: state.ComponentID, ComponentName: state.ComponentName, Version: run.Version, Status: state.BuildStatus, PackageStatus: state.PackageStatus, Result: state.BuildResult, PackageResult: state.PackageResult, TransferStatus: state.TransferStatus, TransferResult: state.TransferResult, Error: state.BuildMessage + " | " + state.PackageMessage + " | " + state.TransferMessage, Timestamp: time.Now()})
}

func (s *Service) markRemainingSkipped(states []models.ReleaseComponentState, reason string, emit func(models.BuildEvent), run models.ReleaseRun) {
	for i := range states {
		if !states[i].Selected {
			continue
		}
		states[i].BuildStatus = models.BuildStatusSkipped
		states[i].BuildMessage = reason
		states[i].PackageStatus = models.PackageStatusSkipped
		states[i].PackageMessage = reason
		states[i].TransferStatus = models.TransferStatusSkipped
		states[i].TransferMessage = reason
		s.emitReleaseState(emit, run, states[i])
	}
}

func transferStatusFor(component models.Component, selected bool) models.TransferStatus {
	if !selected || !component.Package.Enabled {
		return models.TransferStatusSkipped
	}
	return models.TransferStatusReady
}

func transferMessageFor(component models.Component, selected bool) string {
	if !selected {
		return "Not selected"
	}
	if !component.Package.Enabled {
		return "Packaging disabled"
	}
	return "Waiting"
}

func buildStatusForSelection(selected bool) models.BuildStatus {
	if selected {
		return models.BuildStatusReady
	}
	return models.BuildStatusSkipped
}

func messageForSelection(selected bool) string {
	if selected {
		return "Ready"
	}
	return "Not selected"
}

func statusMessage(status string) string {
	if status == "" {
		return "Ready"
	}
	return strings.ToUpper(status[:1]) + status[1:]
}
