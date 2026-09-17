package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"release-launcher/backend/build"
	"release-launcher/backend/models"
	"release-launcher/backend/packaging"
)

const EventRunFinished = "release_run_finished"

// Service runs build and package operations for each component in order.
type Service struct {
	builder  componentBuilder
	packager *packaging.Service
}

type componentBuilder interface {
	BuildComponent(context.Context, models.BuildRun, models.Project, models.Component, build.EventSink) models.BuildResult
}

func NewService(builder componentBuilder, packager *packaging.Service) *Service {
	return &Service{builder: builder, packager: packager}
}

// PrepareRun validates the selected project/components and creates initial state.
func (s *Service) PrepareRun(runID string, project models.Project, request models.PackageRequest) (models.ReleaseRun, error) {
	plan, err := s.packager.Plan(project, request.ComponentIDs, request.Version)
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
			ComponentID:    component.ID,
			ComponentName:  component.Name,
			Selected:       selected[component.ID],
			BuildStatus:    buildStatusForSelection(selected[component.ID]),
			BuildMessage:   messageForSelection(selected[component.ID]),
			PackageStatus:  packageStatus,
			PackageMessage: packageMessage,
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
	plan, err := s.packager.Plan(project, request.ComponentIDs, run.Version)
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

		state.PackageStatus = models.PackageStatusPackaging
		state.PackageMessage = "Packaging..."
		s.emitReleaseState(emit, run, *state)
		item := items[component.ID]
		packageResult := s.packager.PackageOne(ctx, run.ID, project, component, buildResult.OutputPath, item.PackagePath, request.Overwrite)
		state.PackageResult = &packageResult
		state.PackageStatus = packageResult.Status
		state.PackageMessage = packageResult.Error
		if state.PackageMessage == "" {
			state.PackageMessage = statusMessage(string(packageResult.Status))
		}
		s.emitReleaseState(emit, run, *state)
		if !packageResult.Success {
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
	}
	if run.Status == models.ReleaseRunStatusRunning {
		run.Status = models.ReleaseRunStatusCompleted
	}
	return s.finish(run, emit)
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
	s.emit(sink, models.BuildEvent{Type: "release_component_state", Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, ComponentID: state.ComponentID, ComponentName: state.ComponentName, Version: run.Version, Status: state.BuildStatus, PackageStatus: state.PackageStatus, Result: state.BuildResult, PackageResult: state.PackageResult, Error: state.BuildMessage + " | " + state.PackageMessage, Timestamp: time.Now()})
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
		s.emitReleaseState(emit, run, states[i])
	}
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
