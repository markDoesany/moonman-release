package packaging

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
	"release-launcher/backend/pathutil"
)

const (
	Phase = "package"

	EventRunStarted        = "package_run_started"
	EventComponentStarted  = "package_started"
	EventComponentFinished = "package_finished"
	EventRunFinished       = "package_run_finished"

	defaultVersionFormat = "20060102-150405"
)

var invalidPathSegment = regexp.MustCompile(`[<>:"/\\|?*]`)
var validVersion = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var tokenPattern = regexp.MustCompile(`\{([a-z]+)\}`)

const DefaultFilenameTemplate = models.DefaultPackageFilenameTemplate

// EventSink receives packaging events without coupling this service to Wails.
type EventSink func(models.BuildEvent)

// Service creates and validates local ZIP packages.
type Service struct {
	logger      *logging.Logger
	history     *logging.JSONLWriter
	releaseRoot string
}

func NewService(logger *logging.Logger, history *logging.JSONLWriter, releaseRoot string) *Service {
	return &Service{logger: logger, history: history, releaseRoot: releaseRoot}
}

// Plan calculates package paths and detects overwrite conflicts without writing files.
func (s *Service) Plan(project models.Project, componentIDs []string, version string) (models.PackagePlan, error) {
	return s.PlanRequest(project, models.PackageRequest{ProjectID: project.ID, ComponentIDs: componentIDs, Version: version})
}

// PlanRequest resolves a single timestamp for all selected packages. Explicit
// PackageNames are the edits approved in the naming preview.
func (s *Service) PlanRequest(project models.Project, request models.PackageRequest) (models.PackagePlan, error) {
	componentIDs := request.ComponentIDs
	selected := make(map[string]bool, len(componentIDs))
	for _, id := range componentIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			selected[id] = true
		}
	}
	if len(selected) == 0 {
		return models.PackagePlan{}, errors.New("select at least one component to package")
	}
	version, err := normalizeVersion(request.Version)
	if err != nil {
		return models.PackagePlan{}, err
	}

	known := make(map[string]bool, len(project.Components))
	items := make([]models.PackagePlanItem, 0, len(project.Components))
	releaseDirectory := filepath.Join(s.releaseRoot, safePathSegment(project.Name, project.ID), version)
	if requestedDirectory := strings.TrimSpace(request.ReleaseDirectory); requestedDirectory != "" {
		if !filepath.IsAbs(requestedDirectory) {
			return models.PackagePlan{}, errors.New("release directory must be an absolute path")
		}
		releaseDirectory = filepath.Clean(requestedDirectory)
	}
	stamp := time.Now()
	sharedTemplate := strings.TrimSpace(request.FilenameTemplate)
	for _, component := range project.Components {
		known[component.ID] = true
		template := sharedTemplate
		if template == "" {
			template = component.Package.Filename
		}
		resolved, resolveErr := ResolveFilename(template, project.Name, component.Name, version, stamp)
		if override, exists := request.PackageNames[component.ID]; exists {
			resolved = strings.TrimSpace(override)
			resolveErr = nil
		}
		item := models.PackagePlanItem{
			ComponentID:      component.ID,
			ComponentName:    component.Name,
			Selected:         selected[component.ID],
			Enabled:          component.Package.Enabled,
			SourcePath:       sourcePath(component),
			FilenameTemplate: template,
			ResolvedFilename: resolved,
		}
		if item.Selected && item.Enabled {
			if resolveErr != nil {
				return models.PackagePlan{}, fmt.Errorf("component %q: %w", component.Name, resolveErr)
			}
			if err := validateFilename(resolved); err != nil {
				return models.PackagePlan{}, fmt.Errorf("component %q: %w", component.Name, err)
			}
			item.PackagePath = filepath.Join(releaseDirectory, resolved)
			if _, statErr := os.Stat(item.PackagePath); statErr == nil {
				item.Existing = true
			}
		} else if resolveErr == nil {
			item.PackagePath = filepath.Join(releaseDirectory, resolved)
		}
		items = append(items, item)
	}
	for id := range selected {
		if !known[id] {
			return models.PackagePlan{}, fmt.Errorf("component %q is not configured for project %q", id, project.Name)
		}
	}

	conflicts := false
	seen := make(map[string]string)
	for _, item := range items {
		if item.Selected && item.Enabled && item.Existing {
			conflicts = true
		}
		if item.Selected && item.Enabled {
			key := strings.ToLower(item.ResolvedFilename)
			if previous, exists := seen[key]; exists {
				return models.PackagePlan{}, fmt.Errorf("components %q and %q resolve to duplicate package filename %q", previous, item.ComponentName, item.ResolvedFilename)
			}
			seen[key] = item.ComponentName
		}
	}
	return models.PackagePlan{
		ProjectID:        project.ID,
		ProjectName:      project.Name,
		Version:          version,
		ReleaseDirectory: releaseDirectory,
		FilenameTemplate: sharedTemplate,
		Components:       items,
		HasConflicts:     conflicts,
	}, nil
}

