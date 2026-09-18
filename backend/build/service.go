package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
	"release-launcher/backend/pathutil"
)

const (
	EventName              = "release-launcher:build-event"
	EventRunStarted        = "run_started"
	EventComponentStarted  = "component_started"
	EventOutput            = "output"
	EventComponentFinished = "component_finished"
	EventRunFinished       = "run_finished"

	StreamSystem = "system"
	StreamStdout = "stdout"
	StreamStderr = "stderr"
)

// EventSink receives build events without coupling the build service to Wails.
type EventSink func(models.BuildEvent)

type processOutcome struct {
	StartTime time.Time
	EndTime   time.Time
	ExitCode  int
	Started   bool
	Cancelled bool
	Err       error
	Technical error
}

type processRunner interface {
	Run(context.Context, string, string, func(string, string)) processOutcome
}

// Service orchestrates validated, sequential component builds.
type Service struct {
	runner  processRunner
	logger  *logging.Logger
	history *logging.JSONLWriter
}

func NewService(logger *logging.Logger, history *logging.JSONLWriter) *Service {
	return &Service{
		runner:  newProcessRunner(),
		logger:  logger,
		history: history,
	}
}

func newServiceWithRunner(logger *logging.Logger, history *logging.JSONLWriter, runner processRunner) *Service {
	service := NewService(logger, history)
	service.runner = runner
	return service
}

// PrepareRun validates the request and creates the initial run state.
func (s *Service) PrepareRun(runID string, project models.Project, componentIDs []string) (models.BuildRun, error) {
	return s.PrepareRunForEnvironment(runID, project, componentIDs, "")
}

func (s *Service) PrepareRunForEnvironment(runID string, project models.Project, componentIDs []string, environment string) (models.BuildRun, error) {
	selected := make(map[string]bool, len(componentIDs))
	for _, id := range componentIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			selected[id] = true
		}
	}
	if len(selected) == 0 {
		return models.BuildRun{}, errors.New("select at least one component to build")
	}

	known := make(map[string]bool, len(project.Components))
	states := make([]models.BuildComponentState, 0, len(project.Components))
	for _, component := range project.Components {
		known[component.ID] = true
		states = append(states, models.BuildComponentState{
			ComponentID:   component.ID,
			ComponentName: component.Name,
			Selected:      selected[component.ID],
			Status:        models.BuildStatusReady,
			Message:       "Ready",
		})
	}
	for id := range selected {
		if !known[id] {
			return models.BuildRun{}, fmt.Errorf("component %q is not configured for project %q", id, project.Name)
		}
	}

	return models.BuildRun{
		ID:          runID,
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Environment: environment,
		Status:      models.BuildRunStatusRunning,
		Components:  states,
		StartTime:   time.Now(),
	}, nil
}

