package packaging

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
)

func packageProject(t *testing.T, componentNames ...string) (models.Project, string) {
	t.Helper()
	releaseRoot := t.TempDir()
	components := make([]models.Component, 0, len(componentNames))
	for _, name := range componentNames {
		path := filepath.Join(t.TempDir(), "source folder")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		components = append(components, models.Component{
			ID:              name,
			Name:            name,
			Path:            filepath.Dir(path),
			OutputDirectory: "source folder",
			Package:         models.PackageConfig{Enabled: true, Filename: name + ".zip"},
		})
	}
	return models.Project{ID: "lokalstore", Name: "LokalStore", Components: components}, releaseRoot
}

func packageService(t *testing.T, releaseRoot string) (*Service, string) {
	t.Helper()
	base := t.TempDir()
	historyPath := filepath.Join(base, "packaging.jsonl")
	return NewService(logging.New(filepath.Join(base, "app.log")), logging.NewJSONLWriter(historyPath), releaseRoot), historyPath
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPackageOnePreservesOutputDirectoryInArchive(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin")
	component := project.Components[0]
	writeFile(t, filepath.Join(component.Path, component.OutputDirectory, "index.html"), "<html></html>")
	writeFile(t, filepath.Join(component.Path, component.OutputDirectory, "assets", "app.js"), "console.log('ok')")
	service, historyPath := packageService(t, releaseRoot)
	plan, err := service.Plan(project, []string{"admin"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	result := service.PackageOne(context.Background(), "run-1", project, component, plan.Components[0].SourcePath, plan.Components[0].PackagePath, false)
	if !result.Success || result.SizeBytes == 0 {
		t.Fatalf("package result = %+v", result)
	}
	reader, err := zip.OpenReader(result.PackagePath)
	if err != nil {
		t.Fatalf("open package: %v", err)
	}
	defer reader.Close()
	names := make(map[string]bool)
	for _, file := range reader.File {
		names[file.Name] = true
	}
	if !names["source folder/index.html"] || !names["source folder/assets/app.js"] || names["index.html"] {
		t.Fatalf("unexpected archive entries: %v", names)
	}
	contents, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatal(err)
	}
	var record models.PackageRecord
	if err := json.Unmarshal(contents[:len(contents)-1], &record); err != nil {
		t.Fatal(err)
	}
	if record.Success != result.Success || record.SizeBytes != result.SizeBytes {
		t.Fatalf("unexpected history record: %+v", record)
	}
}

func TestPackageOneRejectsMissingAndEmptySources(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin", "empty")
	service, _ := packageService(t, releaseRoot)
	component := project.Components[0]
	missing := filepath.Join(t.TempDir(), "missing")
	result := service.PackageOne(context.Background(), "run-1", project, component, missing, filepath.Join(releaseRoot, "missing.zip"), false)
	if result.Success || result.Error != "Build output directory does not exist." {
		t.Fatalf("missing source result = %+v", result)
	}
	emptyComponent := project.Components[1]
	emptyResult := service.PackageOne(context.Background(), "run-2", project, emptyComponent, sourcePath(emptyComponent), filepath.Join(releaseRoot, "empty.zip"), false)
	if emptyResult.Success || emptyResult.Error != "Build output directory is empty." {
		t.Fatalf("empty source result = %+v", emptyResult)
	}
}

func TestPackagePlanDetectsConflictAndOverwrite(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin")
	component := project.Components[0]
	writeFile(t, filepath.Join(component.Path, component.OutputDirectory, "index.html"), "content")
	service, _ := packageService(t, releaseRoot)
	plan, err := service.Plan(project, []string{"admin"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, plan.Components[0].PackagePath, "old package")
	conflictPlan, err := service.Plan(project, []string{"admin"}, "1.0.0")
	if err != nil || !conflictPlan.HasConflicts || !conflictPlan.Components[0].Existing {
		t.Fatalf("conflict was not detected: %+v, error = %v", conflictPlan, err)
	}
	withoutOverwrite := service.PackageOne(context.Background(), "run-1", project, component, plan.Components[0].SourcePath, plan.Components[0].PackagePath, false)
	if withoutOverwrite.Success || withoutOverwrite.Error != "Package already exists. Confirm replace before packaging." {
		t.Fatalf("overwrite protection result = %+v", withoutOverwrite)
	}
	withOverwrite := service.PackageOne(context.Background(), "run-2", project, component, plan.Components[0].SourcePath, plan.Components[0].PackagePath, true)
	if !withOverwrite.Success {
		t.Fatalf("overwrite result = %+v", withOverwrite)
	}
}

func TestPackagePlanUsesRequestedReleaseDirectory(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin")
	customDirectory := filepath.Join(t.TempDir(), "approved-release")
	service, _ := packageService(t, releaseRoot)

	plan, err := service.PlanRequest(project, models.PackageRequest{
		ProjectID:        project.ID,
		ComponentIDs:     []string{"admin"},
		Version:          "1.0.0",
		ReleaseDirectory: customDirectory,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ReleaseDirectory != customDirectory {
		t.Fatalf("release directory = %q, want %q", plan.ReleaseDirectory, customDirectory)
	}
	if plan.Components[0].PackagePath != filepath.Join(customDirectory, "admin.zip") {
		t.Fatalf("package path = %q, want %q", plan.Components[0].PackagePath, filepath.Join(customDirectory, "admin.zip"))
	}
}

func TestPackagePlanUsesBuildFolderWhenOutputDirectoryIsProjectRoot(t *testing.T) {
	root := t.TempDir()
	componentRoot := filepath.Join(root, "admin")
	if err := os.MkdirAll(filepath.Join(componentRoot, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	project := models.Project{
		ID:   "project",
		Name: "Project",
		Components: []models.Component{{
			ID:              "admin",
			Name:            "Admin",
			Path:            componentRoot,
			OutputDirectory: componentRoot,
			Package:         models.PackageConfig{Enabled: true, Filename: "admin.zip"},
		}},
	}
	service, _ := packageService(t, filepath.Join(root, "release"))
	plan, err := service.Plan(project, []string{"admin"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(componentRoot, "build")
	if plan.Components[0].SourcePath != want {
		t.Fatalf("source path = %q, want %q", plan.Components[0].SourcePath, want)
	}
}

func TestPackagePlanRejectsRelativeReleaseDirectory(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin")
	service, _ := packageService(t, releaseRoot)

	_, err := service.PlanRequest(project, models.PackageRequest{
		ProjectID:        project.ID,
		ComponentIDs:     []string{"admin"},
		Version:          "1.0.0",
		ReleaseDirectory: "relative-release",
	})
	if err == nil || err.Error() != "release directory must be an absolute path" {
		t.Fatalf("unexpected relative directory error: %v", err)
	}
}

func TestExecuteStopsAfterPackageFailure(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin", "customer", "merchant")
	writeFile(t, filepath.Join(project.Components[0].Path, project.Components[0].OutputDirectory, "index.html"), "admin")
	writeFile(t, filepath.Join(project.Components[2].Path, project.Components[2].OutputDirectory, "index.html"), "merchant")
	service, _ := packageService(t, releaseRoot)
	run, err := service.PrepareRun("run-1", project, []string{"admin", "customer", "merchant"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	final := service.Execute(context.Background(), run, project, []string{"admin", "customer", "merchant"}, false, nil)
	if final.Status != models.PackageRunStatusFailed {
		t.Fatalf("run status = %q, want failed", final.Status)
	}
	if final.Components[0].Status != models.PackageStatusSuccess || final.Components[1].Status != models.PackageStatusFailed || final.Components[2].Status != models.PackageStatusSkipped {
		t.Fatalf("unexpected package states: %+v", final.Components)
	}
}

func TestPackageOneHonorsCancellation(t *testing.T) {
	project, releaseRoot := packageProject(t, "admin")
	component := project.Components[0]
	writeFile(t, filepath.Join(component.Path, component.OutputDirectory, "index.html"), "content")
	service, _ := packageService(t, releaseRoot)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := service.PackageOne(ctx, "run-1", project, component, sourcePath(component), filepath.Join(releaseRoot, "cancelled.zip"), false)
	if result.Status != models.PackageStatusCancelled || result.Success {
		t.Fatalf("cancellation result = %+v", result)
	}
}