// ResolveFilename expands supported template tokens and validates the final
// Windows filename. Static filenames continue to work unchanged.
func ResolveFilename(template, project, component, version string, timestamp time.Time) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", errors.New("package filename or template is required")
	}
	if strings.Contains(template, "{") || strings.Contains(template, "}") {
		matches := tokenPattern.FindAllStringSubmatch(template, -1)
		template = tokenPattern.ReplaceAllStringFunc(template, func(token string) string {
			name := tokenPattern.FindStringSubmatch(token)[1]
			switch name {
			case "project":
				return safeFilenameToken(project)
			case "component":
				return safeFilenameToken(component)
			case "version":
				return safeFilenameToken(version)
			case "date":
				return timestamp.Format("20060102")
			case "time":
				return timestamp.Format("150405")
			case "datetime":
				return timestamp.Format("20060102-150405")
			default:
				return token
			}
		})
		if len(matches) == 0 || strings.Contains(template, "{") || strings.Contains(template, "}") {
			return "", errors.New("package filename contains an unknown or malformed template token")
		}
	}
	if template != strings.TrimRight(template, " .") {
		return "", errors.New("package filename must not end with a space or period")
	}
	if !strings.HasSuffix(strings.ToLower(template), ".zip") {
		template += ".zip"
	}
	if err := validateFilename(template); err != nil {
		return "", err
	}
	return template, nil
}

func safeFilenameToken(value string) string {
	value = invalidPathSegment.ReplaceAllString(strings.TrimSpace(value), "_")
	value = strings.Trim(value, " .")
	if value == "" {
		return "item"
	}
	return value
}

// PrepareRun creates initial state for a package-only run.
func (s *Service) PrepareRun(runID string, project models.Project, componentIDs []string, version string) (models.PackageRun, error) {
	return s.PrepareRunRequest(runID, project, models.PackageRequest{ProjectID: project.ID, ComponentIDs: componentIDs, Version: version})
}

func (s *Service) PrepareRunRequest(runID string, project models.Project, request models.PackageRequest) (models.PackageRun, error) {
	plan, err := s.PlanRequest(project, request)
	if err != nil {
		return models.PackageRun{}, err
	}
	states := make([]models.PackageComponentState, 0, len(plan.Components))
	for _, item := range plan.Components {
		states = append(states, models.PackageComponentState{
			ComponentID:   item.ComponentID,
			ComponentName: item.ComponentName,
			Selected:      item.Selected,
			Status:        models.PackageStatusReady,
			Message:       "Ready",
		})
	}
	return models.PackageRun{
		ID:          runID,
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Environment: project.EffectiveEnvironment(request.Environment),
		Version:     plan.Version,
		Status:      models.PackageRunStatusRunning,
		Components:  states,
		StartTime:   time.Now(),
	}, nil
}

// Execute packages selected components sequentially and stops on the first failure.
func (s *Service) Execute(ctx context.Context, run models.PackageRun, project models.Project, componentIDs []string, overwrite bool, emit EventSink) models.PackageRun {
	return s.ExecuteRequest(ctx, run, project, models.PackageRequest{ProjectID: project.ID, ComponentIDs: componentIDs, Version: run.Version, Overwrite: overwrite}, emit)
}