// Execute runs selected components in project configuration order.
func (s *Service) Execute(ctx context.Context, run models.BuildRun, project models.Project, emit EventSink) models.BuildRun {
	s.emit(emit, models.BuildEvent{
		Type:        EventRunStarted,
		RunID:       run.ID,
		ProjectID:   run.ProjectID,
		ProjectName: run.ProjectName,
		RunStatus:   run.Status,
		Environment: run.Environment,
		Timestamp:   time.Now(),
	})

	componentByID := make(map[string]models.Component, len(project.Components))
	for _, component := range project.Components {
		componentByID[component.ID] = component
	}

	for i := range run.Components {
		if !run.Components[i].Selected {
			run.Components[i].Status = models.BuildStatusSkipped
			run.Components[i].Message = "Not selected"
			s.emitComponentState(emit, run, run.Components[i], EventComponentFinished)
		}
	}

	for i := range run.Components {
		state := &run.Components[i]
		if !state.Selected || run.Status != models.BuildRunStatusRunning {
			continue
		}
		if err := ctx.Err(); err != nil {
			state.Status = models.BuildStatusSkipped
			state.Message = "Build cancelled before this component started"
			s.emitComponentState(emit, run, *state, EventComponentFinished)
			skipRemaining(run.Components[i+1:], "Build cancelled", emit, run)
			run.Status = models.BuildRunStatusCancelled
			run.Error = "Build cancelled"
			break
		}

		component := componentByID[state.ComponentID]
		command := component.CommandForEnvironment(run.Environment)
		state.Status = models.BuildStatusBuilding
		state.Message = "Building..."
		s.emitComponentState(emit, run, *state, EventComponentStarted)
		s.emit(emit, models.BuildEvent{
			Type:          EventOutput,
			RunID:         run.ID,
			ProjectID:     run.ProjectID,
			ProjectName:   run.ProjectName,
			ComponentID:   component.ID,
			ComponentName: component.Name,
			Stream:        StreamSystem,
			Text:          "> " + command,
			Environment:   run.Environment,
			Timestamp:     time.Now(),
		})

		result := s.buildComponent(ctx, run, component, emit)
		state.Result = &result
		state.Status = result.Status
		state.Message = result.Error
		if state.Message == "" {
			state.Message = strings.Title(string(result.Status))
		}
		s.emitComponentState(emit, run, *state, EventComponentFinished)

		if result.Status == models.BuildStatusCancelled {
			run.Status = models.BuildRunStatusCancelled
			run.Error = "Build cancelled"
			skipRemaining(run.Components[i+1:], "Build cancelled", emit, run)
			break
		}
		if !result.Success {
			run.Status = models.BuildRunStatusFailed
			run.Error = fmt.Sprintf("%s failed to build.", component.Name)
			skipRemaining(run.Components[i+1:], "Previous component failed", emit, run)
			break
		}
	}

	if run.Status == models.BuildRunStatusRunning {
		run.Status = models.BuildRunStatusCompleted
	}
	run.EndTime = time.Now()
	s.emit(emit, models.BuildEvent{
		Type:        EventRunFinished,
		RunID:       run.ID,
		ProjectID:   run.ProjectID,
		ProjectName: run.ProjectName,
		RunStatus:   run.Status,
		Results:     runResults(run),
		Error:       run.Error,
		Timestamp:   run.EndTime,
	})
	return run
}

// BuildComponent executes one component and validates its output directory.
// It is used by the build-only run and by the build-and-package pipeline.
func (s *Service) BuildComponent(ctx context.Context, run models.BuildRun, project models.Project, component models.Component, emit EventSink) models.BuildResult {
	return s.buildComponent(ctx, run, component, emit)
}

func (s *Service) buildComponent(ctx context.Context, run models.BuildRun, component models.Component, emit EventSink) models.BuildResult {
	start := time.Now()
	result := models.BuildResult{
		ProjectID:       run.ProjectID,
		ProjectName:     run.ProjectName,
		ComponentID:     component.ID,
		ComponentName:   component.Name,
		Environment:     run.Environment,
		Status:          models.BuildStatusFailed,
		ExitCode:        -1,
		OutputDirectory: component.OutputDirectory,
		OutputPath:      outputPath(component),
		StartTime:       start,
	}

	info, err := os.Stat(component.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Error = "Project directory does not exist."
		} else {
			result.Error = "Project directory could not be accessed."
		}
		result.EndTime = time.Now()
		s.finishResult(run, component, &result, err)
		return result
	}
	if !info.IsDir() {
		result.Error = "Project path is not a directory."
		result.EndTime = time.Now()
		s.finishResult(run, component, &result, errors.New("configured project path is not a directory"))
		return result
	}

	command := component.CommandForEnvironment(run.Environment)
	outcome := s.runner.Run(ctx, command, component.Path, func(stream, text string) {
		s.emit(emit, models.BuildEvent{
			Type:          EventOutput,
			RunID:         run.ID,
			ProjectID:     run.ProjectID,
			ProjectName:   run.ProjectName,
			ComponentID:   component.ID,
			ComponentName: component.Name,
			Stream:        stream,
			Text:          text,
			Environment:   run.Environment,
			Timestamp:     time.Now(),
		})
	})
	result.StartTime = outcome.StartTime
	result.EndTime = outcome.EndTime
	if result.StartTime.IsZero() {
		result.StartTime = start
	}
	if result.EndTime.IsZero() {
		result.EndTime = time.Now()
	}
	result.ExitCode = outcome.ExitCode

	if outcome.Cancelled || errors.Is(ctx.Err(), context.Canceled) {
		result.Status = models.BuildStatusCancelled
		result.Error = "Build cancelled."
		s.finishResult(run, component, &result, outcome.Technical)
		return result
	}
	if outcome.Err != nil {
		if result.ExitCode == 9009 {
			result.Error = "Build command could not be started. Make sure Node.js/npm is installed and available in PATH."
		} else if result.ExitCode >= 0 {
			result.Error = fmt.Sprintf("Build failed with exit code %d.", result.ExitCode)
		} else {
			result.Error = "Build command could not be started. Make sure Node.js/npm is installed and available in PATH."
		}
		s.finishResult(run, component, &result, outcome.Technical)
		return result
	}

	outputInfo, err := os.Stat(result.OutputPath)
	if err != nil || !outputInfo.IsDir() {
		result.Error = fmt.Sprintf("Build completed but expected output directory was not found: %s", result.OutputPath)
		if err == nil {
			err = errors.New("configured output path is not a directory")
		}
		s.finishResult(run, component, &result, err)
		return result
	}
	result.Success = true
	result.Status = models.BuildStatusSuccess
	result.Error = ""
	s.finishResult(run, component, &result, nil)
	return result
}

