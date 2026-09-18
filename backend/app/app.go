package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	retryMu      sync.Mutex
	retryRuns    map[string]retryContext
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
		retryRuns:    make(map[string]retryContext),
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
	page := models.ActivityQuery{ProjectID: projectID, Page: 1, PageSize: limit}
	if limit <= 0 {
		page.PageSize = 100
	}
	result, err := a.GetRecentRunsPage(page)
	if err != nil {
		return nil, err
	}
	return result.Runs, nil
}

// GetRecentRunsPage returns filtered activity without requiring the frontend to load all history.
func (a *App) GetRecentRunsPage(query models.ActivityQuery) (models.ActivityPage, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return models.ActivityPage{}, err
	}
	contents, err := os.ReadFile(paths.ActivityLogFile)
	if errors.Is(err, os.ErrNotExist) {
		query = normalizeActivityQuery(query)
		return models.ActivityPage{Runs: []models.RunSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	if err != nil {
		return models.ActivityPage{}, fmt.Errorf("read activity history: %w", err)
	}
	query = normalizeActivityQuery(query)
	lines := strings.Split(string(contents), "\n")
	all := make([]models.RunSummary, 0)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var summary models.RunSummary
		if err := json.Unmarshal([]byte(line), &summary); err != nil {
			continue
		}
		if query.ProjectID != "" && summary.ProjectID != query.ProjectID {
			continue
		}
		if query.Environment != "" && query.Environment != "all" && !strings.EqualFold(summary.Environment, query.Environment) {
			continue
		}
		if query.Status != "" && query.Status != "all" && !strings.EqualFold(summary.Status, query.Status) {
			continue
		}
		if query.Search != "" && !activityMatches(summary, query.Search) {
			continue
		}
		all = append(all, summary)
	}
	pageCount := 0
	if len(all) > 0 {
		pageCount = (len(all) + query.PageSize - 1) / query.PageSize
	}
	start := (query.Page - 1) * query.PageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + query.PageSize
	if end > len(all) {
		end = len(all)
	}
	runs := all[start:end]
	if runs == nil {
		runs = []models.RunSummary{}
	}
	return models.ActivityPage{Runs: runs, Page: query.Page, PageSize: query.PageSize, Total: len(all), TotalPages: pageCount}, nil
}

func normalizeActivityQuery(query models.ActivityQuery) models.ActivityQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.PageSize > 500 {
		query.PageSize = 500
	}
	query.ProjectID = strings.TrimSpace(query.ProjectID)
	query.Environment = strings.TrimSpace(query.Environment)
	query.Status = strings.TrimSpace(query.Status)
	query.Search = strings.ToLower(strings.TrimSpace(query.Search))
	return query
}

