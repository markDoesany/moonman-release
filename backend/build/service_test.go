package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
)

type fakeRunner struct {
	outcomes []processOutcome
	calls    []string
}

func (r *fakeRunner) Run(_ context.Context, command, _ string, emit func(string, string)) processOutcome {
	r.calls = append(r.calls, command)
	emit(StreamStdout, "output from "+command)
	index := len(r.calls) - 1
	outcome := r.outcomes[index]
	if outcome.StartTime.IsZero() {
		outcome.StartTime = time.Now()
	}
	if outcome.EndTime.IsZero() {
		outcome.EndTime = outcome.StartTime.Add(25 * time.Millisecond)
	}
	return outcome
}

func testProject(t *testing.T, names ...string) models.Project {
	t.Helper()
	components := make([]models.Component, 0, len(names))
	for index, name := range names {
		components = append(components, models.Component{
			ID:              name,
			Name:            name,
			Path:            filepath.Join(t.TempDir(), name),
			BuildCommand:    "build-" + name,
			OutputDirectory: "dist",
		})
		if err := os.MkdirAll(components[index].Path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return models.Project{ID: "project", Name: "Project", Components: components}
}

func newTestService(t *testing.T, runner processRunner) (*Service, string) {
	t.Helper()
	base := t.TempDir()
	logger := logging.New(filepath.Join(base, "app.log"))
	historyPath := filepath.Join(base, "builds.jsonl")
	return newServiceWithRunner(logger, logging.NewJSONLWriter(historyPath), runner), historyPath
}

func TestPrepareRunPreservesProjectOrderAndSelection(t *testing.T) {
	project := testProject(t, "admin", "customer", "merchant")
	service, _ := newTestService(t, &fakeRunner{})

	run, err := service.PrepareRun("run-1", project, []string{"merchant", "admin"})
	if err != nil {
		t.Fatalf("PrepareRun() error = %v", err)
	}
	if run.Components[0].ComponentID != "admin" || !run.Components[0].Selected {
		t.Fatalf("first component was not selected in project order: %+v", run.Components[0])
	}
	if run.Components[1].Selected {
		t.Fatal("customer should not be selected")
	}
	if _, err := service.PrepareRun("run-2", project, []string{"missing"}); err == nil {
		t.Fatal("unknown component should be rejected")
	}
}

func TestExecuteStopsAfterFailureAndSkipsRemaining(t *testing.T) {
	project := testProject(t, "admin", "customer", "merchant")
	if err := os.MkdirAll(filepath.Join(project.Components[0].Path, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project.Components[1].Path, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{outcomes: []processOutcome{
		{ExitCode: 0},
		{ExitCode: 7, Err: &fakeExitError{}},
		{ExitCode: 0},
	}}
	service, _ := newTestService(t, runner)
	run, err := service.PrepareRun("run-1", project, []string{"admin", "customer", "merchant"})
	if err != nil {
		t.Fatal(err)
	}
	final := service.Execute(context.Background(), run, project, nil)

	if len(runner.calls) != 2 {
		t.Fatalf("expected two process calls, got %d", len(runner.calls))
	}
	if final.Status != models.BuildRunStatusFailed {
		t.Fatalf("run status = %q, want failed", final.Status)
	}
	if final.Components[0].Status != models.BuildStatusSuccess || final.Components[1].Status != models.BuildStatusFailed {
		t.Fatalf("unexpected completed states: %+v", final.Components)
	}
	if final.Components[2].Status != models.BuildStatusSkipped {
		t.Fatalf("remaining component status = %q, want skipped", final.Components[2].Status)
	}
}

func TestExecuteValidatesOutputAndStreamsOutput(t *testing.T) {
	project := testProject(t, "admin")
	runner := &fakeRunner{outcomes: []processOutcome{{ExitCode: 0}}}
	service, historyPath := newTestService(t, runner)
	run, err := service.PrepareRun("run-1", project, []string{"admin"})
	if err != nil {
		t.Fatal(err)
	}
	var events []models.BuildEvent
	final := service.Execute(context.Background(), run, project, func(event models.BuildEvent) {
		events = append(events, event)
	})

	if final.Status != models.BuildRunStatusFailed || final.Components[0].Status != models.BuildStatusFailed {
		t.Fatalf("missing output should fail the run: %+v", final)
	}
	if len(events) == 0 || !hasOutputEvent(events) {
		t.Fatalf("expected streamed output event, got %+v", events)
	}
	contents, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("read build history: %v", err)
	}
	var record models.BuildRecord
	if err := json.Unmarshal(contents[:len(contents)-1], &record); err != nil {
		t.Fatalf("decode build history: %v", err)
	}
	if record.Status != models.BuildStatusFailed || record.OutputDirectory != "dist" || record.Error == "" {
		t.Fatalf("unexpected build record: %+v", record)
	}
}

func TestExecuteReportsCancellation(t *testing.T) {
	project := testProject(t, "admin", "customer")
	runner := &fakeRunner{outcomes: []processOutcome{{ExitCode: -1, Cancelled: true}}}
	service, _ := newTestService(t, runner)
	run, err := service.PrepareRun("run-1", project, []string{"admin", "customer"})
	if err != nil {
		t.Fatal(err)
	}
	final := service.Execute(context.Background(), run, project, nil)
	if final.Status != models.BuildRunStatusCancelled {
		t.Fatalf("run status = %q, want cancelled", final.Status)
	}
	if final.Components[0].Status != models.BuildStatusCancelled || final.Components[1].Status != models.BuildStatusSkipped {
		t.Fatalf("unexpected cancellation states: %+v", final.Components)
	}
}

func TestExecuteRejectsMissingProjectPath(t *testing.T) {
	project := testProject(t, "admin")
	project.Components[0].Path = filepath.Join(t.TempDir(), "missing")
	runner := &fakeRunner{}
	service, _ := newTestService(t, runner)
	run, err := service.PrepareRun("run-1", project, []string{"admin"})
	if err != nil {
		t.Fatal(err)
	}
	final := service.Execute(context.Background(), run, project, nil)
	if len(runner.calls) != 0 {
		t.Fatal("runner should not be called for a missing project path")
	}
	if final.Components[0].Result == nil || final.Components[0].Result.Error != "Project directory does not exist." {
		t.Fatalf("unexpected path validation result: %+v", final.Components[0])
	}
}

func hasOutputEvent(events []models.BuildEvent) bool {
	for _, event := range events {
		if event.Type == EventOutput && event.Stream == StreamStdout && event.Text != "" {
			return true
		}
	}
	return false
}

type fakeExitError struct{}

func (*fakeExitError) Error() string { return "process exited with code 7" }
