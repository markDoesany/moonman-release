package transfer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
	"release-launcher/backend/packaging"
)

const (
	Phase                  = "transfer"
	EventRunStarted        = "transfer_run_started"
	EventComponentStarted  = "transfer_started"
	EventComponentFinished = "transfer_finished"
	EventRunFinished       = "transfer_run_finished"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

type EventSink func(models.BuildEvent)

type processOutcome struct {
	StartTime time.Time
	EndTime   time.Time
	ExitCode  int
	Cancelled bool
	Err       error
	Technical error
}

type processRunner interface {
	Run(context.Context, string, []string, func(string, string)) processOutcome
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, executable string, args []string, emit func(string, string)) processOutcome {
	started := time.Now()
	cmd := exec.CommandContext(ctx, executable, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return processOutcome{StartTime: started, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return processOutcome{StartTime: started, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return processOutcome{StartTime: started, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}
	var output sync.WaitGroup
	output.Add(2)
	go streamOutput(stdout, "stdout", emit, &output)
	go streamOutput(stderr, "stderr", emit, &output)
	waitErr := cmd.Wait()
	output.Wait()
	outcome := processOutcome{StartTime: started, EndTime: time.Now(), ExitCode: 0, Err: waitErr, Technical: waitErr, Cancelled: errors.Is(ctx.Err(), context.Canceled)}
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		outcome.ExitCode = exitErr.ExitCode()
	}
	if outcome.Cancelled {
		outcome.ExitCode = -1
	}
	return outcome
}

func streamOutput(reader io.Reader, stream string, emit func(string, string), done *sync.WaitGroup) {
	defer done.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 32*1024), 1024*1024)
	for scanner.Scan() {
		if emit != nil {
			emit(stream, cleanOutput(scanner.Text()))
		}
	}
}

func cleanOutput(value string) string {
	return ansiEscape.ReplaceAllString(value, "")
}

type Service struct {
	logger   *logging.Logger
	history  *logging.JSONLWriter
	packager *packaging.Service
	runner   processRunner
	lookPath func(string) (string, error)
}

func NewService(logger *logging.Logger, history *logging.JSONLWriter, packager *packaging.Service) *Service {
	return &Service{logger: logger, history: history, packager: packager, runner: commandRunner{}, lookPath: exec.LookPath}
}

func newServiceWithRunner(logger *logging.Logger, history *logging.JSONLWriter, packager *packaging.Service, runner processRunner, lookPath func(string) (string, error)) *Service {
	service := NewService(logger, history, packager)
	service.runner = runner
	service.lookPath = lookPath
	return service
}

// Plan resolves packaged artifacts and reports any files that are not ready to send.
func (s *Service) Plan(project models.Project, componentIDs []string, version string) (models.TransferPlan, error) {
	return s.PlanRequest(project, models.TransferRequest{ProjectID: project.ID, ComponentIDs: componentIDs, Version: version})
}

func (s *Service) PlanRequest(project models.Project, request models.TransferRequest) (models.TransferPlan, error) {
	packagePlan, err := s.packager.PlanRequest(project, models.PackageRequest{ProjectID: project.ID, ComponentIDs: request.ComponentIDs, Version: request.Version, FilenameTemplate: request.FilenameTemplate, PackageNames: request.PackageNames})
	if err != nil {
		return models.TransferPlan{}, err
	}
	items := make([]models.TransferPlanItem, 0, len(packagePlan.Components))
	hasMissing := false
	for _, item := range packagePlan.Components {
		transferItem := models.TransferPlanItem{
			ComponentID:      item.ComponentID,
			ComponentName:    item.ComponentName,
			Selected:         item.Selected,
			Enabled:          item.Enabled,
			PackagePath:      item.PackagePath,
			FilenameTemplate: item.FilenameTemplate,
			ResolvedFilename: item.ResolvedFilename,
		}
		if item.Selected && item.Enabled {
			info, statErr := os.Stat(item.PackagePath)
			if statErr != nil || !info.Mode().IsRegular() {
				transferItem.Error = "Package file does not exist."
				hasMissing = true
			} else {
				transferItem.Exists = true
			}
		}
		items = append(items, transferItem)
	}
	return models.TransferPlan{ProjectID: packagePlan.ProjectID, ProjectName: packagePlan.ProjectName, Version: packagePlan.Version, ReleaseDirectory: packagePlan.ReleaseDirectory, FilenameTemplate: packagePlan.FilenameTemplate, Components: items, HasMissing: hasMissing}, nil
}

func (s *Service) PrepareRun(runID string, project models.Project, request models.TransferRequest) (models.TransferRun, error) {
	plan, err := s.PlanRequest(project, request)
	if err != nil {
		return models.TransferRun{}, err
	}
	if plan.HasMissing {
		return models.TransferRun{}, errors.New("one or more selected packages do not exist")
	}
	selected := make(map[string]bool, len(request.ComponentIDs))
	for _, id := range request.ComponentIDs {
		selected[strings.TrimSpace(id)] = true
	}
	states := make([]models.TransferComponentState, 0, len(project.Components))
	for _, component := range project.Components {
		status := models.TransferStatusReady
		message := "Ready"
		if !selected[component.ID] {
			status = models.TransferStatusSkipped
			message = "Not selected"
		} else if !component.Package.Enabled {
			status = models.TransferStatusSkipped
			message = "Packaging disabled"
		}
		states = append(states, models.TransferComponentState{ComponentID: component.ID, ComponentName: component.Name, Selected: selected[component.ID], Status: status, Message: message})
	}
	return models.TransferRun{ID: runID, ProjectID: project.ID, ProjectName: project.Name, Environment: project.EffectiveEnvironment(request.Environment), Version: plan.Version, Status: models.TransferRunStatusRunning, Components: states, StartTime: time.Now()}, nil
}

func (s *Service) Execute(ctx context.Context, run models.TransferRun, project models.Project, request models.TransferRequest, settings models.DaliConfig, emit EventSink) models.TransferRun {
	request.Version = run.Version
	plan, err := s.PlanRequest(project, request)
	if err != nil {
		run.Status = models.TransferRunStatusFailed
		run.Error = err.Error()
		return s.finish(run, emit)
	}
	if plan.HasMissing {
		run.Status = models.TransferRunStatusFailed
		run.Error = "one or more selected packages do not exist"
		return s.finish(run, emit)
	}

	s.emit(emit, models.BuildEvent{Type: EventRunStarted, Phase: Phase, RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Version: run.Version, Environment: run.Environment, Timestamp: time.Now()})
	components := make(map[string]models.Component, len(project.Components))
	items := make(map[string]models.TransferPlanItem, len(plan.Components))
	for _, component := range project.Components {
		components[component.ID] = component
	}
	for _, item := range plan.Components {
		items[item.ComponentID] = item
	}

	for i := range run.Components {
		state := &run.Components[i]
		if !state.Selected || state.Status == models.TransferStatusSkipped {
			if state.Status == models.TransferStatusSkipped {
				s.emitState(emit, run, *state, EventComponentFinished)
			}
			continue
		}
		if err := ctx.Err(); err != nil {
			state.Status = models.TransferStatusCancelled
			state.Message = "Transfer cancelled before this component started"
			s.emitState(emit, run, *state, EventComponentFinished)
			skipRemaining(run.Components[i+1:], "Transfer cancelled", emit, run)
			run.Status = models.TransferRunStatusCancelled
			run.Error = "Transfer cancelled"
			break
		}
		component := components[state.ComponentID]
		item := items[state.ComponentID]
		state.Status = models.TransferStatusSending
		state.Message = "Sending..."
		s.emitState(emit, run, *state, EventComponentStarted)
		result := s.SendOneWithMetadata(ctx, run.ID, project, component, item.PackagePath, item.FilenameTemplate, item.ResolvedFilename, run.Version, settings, emit)
		state.Result = &result
		state.Status = result.Status
		state.Message = result.Error
		if state.Message == "" {
			state.Message = statusMessage(string(result.Status))
		}
		s.emitState(emit, run, *state, EventComponentFinished)
		if result.Status == models.TransferStatusCancelled {
			run.Status = models.TransferRunStatusCancelled
			run.Error = "Transfer cancelled"
			skipRemaining(run.Components[i+1:], "Transfer cancelled", emit, run)
			break
		}
		if !result.Success {
			run.Status = models.TransferRunStatusFailed
			run.Error = fmt.Sprintf("%s failed to send.", component.Name)
			skipRemaining(run.Components[i+1:], "Previous transfer failed", emit, run)
			break
		}
	}
	if run.Status == models.TransferRunStatusRunning {
		run.Status = models.TransferRunStatusCompleted
	}
	return s.finish(run, emit)
}

func (s *Service) SendOne(ctx context.Context, runID string, project models.Project, component models.Component, packagePath string, settings models.DaliConfig, emit EventSink) models.TransferResult {
	return s.SendOneWithMetadata(ctx, runID, project, component, packagePath, "", filepath.Base(packagePath), "", settings, emit)
}

func (s *Service) SendOneWithMetadata(ctx context.Context, runID string, project models.Project, component models.Component, packagePath, filenameTemplate, resolvedFilename, version string, settings models.DaliConfig, emit EventSink) models.TransferResult {
	started := time.Now()
	executable := strings.TrimSpace(settings.Executable)
	if executable == "" {
		executable = models.DefaultDaliConfig().Executable
	}
	args, err := commandArgs(packagePath, settings)
	result := models.TransferResult{ProjectID: project.ID, ProjectName: project.Name, ComponentID: component.ID, ComponentName: component.Name, PackagePath: packagePath, Version: version, FilenameTemplate: filenameTemplate, ResolvedFilename: resolvedFilename, Executable: executable, Arguments: args, PeerName: settings.PeerName, PeerAddress: settings.PeerAddress, Status: models.TransferStatusFailed, ExitCode: -1, StartTime: started}
	finish := func(technical error) models.TransferResult {
		result.EndTime = time.Now()
		result.DurationMs = result.EndTime.Sub(result.StartTime).Milliseconds()
		s.writeRecord(runID, result, technical)
		return result
	}
	if err != nil {
		result.Error = err.Error()
		return finish(err)
	}
	if _, err := os.Stat(packagePath); err != nil {
		result.Error = "Package file does not exist."
		return finish(err)
	}
	resolved, err := s.lookPath(executable)
	if err != nil {
		result.Error = "Dali executable could not be found. Install Dali or configure its full path."
		return finish(err)
	}
	result.Executable = resolved
	var output strings.Builder
	s.emit(emit, models.BuildEvent{Type: "output", Phase: Phase, RunID: runID, ProjectID: project.ID, ProjectName: project.Name, ComponentID: component.ID, ComponentName: component.Name, Stream: "system", Text: formatCommand(resolved, args), Timestamp: time.Now()})
	outcome := s.runner.Run(ctx, resolved, args, func(stream, text string) {
		text = cleanOutput(text)
		output.WriteString(text)
		output.WriteByte('\n')
		s.emit(emit, models.BuildEvent{Type: "output", Phase: Phase, RunID: runID, ProjectID: project.ID, ProjectName: project.Name, ComponentID: component.ID, ComponentName: component.Name, Stream: stream, Text: text, Timestamp: time.Now()})
	})
	result.StartTime = outcome.StartTime
	result.EndTime = outcome.EndTime
	result.ExitCode = outcome.ExitCode
	if result.StartTime.IsZero() {
		result.StartTime = started
	}
	if result.EndTime.IsZero() {
		result.EndTime = time.Now()
	}
	if outcome.Cancelled || errors.Is(ctx.Err(), context.Canceled) {
		result.Status = models.TransferStatusCancelled
		result.Error = "Transfer cancelled."
		return finish(outcome.Technical)
	}
	if outcome.Err != nil {
		result.Error = fmt.Sprintf("Dali command failed with exit code %d.", result.ExitCode)
		return finish(outcome.Technical)
	}
	message := strings.ToLower(output.String())
	switch {
	case strings.Contains(message, "file sent successfully"):
		result.Success = true
		result.Status = models.TransferStatusSuccess
	case strings.Contains(message, "peer rejected"), strings.Contains(message, "no peers found"), strings.Contains(message, "error:"), strings.Contains(message, "failed to"):
		result.Error = "Dali did not complete the file transfer."
	default:
		result.Error = "Dali exited without confirming the file transfer."
	}
	return finish(nil)
}

func commandArgs(packagePath string, settings models.DaliConfig) ([]string, error) {
	if strings.TrimSpace(packagePath) == "" {
		return nil, errors.New("package path cannot be empty")
	}
	args := []string{"send", "file=" + packagePath}
	switch {
	case strings.TrimSpace(settings.PeerAddress) != "":
		args = append(args, "to="+strings.TrimSpace(settings.PeerAddress))
	case strings.TrimSpace(settings.PeerName) != "":
		args = append(args, "for="+strings.TrimSpace(settings.PeerName))
	case settings.Auto:
		args = append(args, "auto=1")
	default:
		return nil, errors.New("configure a Dali peer name/address or enable automatic peer selection")
	}
	if settings.Wait {
		args = append(args, "wait")
	}
	return args, nil
}

func formatCommand(executable string, args []string) string {
	return executable + " " + strings.Join(args, " ")
}

func (s *Service) finish(run models.TransferRun, emit EventSink) models.TransferRun {
	run.EndTime = time.Now()
	results := make([]models.TransferResult, 0, len(run.Components))
	for _, state := range run.Components {
		if state.Result != nil {
			results = append(results, *state.Result)
		}
	}
	s.emit(emit, models.BuildEvent{Type: EventRunFinished, Phase: Phase, RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, Version: run.Version, Environment: run.Environment, RunStatus: models.BuildRunStatus(run.Status), TransferResults: results, Error: run.Error, Timestamp: run.EndTime})
	return run
}

func (s *Service) emit(sink EventSink, event models.BuildEvent) {
	if sink != nil {
		sink(event)
	}
}

func (s *Service) emitState(sink EventSink, run models.TransferRun, state models.TransferComponentState, eventType string) {
	s.emit(sink, models.BuildEvent{Type: eventType, Phase: Phase, RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, ComponentID: state.ComponentID, ComponentName: state.ComponentName, Version: run.Version, Environment: run.Environment, TransferStatus: state.Status, TransferResult: state.Result, Error: state.Message, Timestamp: time.Now()})
}

func skipRemaining(states []models.TransferComponentState, reason string, sink EventSink, run models.TransferRun) {
	for i := range states {
		if !states[i].Selected || states[i].Status != models.TransferStatusReady {
			continue
		}
		states[i].Status = models.TransferStatusSkipped
		states[i].Message = reason
		if sink != nil {
			sink(models.BuildEvent{Type: EventComponentFinished, Phase: Phase, RunID: run.ID, ProjectID: run.ProjectID, ProjectName: run.ProjectName, ComponentID: states[i].ComponentID, ComponentName: states[i].ComponentName, Version: run.Version, Environment: run.Environment, TransferStatus: states[i].Status, Error: reason, Timestamp: time.Now()})
		}
	}
}

func (s *Service) writeRecord(runID string, result models.TransferResult, technical error) {
	if s.logger != nil && technical != nil {
		s.logger.Error(fmt.Sprintf("Dali transfer %s/%s: %v", result.ProjectName, result.ComponentName, technical))
	}
	if s.history == nil {
		return
	}
	record := models.TransferRecord{Timestamp: time.Now(), RunID: runID, ProjectID: result.ProjectID, ProjectName: result.ProjectName, ComponentID: result.ComponentID, ComponentName: result.ComponentName, PackagePath: result.PackagePath, Version: result.Version, FilenameTemplate: result.FilenameTemplate, ResolvedFilename: result.ResolvedFilename, Executable: result.Executable, Arguments: result.Arguments, PeerName: result.PeerName, PeerAddress: result.PeerAddress, ExitCode: result.ExitCode, StartTime: result.StartTime, EndTime: result.EndTime, DurationMs: result.DurationMs, Success: result.Success, Status: result.Status, Error: result.Error}
	if technical != nil {
		record.TechnicalError = technical.Error()
	}
	if err := s.history.Append(record); err != nil && s.logger != nil {
		s.logger.Error("Dali transfer history could not be written: " + err.Error())
	}
}

func statusMessage(status string) string {
	if status == "" {
		return "Ready"
	}
	return strings.ToUpper(status[:1]) + status[1:]
}
