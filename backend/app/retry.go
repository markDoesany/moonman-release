package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"release-launcher/backend/build"
	"release-launcher/backend/models"
	"release-launcher/backend/transfer"
)

// retryContext is intentionally kept in memory. Activity history remains the
// durable path for loading a complete release for review after a restart;
// stage retries are available for the current application session.
type retryContext struct {
	operation       string
	project         models.Project
	buildRequest    *models.BuildRequest
	packageRequest  *models.PackageRequest
	transferRequest *models.TransferRequest
	buildRun        *models.BuildRun
	packageRun      *models.PackageRun
	releaseRun      *models.ReleaseRun
	transferRun     *models.TransferRun
	runID           string
	attempt         int
}

func (a *App) rememberRetryContext(ctx retryContext) {
	a.retryMu.Lock()
	defer a.retryMu.Unlock()
	a.retryRuns[ctx.runID] = ctx
}

func (a *App) retryContextFor(runID string) (retryContext, bool) {
	a.retryMu.Lock()
	defer a.retryMu.Unlock()
	ctx, ok := a.retryRuns[strings.TrimSpace(runID)]
	return ctx, ok
}

// RetryRun retries only the requested failed stage. Successful component work
// is copied into the new attempt and is never executed again.
func (a *App) RetryRun(request models.RetryRequest) (models.RetryRun, error) {
	if a.loadErr != nil {
		return models.RetryRun{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	if request.Stage != models.RetryStageBuild && request.Stage != models.RetryStagePackage && request.Stage != models.RetryStageTransfer {
		return models.RetryRun{}, errors.New("retry stage must be build, package, or transfer")
	}
	previous, ok := a.retryContextFor(request.RunID)
	if !ok {
		return models.RetryRun{}, errors.New("this run can no longer be retried; load it for review and start a new release")
	}

	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel != nil {
		return models.RetryRun{}, errors.New("another build, packaging, or transfer operation is already running")
	}

	ids := request.ComponentIDs
	if len(ids) == 0 {
		ids = retryableComponentIDs(previous, request.Stage)
	}
	if len(ids) == 0 {
		return models.RetryRun{}, errors.New("there are no failed components available for this retry")
	}

	newID := a.nextRunID()
	attempt := previous.attempt + 1
	if attempt < 2 {
		attempt = 2
	}
	eventContext := a.runtimeCtx

	switch previous.operation {
	case "build":
		if previous.buildRequest == nil || previous.buildRun == nil {
			return models.RetryRun{}, errors.New("build retry data is unavailable")
		}
		runRequest := *previous.buildRequest
		runRequest.ComponentIDs = ids
		run, err := a.buildService.PrepareRunForEnvironment(newID, previous.project, ids, runRequest.Environment)
		if err != nil {
			return models.RetryRun{}, err
		}
		ctx, cancel, done := a.beginActiveRunLocked(run.ID)
		go func() {
			finalRun := a.buildService.Execute(ctx, run, previous.project, a.eventSink(eventContext))
			a.recordRetryBuild(finalRun, runRequest, previous, request.Stage, attempt)
			a.rememberRetryContext(retryContext{operation: "build", project: previous.project, buildRequest: &runRequest, buildRun: &finalRun, runID: finalRun.ID, attempt: attempt})
			a.finishActiveRun(run.ID, cancel, done)
		}()
		return models.RetryRun{Operation: "build", Stage: request.Stage, Attempt: attempt, RetryOfRunID: previous.runID, Build: &run}, nil

	case "package":
		if previous.packageRequest == nil || previous.packageRun == nil {
			return models.RetryRun{}, errors.New("package retry data is unavailable")
		}
		runRequest := clonePackageRequest(*previous.packageRequest)
		runRequest.ComponentIDs = ids
		plan, err := a.packager.PlanRequest(previous.project, runRequest)
		if err != nil {
			return models.RetryRun{}, err
		}
		if plan.HasConflicts && !runRequest.Overwrite {
			return models.RetryRun{}, errors.New("a retry would replace an existing package; review the release before retrying")
		}
		run, err := a.packager.PrepareRunRequest(newID, previous.project, runRequest)
		if err != nil {
			return models.RetryRun{}, err
		}
		ctx, cancel, done := a.beginActiveRunLocked(run.ID)
		go func() {
			finalRun := a.packager.ExecuteRequest(ctx, run, previous.project, runRequest, a.eventSink(eventContext))
			a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "package", ComponentIDs: runRequest.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: runRequest.FilenameTemplate, ApprovedPackageNames: runRequest.PackageNames, ReleaseDirectory: plan.ReleaseDirectory, RetryOfRunID: previous.runID, Attempt: attempt, RetryStage: string(request.Stage)})
			a.rememberRetryContext(retryContext{operation: "package", project: previous.project, packageRequest: &runRequest, packageRun: &finalRun, runID: finalRun.ID, attempt: attempt})
			a.finishActiveRun(run.ID, cancel, done)
		}()
		return models.RetryRun{Operation: "package", Stage: request.Stage, Attempt: attempt, RetryOfRunID: previous.runID, Package: &run}, nil

	case "transfer":
		if previous.transferRequest == nil || previous.transferRun == nil {
			return models.RetryRun{}, errors.New("transfer retry data is unavailable")
		}
		settings, err := a.config.DaliConfig()
		if err != nil {
			return models.RetryRun{}, err
		}
		runRequest := cloneTransferRequest(*previous.transferRequest)
		if len(runRequest.PackagePaths) > 0 && len(ids) > 0 {
			selected := make(map[string]bool, len(ids))
			for _, id := range ids {
				selected[id] = true
			}
			paths := make([]string, 0, len(runRequest.PackagePaths))
			for _, path := range runRequest.PackagePaths {
				if selected[path] {
					paths = append(paths, path)
				}
			}
			runRequest.PackagePaths = paths
		} else {
			runRequest.ComponentIDs = ids
		}
		plan, err := a.transfer.PlanRequest(previous.project, runRequest)
		if err != nil {
			return models.RetryRun{}, err
		}
		if plan.HasMissing {
			return models.RetryRun{}, errors.New("one or more retry packages no longer exist; review the release before retrying")
		}
		run, err := a.transfer.PrepareRun(newID, previous.project, runRequest)
		if err != nil {
			return models.RetryRun{}, err
		}
		ctx, cancel, done := a.beginActiveRunLocked(run.ID)
		go func() {
			finalRun := a.transfer.Execute(ctx, run, previous.project, runRequest, settings, a.eventSink(eventContext))
			summary := models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "transfer", ComponentIDs: runRequest.ComponentIDs, PackagePaths: append([]string(nil), runRequest.PackagePaths...), Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: runRequest.FilenameTemplate, ApprovedPackageNames: runRequest.PackageNames, ReleaseDirectory: plan.ReleaseDirectory, RetryOfRunID: previous.runID, Attempt: attempt, RetryStage: string(request.Stage)}
			a.addTransferSummaryDetails(&summary, finalRun)
			a.recordRun(summary)
			a.rememberRetryContext(retryContext{operation: "transfer", project: previous.project, transferRequest: &runRequest, transferRun: &finalRun, runID: finalRun.ID, attempt: attempt})
			a.finishActiveRun(run.ID, cancel, done)
		}()
		return models.RetryRun{Operation: "transfer", Stage: request.Stage, Attempt: attempt, RetryOfRunID: previous.runID, Transfer: &run}, nil

	case "release", "release-transfer":
		if previous.releaseRun == nil || previous.packageRequest == nil {
			return models.RetryRun{}, errors.New("release retry data is unavailable")
		}
		var settings *models.DaliConfig
		if previous.operation == "release-transfer" {
			currentSettings, err := a.config.DaliConfig()
			if err != nil {
				return models.RetryRun{}, err
			}
			settings = &currentSettings
		}
		runRequest := clonePackageRequest(*previous.packageRequest)
		runRequest.ComponentIDs = ids
		run := cloneReleaseRun(*previous.releaseRun)
		run.ID = newID
		run.Status = models.ReleaseRunStatusRunning
		run.Error = ""
		run.StartTime = time.Now()
		run.EndTime = time.Time{}
		ctx, cancel, done := a.beginActiveRunLocked(run.ID)
		go func() {
			finalRun := a.executeReleaseRetry(ctx, run, previous.project, runRequest, settings, request.Stage, ids, a.eventSink(eventContext))
			a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: previous.operation, ComponentIDs: runRequest.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: runRequest.FilenameTemplate, ApprovedPackageNames: runRequest.PackageNames, ReleaseDirectory: runRequest.ReleaseDirectory, RetryOfRunID: previous.runID, Attempt: attempt, RetryStage: string(request.Stage)})
			a.rememberRetryContext(retryContext{operation: previous.operation, project: previous.project, packageRequest: &runRequest, releaseRun: &finalRun, runID: finalRun.ID, attempt: attempt})
			a.finishActiveRun(run.ID, cancel, done)
		}()
		return models.RetryRun{Operation: previous.operation, Stage: request.Stage, Attempt: attempt, RetryOfRunID: previous.runID, Release: &run}, nil
	default:
		return models.RetryRun{}, fmt.Errorf("operation %q cannot be retried", previous.operation)
	}
}