func (s *Service) ExecuteRequest(ctx context.Context, run models.PackageRun, project models.Project, request models.PackageRequest, emit EventSink) models.PackageRun {
	request.Version = run.Version
	plan, err := s.PlanRequest(project, request)
	if err != nil {
		run.Status = models.PackageRunStatusFailed
		run.Error = err.Error()
		run.EndTime = time.Now()
		return run
	}

	s.emit(emit, models.BuildEvent{
		Type:        EventRunStarted,
		Phase:       Phase,
		RunID:       run.ID,
		ProjectID:   run.ProjectID,
		ProjectName: run.ProjectName,
		Version:     run.Version,
		Environment: run.Environment,
		Timestamp:   time.Now(),
	})

	components := make(map[string]models.Component, len(project.Components))
	items := make(map[string]models.PackagePlanItem, len(plan.Components))
	for _, component := range project.Components {
		components[component.ID] = component
	}
	for _, item := range plan.Components {
		items[item.ComponentID] = item
	}
	for i := range run.Components {
		state := &run.Components[i]
		item := items[state.ComponentID]
		if !state.Selected {
			state.Status = models.PackageStatusSkipped
			state.Message = "Not selected"
			s.emitState(emit, run, *state, EventComponentFinished)
			continue
		}
		if !item.Enabled {
			state.Status = models.PackageStatusSkipped
			state.Message = "Packaging disabled"
			s.emitState(emit, run, *state, EventComponentFinished)
		}
	}

	for i := range run.Components {
		state := &run.Components[i]
		if !state.Selected || state.Status == models.PackageStatusSkipped || run.Status != models.PackageRunStatusRunning {
			continue
		}
		if err := ctx.Err(); err != nil {
			state.Status = models.PackageStatusCancelled
			state.Message = "Packaging cancelled before this component started"
			s.emitState(emit, run, *state, EventComponentFinished)
			skipRemaining(run.Components[i+1:], "Packaging cancelled", emit, run)
			run.Status = models.PackageRunStatusCancelled
			run.Error = "Packaging cancelled"
			break
		}

		component := components[state.ComponentID]
		item := items[state.ComponentID]
		state.Status = models.PackageStatusPackaging
		state.Message = "Packaging..."
		s.emitState(emit, run, *state, EventComponentStarted)
		s.emit(emit, models.BuildEvent{
			Type:          "output",
			Phase:         Phase,
			RunID:         run.ID,
			ProjectID:     run.ProjectID,
			ProjectName:   run.ProjectName,
			ComponentID:   component.ID,
			ComponentName: component.Name,
			Version:       run.Version,
			Environment:   run.Environment,
			Stream:        "system",
			Text:          "Creating: " + item.PackagePath,
			Timestamp:     time.Now(),
		})

		result := s.packageOne(ctx, run.ID, project, component, item.SourcePath, item.PackagePath, request.Overwrite, item.FilenameTemplate, item.ResolvedFilename, run.Version)
		state.Result = &result
		state.Status = result.Status
		state.Message = result.Error
		if state.Message == "" {
			state.Message = statusMessage(string(result.Status))
		}
		s.emitState(emit, run, *state, EventComponentFinished)
		if result.Status == models.PackageStatusCancelled {
			run.Status = models.PackageRunStatusCancelled
			run.Error = "Packaging cancelled"
			skipRemaining(run.Components[i+1:], "Packaging cancelled", emit, run)
			break
		}
		if !result.Success {
			run.Status = models.PackageRunStatusFailed
			run.Error = fmt.Sprintf("%s failed to package.", component.Name)
			skipRemaining(run.Components[i+1:], "Previous component failed", emit, run)
			break
		}
	}

	if run.Status == models.PackageRunStatusRunning {
		run.Status = models.PackageRunStatusCompleted
	}
	run.EndTime = time.Now()
	s.emit(emit, models.BuildEvent{
		Type:           EventRunFinished,
		Phase:          Phase,
		RunID:          run.ID,
		ProjectID:      run.ProjectID,
		ProjectName:    run.ProjectName,
		Version:        run.Version,
		Environment:    run.Environment,
		PackageResults: packageResults(run),
		Error:          run.Error,
		Timestamp:      run.EndTime,
	})
	return run
}

// PackageOne creates and validates one archive from a successful build output.
func (s *Service) PackageOne(ctx context.Context, runID string, project models.Project, component models.Component, source, destination string, overwrite bool) models.PackageResult {
	return s.packageOne(ctx, runID, project, component, source, destination, overwrite, "", filepath.Base(destination), "")
}

func (s *Service) PackageOneWithMetadata(ctx context.Context, runID string, project models.Project, component models.Component, source, destination string, overwrite bool, filenameTemplate, resolvedFilename, version string) models.PackageResult {
	return s.packageOne(ctx, runID, project, component, source, destination, overwrite, filenameTemplate, resolvedFilename, version)
}

