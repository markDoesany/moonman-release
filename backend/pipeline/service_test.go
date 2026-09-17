package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"release-launcher/backend/build"
	"release-launcher/backend/logging"
	"release-launcher/backend/models"
	"release-launcher/backend/packaging"
)

type fakeBuilder struct{}

func (fakeBuilder) BuildComponent(_ context.Context, run models.BuildRun, project models.Project, component models.Component, emit build.EventSink) models.BuildResult {
	outputPath := filepath.Join(component.Path, component.OutputDirectory)
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return models.BuildResult{ProjectID: project.ID, ProjectName: project.Name, ComponentID: component.ID, ComponentName: component.Name, Status: models.BuildStatusFailed, ExitCode: -1, Error: err.Error()}
	}
	if err := os.WriteFile(filepath.Join(outputPath, "index.html"), []byte(component.Name), 0o644); err != nil {
		return models.BuildResult{ProjectID: project.ID, ProjectName: project.Name, ComponentID: component.ID, ComponentName: component.Name, Status: models.BuildStatusFailed, ExitCode: -1, Error: err.Error()}
	}
	if emit != nil {
		emit(models.BuildEvent{Type: build.EventOutput, Stream: build.StreamStdout, Text: "build output"})
	}
	now := time.Now()
	return models.BuildResult{ProjectID: run.ProjectID, ProjectName: run.ProjectName, ComponentID: component.ID, ComponentName: component.Name, Success: true, Status: models.BuildStatusSuccess, ExitCode: 0, OutputPath: outputPath, StartTime: now, EndTime: now, DurationMs: 0}
}

func TestExecuteBuildsThenPackagesEachComponent(t *testing.T) {
	root := t.TempDir()
	components := []models.Component{
		{ID: "admin", Name: "Admin", Path: filepath.Join(root, "admin"), OutputDirectory: "build", Package: models.PackageConfig{Enabled: true, Filename: "admin.zip"}},
		{ID: "customer", Name: "Customer", Path: filepath.Join(root, "customer"), OutputDirectory: "build", Package: models.PackageConfig{Enabled: true, Filename: "customer.zip"}},
	}
	project := models.Project{ID: "lokalstore", Name: "LokalStore", Components: components}
	for _, component := range components {
		if err := os.MkdirAll(component.Path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	base := t.TempDir()
	packager := packaging.NewService(logging.New(filepath.Join(base, "app.log")), logging.NewJSONLWriter(filepath.Join(base, "packaging.jsonl")), filepath.Join(root, "releases"))
	pipeline := NewService(fakeBuilder{}, packager)
	request := models.PackageRequest{ProjectID: project.ID, ComponentIDs: []string{"admin", "customer"}, Version: "1.0.0"}
	run, err := pipeline.PrepareRun("run-1", project, request)
	if err != nil {
		t.Fatal(err)
	}
	var events []models.BuildEvent
	final := pipeline.Execute(context.Background(), run, project, request, func(event models.BuildEvent) { events = append(events, event) })
	if final.Status != models.ReleaseRunStatusCompleted {
		t.Fatalf("run status = %q, want completed", final.Status)
	}
	for _, state := range final.Components {
		if state.BuildStatus != models.BuildStatusSuccess || state.PackageStatus != models.PackageStatusSuccess || state.PackageResult == nil {
			t.Fatalf("component was not built and packaged: %+v", state)
		}
		if _, err := os.Stat(state.PackageResult.PackagePath); err != nil {
			t.Fatalf("package does not exist: %v", err)
		}
	}
	if !hasPhaseEvent(events, "build") || !hasPackageEvent(events) {
		t.Fatalf("expected build and package events, got %+v", events)
	}
}

func hasPhaseEvent(events []models.BuildEvent, phase string) bool {
	for _, event := range events {
		if event.Phase == phase {
			return true
		}
	}
	return false
}

func hasPackageEvent(events []models.BuildEvent) bool {
	for _, event := range events {
		if event.PackageStatus == models.PackageStatusPackaging || event.PackageStatus == models.PackageStatusSuccess {
			return true
		}
	}
	return false
}
