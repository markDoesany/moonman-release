package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"release-launcher/backend/build"
	"release-launcher/backend/config"
	"release-launcher/backend/logging"
	"release-launcher/backend/models"
	"release-launcher/backend/packaging"
	"release-launcher/backend/pipeline"
	"release-launcher/backend/transfer"
)

// App is the small Wails-facing facade. Domain logic lives in backend services.
type App struct {
	config       *config.Service
	logger       *logging.Logger
	buildService *build.Service
	packager     *packaging.Service
	transfer     *transfer.Service
	pipeline     *pipeline.Service
	activity     *logging.JSONLWriter
	loadErr      error

	buildMu      sync.Mutex
	runtimeCtx   context.Context
	activeRunID  string
	activeCancel context.CancelFunc
	activeDone   chan struct{}
	runSequence  atomic.Uint64
}

func New() (*App, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	logger := logging.New(paths.LogFile)
	service := config.NewService(paths, logger)
	history := logging.NewJSONLWriter(paths.BuildLogFile)
	packagingHistory := logging.NewJSONLWriter(paths.PackageLogFile)
	transferHistory := logging.NewJSONLWriter(paths.TransferLogFile)
	activityHistory := logging.NewJSONLWriter(paths.ActivityLogFile)
	packager := packaging.NewService(logger, packagingHistory, paths.ReleaseDir)
	transferService := transfer.NewService(logger, transferHistory, packager)
	builder := build.NewService(logger, history)
	return &App{
		config:       service,
		logger:       logger,
		buildService: builder,
		packager:     packager,
		transfer:     transferService,
		pipeline:     pipeline.NewService(builder, packager, transferService),
		activity:     activityHistory,
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.buildMu.Lock()
	a.runtimeCtx = ctx
	a.buildMu.Unlock()
	a.logger.Info("Application started")
	if err := a.config.Load(); err != nil {
		a.loadErr = err
	}
}

func (a *App) Shutdown(_ context.Context) {
	a.buildMu.Lock()
	cancel := a.activeCancel
	done := a.activeDone
	a.buildMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	a.logger.Info("Application stopped")
}

// GetProjects returns every configured project.
func (a *App) GetProjects() ([]models.Project, error) {
	if a.loadErr != nil {
		return nil, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.Projects()
}

// GetProject returns one project by its stable ID.
func (a *App) GetProject(id string) (models.Project, error) {
	if a.loadErr != nil {
		return models.Project{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.Project(id)
}

// GetRecentRuns returns durable release activity, newest first.
func (a *App) GetRecentRuns(projectID string, limit int) ([]models.RunSummary, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(paths.ActivityLogFile)
	if errors.Is(err, os.ErrNotExist) {
		return []models.RunSummary{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read activity history: %w", err)
	}
	lines := strings.Split(string(contents), "\n")
	result := make([]models.RunSummary, 0)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var summary models.RunSummary
		if err := json.Unmarshal([]byte(line), &summary); err != nil {
			continue
		}
		if strings.TrimSpace(projectID) != "" && summary.ProjectID != projectID {
			continue
		}
		result = append(result, summary)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

// SaveProject creates or updates a project and returns the canonical saved value.
func (a *App) SaveProject(project models.Project) (models.Project, error) {
	if a.loadErr != nil {
		return models.Project{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	if err := a.ensureConfigEditable(); err != nil {
		return models.Project{}, err
	}
	return a.config.SaveProject(project)
}

// DeleteProject deletes a project and all of its component definitions.
func (a *App) DeleteProject(id string) error {
	if a.loadErr != nil {
		return fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	if err := a.ensureConfigEditable(); err != nil {
		return err
	}
	return a.config.DeleteProject(id)
}

// ValidateProject validates a project before it is saved.
func (a *App) ValidateProject(project models.Project) []models.ValidationIssue {
	return a.config.Validate([]models.Project{project})
}

// GetDaliConfig returns the global Dali transfer settings.
func (a *App) GetDaliConfig() (models.DaliConfig, error) {
	if a.loadErr != nil {
		return models.DaliConfig{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.DaliConfig()
}

// SaveDaliConfig persists the global Dali transfer settings.
func (a *App) SaveDaliConfig(settings models.DaliConfig) (models.DaliConfig, error) {
	if a.loadErr != nil {
		return models.DaliConfig{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	if err := a.ensureConfigEditable(); err != nil {
		return models.DaliConfig{}, err
	}
	return a.config.SaveDaliConfig(settings)
}

// PickDirectory opens the native directory picker. An empty result means the
// user cancelled and is intentionally not an error.
func (a *App) PickDirectory(initialPath string) (string, error) {
	ctx := a.runtimeCtx
	if ctx == nil {
		return "", errors.New("native dialogs are unavailable before application startup")
	}
	initialPath = dialogDirectory(initialPath)
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "Select directory", DefaultDirectory: initialPath})
}

// PickFile opens the native file picker. An empty result means the user
// cancelled and leaves the current text field unchanged.
func (a *App) PickFile(initialPath string) (string, error) {
	ctx := a.runtimeCtx
	if ctx == nil {
		return "", errors.New("native dialogs are unavailable before application startup")
	}
	options := runtime.OpenDialogOptions{Title: "Select file"}
	if info, err := os.Stat(initialPath); err == nil && !info.IsDir() {
		options.DefaultDirectory = filepath.Dir(initialPath)
		options.DefaultFilename = filepath.Base(initialPath)
	} else {
		options.DefaultDirectory = dialogDirectory(initialPath)
	}
	return runtime.OpenFileDialog(ctx, options)
}

// OpenReleaseFolder opens a local release directory in the host file manager.
func (a *App) OpenReleaseFolder(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return errors.New("release folder is empty")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return fmt.Errorf("release folder does not exist: %s", path)
	}
	if a.runtimeCtx == nil {
		return errors.New("native folder actions are unavailable before application startup")
	}
	runtime.BrowserOpenURL(a.runtimeCtx, "file:///"+filepath.ToSlash(path))
	return nil
}

func dialogDirectory(value string) string {
	value = filepath.Clean(value)
	if info, err := os.Stat(value); err == nil {
		if info.IsDir() {
			return value
		}
		return filepath.Dir(value)
	}
	if value != "." && filepath.Ext(value) != "" {
		return filepath.Dir(value)
	}
	return ""
}

// GetPackagePlan calculates output paths and overwrite conflicts without writing files.
func (a *App) GetPackagePlan(request models.PackageRequest) (models.PackagePlan, error) {
	if a.loadErr != nil {
		return models.PackagePlan{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	if a.BuildRunning() {
		return models.PackagePlan{}, errors.New("a build or packaging operation is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.PackagePlan{}, err
	}
	return a.packager.PlanRequest(project, request)
}

// StartPackage packages already-built output directories sequentially.
func (a *App) StartPackage(request models.PackageRequest) (models.PackageRun, error) {
	if a.loadErr != nil {
		return models.PackageRun{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel != nil {
		return models.PackageRun{}, errors.New("a build or packaging operation is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.PackageRun{}, err
	}
	plan, err := a.packager.PlanRequest(project, request)
	if err != nil {
		return models.PackageRun{}, err
	}
	if plan.HasConflicts && !request.Overwrite {
		return models.PackageRun{}, errors.New("one or more packages already exist; confirm replacement before packaging")
	}
	runID := a.nextRunID()
	request.Version = plan.Version
	run, err := a.packager.PrepareRunRequest(runID, project, request)
	if err != nil {
		return models.PackageRun{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.activeRunID, a.activeCancel, a.activeDone = run.ID, cancel, done
	eventContext := a.runtimeCtx
	a.logger.Info(fmt.Sprintf("Packaging started: %s", project.Name))
	go func() {
		finalRun := a.packager.ExecuteRequest(ctx, run, project, request, a.eventSink(eventContext))
		if finalRun.Status == models.PackageRunStatusCompleted {
			a.logger.Info(fmt.Sprintf("Packaging completed: %s", project.Name))
		} else if finalRun.Status == models.PackageRunStatusCancelled {
			a.logger.Info(fmt.Sprintf("Packaging cancelled: %s", project.Name))
		} else {
			a.logger.Error(fmt.Sprintf("Packaging failed: %s", project.Name))
		}
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "package", ComponentIDs: componentIDsFromPackageRequest(request), Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory})
		a.releaseRun(run.ID, done)
	}()
	return run, nil
}

// GetTransferPlan calculates the package files that are ready to send.
func (a *App) GetTransferPlan(request models.TransferRequest) (models.TransferPlan, error) {
	if a.loadErr != nil {
		return models.TransferPlan{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	if a.BuildRunning() {
		return models.TransferPlan{}, errors.New("a build or packaging operation is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.TransferPlan{}, err
	}
	return a.transfer.PlanRequest(project, request)
}

// StartTransfer sends existing release packages sequentially through Dali.
func (a *App) StartTransfer(request models.TransferRequest) (models.TransferRun, error) {
	if a.loadErr != nil {
		return models.TransferRun{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel != nil {
		return models.TransferRun{}, errors.New("a build, packaging, or transfer operation is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.TransferRun{}, err
	}
	plan, err := a.transfer.PlanRequest(project, request)
	if err != nil {
		return models.TransferRun{}, err
	}
	if plan.HasMissing {
		return models.TransferRun{}, errors.New("one or more selected packages do not exist")
	}
	settings, err := a.config.DaliConfig()
	if err != nil {
		return models.TransferRun{}, err
	}
	runID := a.nextRunID()
	run, err := a.transfer.PrepareRun(runID, project, request)
	if err != nil {
		return models.TransferRun{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.activeRunID, a.activeCancel, a.activeDone = run.ID, cancel, done
	eventContext := a.runtimeCtx
	a.logger.Info(fmt.Sprintf("Dali transfer started: %s", project.Name))
	go func() {
		finalRun := a.transfer.Execute(ctx, run, project, request, settings, a.eventSink(eventContext))
		if finalRun.Status == models.TransferRunStatusCompleted {
			a.logger.Info(fmt.Sprintf("Dali transfer completed: %s", project.Name))
		} else if finalRun.Status == models.TransferRunStatusCancelled {
			a.logger.Info(fmt.Sprintf("Dali transfer cancelled: %s", project.Name))
		} else {
			a.logger.Error(fmt.Sprintf("Dali transfer failed: %s", project.Name))
		}
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "transfer", ComponentIDs: request.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory})
		a.releaseRun(run.ID, done)
	}()
	return run, nil
}

// StartBuildAndPackage runs each selected component through build then packaging.
func (a *App) StartBuildAndPackage(request models.PackageRequest) (models.ReleaseRun, error) {
	if a.loadErr != nil {
		return models.ReleaseRun{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel != nil {
		return models.ReleaseRun{}, errors.New("a build or packaging operation is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	plan, err := a.packager.PlanRequest(project, request)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	if plan.HasConflicts && !request.Overwrite {
		return models.ReleaseRun{}, errors.New("one or more packages already exist; confirm replacement before building")
	}
	runID := a.nextRunID()
	run, err := a.pipeline.PrepareRun(runID, project, request)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.activeRunID, a.activeCancel, a.activeDone = run.ID, cancel, done
	eventContext := a.runtimeCtx
	a.logger.Info(fmt.Sprintf("Build and package started: %s", project.Name))
	go func() {
		finalRun := a.pipeline.Execute(ctx, run, project, request, a.eventSink(eventContext))
		if finalRun.Status == models.ReleaseRunStatusCompleted {
			a.logger.Info(fmt.Sprintf("Build and package completed: %s", project.Name))
		} else if finalRun.Status == models.ReleaseRunStatusCancelled {
			a.logger.Info(fmt.Sprintf("Build and package cancelled: %s", project.Name))
		} else {
			a.logger.Error(fmt.Sprintf("Build and package failed: %s", project.Name))
		}
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "build-package", ComponentIDs: request.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory})
		a.releaseRun(run.ID, done)
	}()
	return run, nil
}

// StartBuildPackageAndSend runs build, package, and Dali transfer sequentially.
func (a *App) StartBuildPackageAndSend(request models.PackageRequest) (models.ReleaseRun, error) {
	if a.loadErr != nil {
		return models.ReleaseRun{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel != nil {
		return models.ReleaseRun{}, errors.New("a build, packaging, or transfer operation is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	plan, err := a.packager.PlanRequest(project, request)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	if plan.HasConflicts && !request.Overwrite {
		return models.ReleaseRun{}, errors.New("one or more packages already exist; confirm replacement before building")
	}
	settings, err := a.config.DaliConfig()
	if err != nil {
		return models.ReleaseRun{}, err
	}
	runID := a.nextRunID()
	run, err := a.pipeline.PrepareRun(runID, project, request)
	if err != nil {
		return models.ReleaseRun{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.activeRunID, a.activeCancel, a.activeDone = run.ID, cancel, done
	eventContext := a.runtimeCtx
	a.logger.Info(fmt.Sprintf("Build, package, and Dali transfer started: %s", project.Name))
	go func() {
		finalRun := a.pipeline.ExecuteBuildPackageAndSend(ctx, run, project, request, settings, a.eventSink(eventContext))
		if finalRun.Status == models.ReleaseRunStatusCompleted {
			a.logger.Info(fmt.Sprintf("Build, package, and Dali transfer completed: %s", project.Name))
		} else if finalRun.Status == models.ReleaseRunStatusCancelled {
			a.logger.Info(fmt.Sprintf("Build, package, and Dali transfer cancelled: %s", project.Name))
		} else {
			a.logger.Error(fmt.Sprintf("Build, package, and Dali transfer failed: %s", project.Name))
		}
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "build-package-send", ComponentIDs: request.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory})
		a.releaseRun(run.ID, done)
	}()
	return run, nil
}

// StartBuild starts a sequential build and returns its initial state immediately.
func (a *App) StartBuild(request models.BuildRequest) (models.BuildRun, error) {
	if a.loadErr != nil {
		return models.BuildRun{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}

	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel != nil {
		return models.BuildRun{}, errors.New("a build is already running")
	}
	project, err := a.config.Project(request.ProjectID)
	if err != nil {
		return models.BuildRun{}, err
	}
	runID := fmt.Sprintf("build-%d-%d", time.Now().UnixNano(), a.runSequence.Add(1))
	run, err := a.buildService.PrepareRunForEnvironment(runID, project, request.ComponentIDs, request.Environment)
	if err != nil {
		return models.BuildRun{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.activeRunID = run.ID
	a.activeCancel = cancel
	a.activeDone = done
	eventContext := a.runtimeCtx
	a.logger.Info(fmt.Sprintf("Build started: %s", project.Name))

	go func() {
		finalRun := a.buildService.Execute(ctx, run, project, func(event models.BuildEvent) {
			if eventContext != nil {
				runtime.EventsEmit(eventContext, build.EventName, event)
			}
		})
		switch finalRun.Status {
		case models.BuildRunStatusCompleted:
			a.logger.Info(fmt.Sprintf("Build completed: %s", project.Name))
		case models.BuildRunStatusCancelled:
			a.logger.Info(fmt.Sprintf("Build cancelled: %s", project.Name))
		default:
			a.logger.Error(fmt.Sprintf("Build failed: %s", project.Name))
		}
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "build", ComponentIDs: request.ComponentIDs, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error})
		a.buildMu.Lock()
		if a.activeRunID == run.ID {
			a.activeRunID = ""
			a.activeCancel = nil
			a.activeDone = nil
			close(done)
		}
		a.buildMu.Unlock()
	}()

	return run, nil
}

func (a *App) nextRunID() string {
	return fmt.Sprintf("run-%d-%d", time.Now().UnixNano(), a.runSequence.Add(1))
}

func (a *App) recordRun(summary models.RunSummary) {
	if a.activity == nil {
		return
	}
	if err := a.activity.Append(summary); err != nil && a.logger != nil {
		a.logger.Error("Activity history could not be written: " + err.Error())
	}
}

func componentIDsFromPackageRequest(request models.PackageRequest) []string {
	return append([]string(nil), request.ComponentIDs...)
}

func (a *App) eventSink(eventContext context.Context) func(models.BuildEvent) {
	return func(event models.BuildEvent) {
		if eventContext != nil {
			runtime.EventsEmit(eventContext, build.EventName, event)
		}
	}
}

func (a *App) releaseRun(runID string, done chan struct{}) {
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeRunID == runID {
		a.activeRunID = ""
		a.activeCancel = nil
		a.activeDone = nil
		close(done)
	}
}

// CancelBuild requests cancellation of the active build.
func (a *App) CancelBuild(runID string) error {
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.activeCancel == nil {
		return errors.New("no build is running")
	}
	if runID != "" && runID != a.activeRunID {
		return fmt.Errorf("build %q is not running", runID)
	}
	a.activeCancel()
	return nil
}

// BuildRunning reports whether the application currently owns an active build.
func (a *App) BuildRunning() bool {
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	return a.activeCancel != nil
}

func (a *App) ensureConfigEditable() error {
	if a.BuildRunning() {
		return errors.New("configuration cannot be changed while a build is running")
	}
	return nil
}