func (s *Service) finishResult(run models.BuildRun, component models.Component, result *models.BuildResult, technical error) {
	if result.EndTime.IsZero() {
		result.EndTime = time.Now()
	}
	result.DurationMs = result.EndTime.Sub(result.StartTime).Milliseconds()
	if s.logger != nil && technical != nil {
		s.logger.Error(fmt.Sprintf("Build %s/%s: %v", run.ProjectName, component.Name, technical))
	}
	if s.history == nil {
		return
	}
	record := models.BuildRecord{
		Timestamp:       time.Now(),
		RunID:           run.ID,
		ProjectID:       run.ProjectID,
		ProjectName:     run.ProjectName,
		ComponentID:     component.ID,
		ComponentName:   component.Name,
		Environment:     run.Environment,
		BuildCommand:    component.CommandForEnvironment(run.Environment),
		ProjectPath:     component.Path,
		StartTime:       result.StartTime,
		EndTime:         result.EndTime,
		DurationMs:      result.DurationMs,
		ExitCode:        result.ExitCode,
		Success:         result.Success,
		Status:          result.Status,
		OutputDirectory: result.OutputDirectory,
		OutputPath:      result.OutputPath,
		Error:           result.Error,
	}
	if technical != nil {
		record.TechnicalError = technical.Error()
	}
	if err := s.history.Append(record); err != nil && s.logger != nil {
		s.logger.Error("Build history could not be written: " + err.Error())
	}
}

func (s *Service) emit(sink EventSink, event models.BuildEvent) {
	if sink != nil {
		sink(event)
	}
}

func (s *Service) emitComponentState(sink EventSink, run models.BuildRun, state models.BuildComponentState, eventType string) {
	s.emit(sink, models.BuildEvent{
		Type:          eventType,
		RunID:         run.ID,
		ProjectID:     run.ProjectID,
		ProjectName:   run.ProjectName,
		ComponentID:   state.ComponentID,
		ComponentName: state.ComponentName,
		Status:        state.Status,
		Result:        state.Result,
		Error:         state.Message,
		Environment:   run.Environment,
		Timestamp:     time.Now(),
	})
}

func skipRemaining(states []models.BuildComponentState, reason string, sink EventSink, run models.BuildRun) {
	for i := range states {
		if !states[i].Selected || states[i].Status != models.BuildStatusReady {
			continue
		}
		states[i].Status = models.BuildStatusSkipped
		states[i].Message = reason
		if sink != nil {
			sink(models.BuildEvent{
				Type:          EventComponentFinished,
				RunID:         run.ID,
				ProjectID:     run.ProjectID,
				ProjectName:   run.ProjectName,
				ComponentID:   states[i].ComponentID,
				ComponentName: states[i].ComponentName,
				Status:        states[i].Status,
				Error:         reason,
				Timestamp:     time.Now(),
			})
		}
	}
}

func outputPath(component models.Component) string {
	path, err := filepath.Abs(pathutil.Resolve(component.Path, component.OutputDirectory))
	if err != nil {
		return pathutil.Resolve(component.Path, component.OutputDirectory)
	}
	return filepath.Clean(path)
}

func runResults(run models.BuildRun) []models.BuildResult {
	results := make([]models.BuildResult, 0, len(run.Components))
	for _, state := range run.Components {
		if state.Result != nil {
			results = append(results, *state.Result)
		}
	}
	return results
}