func activityMatches(summary models.RunSummary, query string) bool {
	values := []string{summary.RunID, summary.ProjectID, summary.ProjectName, summary.Environment, summary.Operation, summary.Version, summary.ErrorSummary, summary.RetryOfRunID, summary.RetryStage, summary.Command, fmt.Sprint(summary.Attempt)}
	values = append(values, summary.ComponentIDs...)
	values = append(values, summary.ComponentNames...)
	values = append(values, summary.PackagePaths...)
	for key, command := range summary.DaliCommands {
		values = append(values, key, command)
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
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

// CheckDaliAvailability resolves the configured executable without starting it.
func (a *App) CheckDaliAvailability() (models.DaliAvailability, error) {
	if a.loadErr != nil {
		return models.DaliAvailability{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	settings, err := a.config.DaliConfig()
	if err != nil {
		return models.DaliAvailability{}, err
	}
	return a.transfer.CheckDaliAvailability(settings), nil
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

// PickFiles opens a native multi-select picker limited to ZIP files.
func (a *App) PickFiles(initialPath string) ([]string, error) {
	ctx := a.runtimeCtx
	if ctx == nil {
		return nil, errors.New("native dialogs are unavailable before application startup")
	}
	options := runtime.OpenDialogOptions{Title: "Select ZIP packages", Filters: []runtime.FileFilter{{DisplayName: "ZIP packages", Pattern: "*.zip"}}}
	options.DefaultDirectory = dialogDirectory(initialPath)
	paths, err := runtime.OpenMultipleFilesDialog(ctx, options)
	if err != nil {
		return nil, err
	}
	return normalizeZipSelections(paths)
}

// ListZipFiles lists only regular, top-level ZIP files in a directory.
func (a *App) ListZipFiles(directory string) ([]string, error) {
	directory = filepath.Clean(strings.TrimSpace(directory))
	if directory == "." || directory == "" {
		return nil, errors.New("package folder is empty")
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("package folder does not exist: %s", directory)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read package folder: %w", err)
	}
	result := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".zip") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		fileInfo, statErr := os.Stat(path)
		if statErr == nil && fileInfo.Mode().IsRegular() {
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result, nil
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
	return openReleaseFolderNative(path)
}

// OpenExternalURL opens a documentation or release URL in the default browser.
func (a *App) OpenExternalURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" || !(strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) {
		return errors.New("external URL must use http or https")
	}
	if a.runtimeCtx == nil {
		return errors.New("external links are unavailable before application startup")
	}
	runtime.BrowserOpenURL(a.runtimeCtx, url)
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

func normalizeZipSelections(paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, value := range paths {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		absolute, err := filepath.Abs(value)
		if err != nil {
			return nil, fmt.Errorf("resolve selected package: %w", err)
		}
		absolute = filepath.Clean(absolute)
		info, err := os.Stat(absolute)
		if err != nil || !info.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(absolute), ".zip") {
			return nil, fmt.Errorf("selected file is not a ZIP package: %s", absolute)
		}
		if !seen[absolute] {
			seen[absolute] = true
			result = append(result, absolute)
		}
	}
	return result, nil
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
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "package", ComponentIDs: componentIDsFromPackageRequest(request), Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory, Attempt: 1})
		retryRequest := clonePackageRequest(request)
		a.rememberRetryContext(retryContext{operation: "package", project: project, packageRequest: &retryRequest, packageRun: &finalRun, runID: finalRun.ID, attempt: 1})
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
		summary := models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "transfer", ComponentIDs: request.ComponentIDs, PackagePaths: append([]string(nil), request.PackagePaths...), Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory, Attempt: 1}
		a.addTransferSummaryDetails(&summary, finalRun)
		a.recordRun(summary)
		retryRequest := cloneTransferRequest(request)
		a.rememberRetryContext(retryContext{operation: "transfer", project: project, transferRequest: &retryRequest, transferRun: &finalRun, runID: finalRun.ID, attempt: 1})
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
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "build-package", ComponentIDs: request.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory, Attempt: 1})
		retryRequest := clonePackageRequest(request)
		a.rememberRetryContext(retryContext{operation: "release", project: project, packageRequest: &retryRequest, releaseRun: &finalRun, runID: finalRun.ID, attempt: 1})
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
		summary := models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "build-package-send", ComponentIDs: request.ComponentIDs, Version: finalRun.Version, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, FilenameTemplate: request.FilenameTemplate, ApprovedPackageNames: request.PackageNames, ReleaseDirectory: plan.ReleaseDirectory, Attempt: 1}
		a.addReleaseTransferSummaryDetails(&summary, finalRun)
		a.recordRun(summary)
		retryRequest := clonePackageRequest(request)
		a.rememberRetryContext(retryContext{operation: "release-transfer", project: project, packageRequest: &retryRequest, releaseRun: &finalRun, runID: finalRun.ID, attempt: 1})
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
		a.recordRun(models.RunSummary{RunID: finalRun.ID, ProjectID: finalRun.ProjectID, ProjectName: finalRun.ProjectName, Environment: finalRun.Environment, Operation: "build", ComponentIDs: request.ComponentIDs, StartTime: finalRun.StartTime, EndTime: finalRun.EndTime, Status: string(finalRun.Status), ErrorSummary: finalRun.Error, Attempt: 1})
		retryRequest := models.BuildRequest{ProjectID: request.ProjectID, ComponentIDs: append([]string(nil), request.ComponentIDs...), Environment: request.Environment}
		a.rememberRetryContext(retryContext{operation: "build", project: project, buildRequest: &retryRequest, buildRun: &finalRun, runID: finalRun.ID, attempt: 1})
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
	if len(summary.ComponentNames) == 0 {
		if a.config != nil {
			if project, err := a.config.Project(summary.ProjectID); err == nil {
				for _, id := range summary.ComponentIDs {
					for _, component := range project.Components {
						if component.ID == id {
							summary.ComponentNames = append(summary.ComponentNames, component.Name)
							break
						}
					}
				}
			}
		}
		for _, path := range summary.PackagePaths {
			summary.ComponentNames = append(summary.ComponentNames, filepath.Base(path))
		}
	}
	if err := a.activity.Append(summary); err != nil && a.logger != nil {
		a.logger.Error("Activity history could not be written: " + err.Error())
	}
}

func (a *App) addTransferSummaryDetails(summary *models.RunSummary, run models.TransferRun) {
	commands := make(map[string]string)
	for _, state := range run.Components {
		if state.Result == nil {
			continue
		}
		if summary.Command == "" {
			summary.Command = state.Result.Command
		}
		key := state.ComponentID
		if key == "" {
			key = state.ComponentName
		}
		if state.Result.Command != "" {
			commands[key] = state.Result.Command
		}
		if state.ComponentName != "" {
			summary.ComponentNames = append(summary.ComponentNames, state.ComponentName)
		}
	}
	if len(commands) > 0 {
		summary.DaliCommands = commands
	}
}

func (a *App) addReleaseTransferSummaryDetails(summary *models.RunSummary, run models.ReleaseRun) {
	commands := make(map[string]string)
	for _, state := range run.Components {
		if state.TransferResult == nil {
			continue
		}
		if summary.Command == "" {
			summary.Command = state.TransferResult.Command
		}
		if state.TransferResult.Command != "" {
			commands[state.ComponentID] = state.TransferResult.Command
		}
	}
	if len(commands) > 0 {
		summary.DaliCommands = commands
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