func (s *Service) packageOne(ctx context.Context, runID string, project models.Project, component models.Component, source, destination string, overwrite bool, filenameTemplate, resolvedFilename, version string) models.PackageResult {
	started := time.Now()
	result := models.PackageResult{
		ProjectID:        project.ID,
		ProjectName:      project.Name,
		ComponentID:      component.ID,
		ComponentName:    component.Name,
		Status:           models.PackageStatusFailed,
		SourcePath:       source,
		PackagePath:      destination,
		Version:          version,
		FilenameTemplate: filenameTemplate,
		ResolvedFilename: resolvedFilename,
		StartTime:        started,
	}
	finish := func(technical error) models.PackageResult {
		result.EndTime = time.Now()
		result.DurationMs = result.EndTime.Sub(result.StartTime).Milliseconds()
		s.writeRecord(runID, result, technical)
		return result
	}

	if err := ctx.Err(); err != nil {
		result.Status = models.PackageStatusCancelled
		result.Error = "Packaging cancelled."
		return finish(err)
	}
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
		if errors.Is(err, os.ErrNotExist) {
			result.Error = "Build output directory does not exist."
		} else {
			result.Error = "Build output directory could not be accessed."
		}
		if err == nil {
			err = errors.New("configured build output is not a directory")
		}
		return finish(err)
	}
	if err := validateFilename(filepath.Base(destination)); err != nil {
		result.Error = "Package filename is invalid."
		return finish(err)
	}

	if destinationInfo, statErr := os.Stat(destination); statErr == nil {
		if destinationInfo.IsDir() {
			result.Error = "Package path exists and is not a file."
			return finish(errors.New("package destination is a directory"))
		}
		if !overwrite {
			result.Error = "Package already exists. Confirm replace before packaging."
			return finish(errors.New("package destination already exists"))
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		result.Error = "Package destination could not be accessed."
		return finish(statErr)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		result.Error = "Release directory could not be created."
		return finish(err)
	}
	temporary := destination + ".tmp"
	_ = os.Remove(temporary)
	archive, err := os.Create(temporary)
	if err != nil {
		result.Error = "Package file could not be created."
		return finish(err)
	}
	zipWriter := zip.NewWriter(archive)
	fileCount := 0
	walkErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(filepath.Base(source), relative))
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := copyWithContext(ctx, writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		fileCount++
		return nil
	})
	closeZipErr := zipWriter.Close()
	closeArchiveErr := archive.Close()
	if walkErr == nil {
		walkErr = closeZipErr
	}
	if walkErr == nil {
		walkErr = closeArchiveErr
	}
	if errors.Is(walkErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		result.Status = models.PackageStatusCancelled
		result.Error = "Packaging cancelled."
		_ = os.Remove(temporary)
		return finish(walkErr)
	}
	if walkErr != nil {
		result.Error = "Package creation failed."
		_ = os.Remove(temporary)
		return finish(walkErr)
	}
	if fileCount == 0 {
		result.Error = "Build output directory is empty."
		_ = os.Remove(temporary)
		return finish(errors.New("no regular files found in build output"))
	}
	if err := validateArchive(temporary); err != nil {
		result.Error = "Created package could not be validated."
		_ = os.Remove(temporary)
		return finish(err)
	}
	if err := ctx.Err(); err != nil {
		result.Status = models.PackageStatusCancelled
		result.Error = "Packaging cancelled."
		_ = os.Remove(temporary)
		return finish(err)
	}
	if overwrite {
		if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
			result.Error = "Existing package could not be replaced."
			_ = os.Remove(temporary)
			return finish(err)
		}
	}
	if err := os.Rename(temporary, destination); err != nil {
		result.Error = "Package could not be finalized."
		_ = os.Remove(temporary)
		return finish(err)
	}
	fileInfo, err := os.Stat(destination)
	if err != nil || fileInfo.Size() == 0 {
		result.Error = "Created package is empty or could not be read."
		return finish(err)
	}
	result.Success = true
	result.Status = models.PackageStatusSuccess
	result.SizeBytes = fileInfo.Size()
	return finish(nil)
}