func (a *App) beginActiveRunLocked(runID string) (context.Context, context.CancelFunc, chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.activeRunID, a.activeCancel, a.activeDone = runID, cancel, done
	return ctx, cancel, done
}

func (a *App) finishActiveRun(runID string, cancel context.CancelFunc, done chan struct{}) {
	cancel()
	a.releaseRun(runID, done)
}

func retryableComponentIDs(ctx retryContext, stage models.RetryStage) []string {
	var selected []string
	if ctx.buildRun != nil {
		for _, item := range ctx.buildRun.Components {
			if item.Selected && (item.Status == models.BuildStatusFailed || item.Status == models.BuildStatusCancelled || strings.Contains(strings.ToLower(item.Message), "previous component")) {
				selected = append(selected, item.ComponentID)
			}
		}
	}
	if ctx.packageRun != nil {
		for _, item := range ctx.packageRun.Components {
			if item.Selected && (item.Status == models.PackageStatusFailed || item.Status == models.PackageStatusCancelled || strings.Contains(strings.ToLower(item.Message), "previous component")) {
				selected = append(selected, item.ComponentID)
			}
		}
	}
	if ctx.transferRun != nil {
		for _, item := range ctx.transferRun.Components {
			if item.Selected && (item.Status == models.TransferStatusFailed || item.Status == models.TransferStatusCancelled || strings.Contains(strings.ToLower(item.Message), "previous component")) {
				selected = append(selected, item.ComponentID)
			}
		}
	}
	if ctx.releaseRun != nil {
		for _, item := range ctx.releaseRun.Components {
			if !item.Selected {
				continue
			}
			failed := false
			switch stage {
			case models.RetryStageBuild:
				failed = item.BuildStatus != models.BuildStatusSuccess
			case models.RetryStagePackage:
				failed = item.PackageStatus != models.PackageStatusSuccess
			case models.RetryStageTransfer:
				failed = item.TransferStatus != models.TransferStatusSuccess
			}
			if failed {
				selected = append(selected, item.ComponentID)
			}
		}
	}
	return uniqueIDs(selected)
}

func uniqueIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func clonePackageRequest(request models.PackageRequest) models.PackageRequest {
	if request.PackageNames != nil {
		request.PackageNames = mapsClone(request.PackageNames)
	}
	request.ComponentIDs = append([]string(nil), request.ComponentIDs...)
	return request
}

func cloneTransferRequest(request models.TransferRequest) models.TransferRequest {
	if request.PackageNames != nil {
		request.PackageNames = mapsClone(request.PackageNames)
	}
	request.ComponentIDs = append([]string(nil), request.ComponentIDs...)
	request.PackagePaths = append([]string(nil), request.PackagePaths...)
	return request
}

func mapsClone(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneReleaseRun(run models.ReleaseRun) models.ReleaseRun {
	run.Components = append([]models.ReleaseComponentState(nil), run.Components...)
	return run
}

func (a *App) recordRetryBuild(run models.BuildRun, request models.BuildRequest, previous retryContext, stage models.RetryStage, attempt int) {
	a.recordRun(models.RunSummary{RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Environment: run.Environment, Operation: "build", ComponentIDs: request.ComponentIDs, StartTime: run.StartTime, EndTime: run.EndTime, Status: string(run.Status), ErrorSummary: run.Error, RetryOfRunID: previous.runID, Attempt: attempt, RetryStage: string(stage)})
}

func (a *App) executeReleaseRetry(ctx context.Context, run models.ReleaseRun, project models.Project, request models.PackageRequest, settings *models.DaliConfig, stage models.RetryStage, ids []string, emit func(models.BuildEvent)) models.ReleaseRun {
	_ = stage
	request.Version = run.Version
	plan, err := a.packager.PlanRequest(project, request)
	if err != nil {
		run.Status = models.ReleaseRunStatusFailed
		run.Error = err.Error()
		return finishReleaseRetry(run, emit)
	}
	components := make(map[string]models.Component, len(project.Components))
	items := make(map[string]models.PackagePlanItem, len(plan.Components))
	for _, component := range project.Components {
		components[component.ID] = component
	}
	for _, item := range plan.Components {
		items[item.ComponentID] = item
	}
	target := make(map[string]bool, len(ids))
	for _, id := range ids {
		target[id] = true
	}
	emitReleaseRetry(emit, models.BuildEvent{Type: build.EventRunStarted, Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Version: run.Version, Environment: run.Environment, Timestamp: time.Now()})

	for i := range run.Components {
		state := &run.Components[i]
		if !target[state.ComponentID] || !state.Selected {
			continue
		}
		component := components[state.ComponentID]
		buildResult := state.BuildResult
		if buildResult == nil || state.BuildStatus != models.BuildStatusSuccess {
			state.BuildStatus = models.BuildStatusBuilding
			state.BuildMessage = "Building..."
			emitReleaseRetryState(emit, run, *state)
			buildRun := models.BuildRun{ID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Environment: run.Environment}
			result := a.buildService.BuildComponent(ctx, buildRun, project, component, func(event models.BuildEvent) {
				event.Phase = "build"
				event.Version = run.Version
				emitReleaseRetry(emit, event)
			})
			state.BuildResult = &result
			state.BuildStatus = result.Status
			state.BuildMessage = result.Error
			if state.BuildMessage == "" {
				state.BuildMessage = strings.Title(string(result.Status))
			}
			if !result.Success {
				run.Status = releaseStatusForBuild(result.Status)
				run.Error = fmt.Sprintf("%s failed to build.", component.Name)
				emitReleaseRetryState(emit, run, *state)
				break
			}
		}
		if !component.Package.Enabled {
			state.PackageStatus = models.PackageStatusSkipped
			state.PackageMessage = "Packaging disabled"
			state.TransferStatus = models.TransferStatusSkipped
			state.TransferMessage = "Packaging disabled"
			emitReleaseRetryState(emit, run, *state)
			continue
		}
		item := items[component.ID]
		if state.PackageResult == nil || state.PackageStatus != models.PackageStatusSuccess {
			state.PackageStatus = models.PackageStatusPackaging
			state.PackageMessage = "Packaging..."
			emitReleaseRetryState(emit, run, *state)
			result := a.packager.PackageOneWithMetadata(ctx, run.ID, project, component, state.BuildResult.OutputPath, item.PackagePath, request.Overwrite, item.FilenameTemplate, item.ResolvedFilename, run.Version)
			state.PackageResult = &result
			state.PackageStatus = result.Status
			state.PackageMessage = result.Error
			if state.PackageMessage == "" {
				state.PackageMessage = strings.Title(string(result.Status))
			}
			if !result.Success {
				run.Status = releaseStatusForPackage(result.Status)
				run.Error = fmt.Sprintf("%s failed to package.", component.Name)
				emitReleaseRetryState(emit, run, *state)
				break
			}
		}
		if settings == nil {
			continue
		}
		state.TransferStatus = models.TransferStatusSending
		state.TransferMessage = "Sending..."
		emitReleaseRetryState(emit, run, *state)
		result := a.transfer.SendOneWithMetadata(ctx, run.ID, project, component, state.PackageResult.PackagePath, item.FilenameTemplate, item.ResolvedFilename, run.Version, *settings, func(event models.BuildEvent) {
			event.Phase = transfer.Phase
			event.Version = run.Version
			emitReleaseRetry(emit, event)
		})
		state.TransferResult = &result
		state.TransferStatus = result.Status
		state.TransferMessage = result.Error
		if state.TransferMessage == "" {
			state.TransferMessage = strings.Title(string(result.Status))
		}
		emitReleaseRetryState(emit, run, *state)
		if !result.Success {
			run.Status = releaseStatusForTransfer(result.Status)
			run.Error = fmt.Sprintf("%s failed to send.", component.Name)
			break
		}
	}
	if run.Status == models.ReleaseRunStatusRunning {
		run.Status = models.ReleaseRunStatusCompleted
	}
	return finishReleaseRetry(run, emit)
}

func finishReleaseRetry(run models.ReleaseRun, emit func(models.BuildEvent)) models.ReleaseRun {
	run.EndTime = time.Now()
	emitReleaseRetry(emit, models.BuildEvent{Type: "release_run_finished", Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Version: run.Version, Environment: run.Environment, RunStatus: models.BuildRunStatus(run.Status), Error: run.Error, Timestamp: run.EndTime})
	return run
}

func emitReleaseRetry(emit func(models.BuildEvent), event models.BuildEvent) {
	if emit != nil {
		emit(event)
	}
}

func emitReleaseRetryState(emit func(models.BuildEvent), run models.ReleaseRun, state models.ReleaseComponentState) {
	emitReleaseRetry(emit, models.BuildEvent{Type: "release_component_state", Phase: "release", RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, ComponentID: state.ComponentID, ComponentName: state.ComponentName, Version: run.Version, Environment: run.Environment, Status: state.BuildStatus, PackageStatus: state.PackageStatus, Result: state.BuildResult, PackageResult: state.PackageResult, TransferStatus: state.TransferStatus, TransferResult: state.TransferResult, Error: state.BuildMessage + " | " + state.PackageMessage + " | " + state.TransferMessage, Timestamp: time.Now()})
}

func releaseStatusForBuild(status models.BuildStatus) models.ReleaseRunStatus {
	if status == models.BuildStatusCancelled {
		return models.ReleaseRunStatusCancelled
	}
	return models.ReleaseRunStatusFailed
}
func releaseStatusForPackage(status models.PackageStatus) models.ReleaseRunStatus {
	if status == models.PackageStatusCancelled {
		return models.ReleaseRunStatusCancelled
	}
	return models.ReleaseRunStatusFailed
}
func releaseStatusForTransfer(status models.TransferStatus) models.ReleaseRunStatus {
	if status == models.TransferStatusCancelled {
		return models.ReleaseRunStatusCancelled
	}
	return models.ReleaseRunStatusFailed
}