func (s *Service) writeRecord(runID string, result models.PackageResult, technical error) {
	if s.logger != nil && technical != nil {
		s.logger.Error(fmt.Sprintf("Packaging %s/%s: %v", result.ProjectName, result.ComponentName, technical))
	}
	if s.history == nil {
		return
	}
	record := models.PackageRecord{
		Timestamp:        time.Now(),
		RunID:            runID,
		ProjectID:        result.ProjectID,
		ProjectName:      result.ProjectName,
		ComponentID:      result.ComponentID,
		ComponentName:    result.ComponentName,
		SourcePath:       result.SourcePath,
		PackagePath:      result.PackagePath,
		Version:          result.Version,
		FilenameTemplate: result.FilenameTemplate,
		ResolvedFilename: result.ResolvedFilename,
		StartTime:        result.StartTime,
		EndTime:          result.EndTime,
		DurationMs:       result.DurationMs,
		SizeBytes:        result.SizeBytes,
		Success:          result.Success,
		Status:           result.Status,
		Error:            result.Error,
	}
	if technical != nil {
		record.TechnicalError = technical.Error()
	}
	if err := s.history.Append(record); err != nil && s.logger != nil {
		s.logger.Error("Packaging history could not be written: " + err.Error())
	}
}

func (s *Service) emit(sink EventSink, event models.BuildEvent) {
	if sink != nil {
		sink(event)
	}
}

func (s *Service) emitState(sink EventSink, run models.PackageRun, state models.PackageComponentState, eventType string) {
	s.emit(sink, models.BuildEvent{
		Type:          eventType,
		Phase:         Phase,
		RunID:         run.ID,
		ProjectID:     run.ProjectID,
		ProjectName:   run.ProjectName,
		ComponentID:   state.ComponentID,
		ComponentName: state.ComponentName,
		Version:       run.Version,
		Environment:   run.Environment,
		PackageStatus: state.Status,
		PackageResult: state.Result,
		Error:         state.Message,
		Timestamp:     time.Now(),
	})
}

func skipRemaining(states []models.PackageComponentState, reason string, sink EventSink, run models.PackageRun) {
	for i := range states {
		if !states[i].Selected || states[i].Status != models.PackageStatusReady {
			continue
		}
		states[i].Status = models.PackageStatusSkipped
		states[i].Message = reason
		if sink != nil {
			sink(models.BuildEvent{
				Type:          EventComponentFinished,
				Phase:         Phase,
				RunID:         run.ID,
				ProjectID:     run.ProjectID,
				ProjectName:   run.ProjectName,
				ComponentID:   states[i].ComponentID,
				ComponentName: states[i].ComponentName,
				Version:       run.Version,
				Environment:   run.Environment,
				PackageStatus: states[i].Status,
				Error:         reason,
				Timestamp:     time.Now(),
			})
		}
	}
}

func packageResults(run models.PackageRun) []models.PackageResult {
	results := make([]models.PackageResult, 0, len(run.Components))
	for _, state := range run.Components {
		if state.Result != nil {
			results = append(results, *state.Result)
		}
	}
	return results
}

func sourcePath(component models.Component) string {
	path, err := filepath.Abs(pathutil.ResolveOutputDirectory(component.Path, component.OutputDirectory))
	if err != nil {
		return filepath.Clean(pathutil.ResolveOutputDirectory(component.Path, component.OutputDirectory))
	}
	return filepath.Clean(path)
}

func normalizeVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return time.Now().Format(defaultVersionFormat), nil
	}
	if !validVersion.MatchString(version) || version == "." || version == ".." {
		return "", errors.New("version may contain only letters, numbers, dots, underscores, and hyphens")
	}
	return version, nil
}

func safePathSegment(value, fallback string) string {
	value = strings.TrimSpace(value)
	value = invalidPathSegment.ReplaceAllString(value, "_")
	value = strings.Trim(value, " .")
	if value == "" {
		return fallback
	}
	return value
}

func validateFilename(filename string) error {
	original := filename
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == ".." || filepath.Base(filename) != filename || invalidPathSegment.MatchString(filename) || original != strings.TrimRight(original, " .") {
		return errors.New("package filename must be a simple file name without path separators or invalid Windows characters")
	}
	base := strings.ToUpper(strings.SplitN(filename, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
		return errors.New("package filename uses a reserved Windows device name")
	}
	for _, character := range filename {
		if character < 32 {
			return errors.New("package filename contains an invalid control character")
		}
	}
	return nil
}

func validateArchive(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return errors.New("package archive is empty")
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) == 0 {
		return errors.New("package archive contains no files")
	}
	return nil
}

func copyWithContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 64*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return total, nil
			}
			return total, readErr
		}
	}
}

func statusMessage(status string) string {
	if status == "" {
		return "Ready"
	}
	return strings.ToUpper(status[:1]) + status[1:]
}
