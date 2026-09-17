package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	base := t.TempDir()
	return Paths{
		BaseDir:        base,
		ConfigDir:      filepath.Join(base, "configs"),
		ConfigFile:     filepath.Join(base, "configs", "projects.yaml"),
		BackupFile:     filepath.Join(base, "configs", "projects.yaml.bak"),
		LogDir:         filepath.Join(base, "logs"),
		LogFile:        filepath.Join(base, "logs", "app.log"),
		BuildLogFile:   filepath.Join(base, "logs", "builds.jsonl"),
		PackageLogFile: filepath.Join(base, "logs", "packaging.jsonl"),
		ReleaseDir:     filepath.Join(base, "releases"),
	}
}

func TestLoadSeedsSampleConfiguration(t *testing.T) {
	paths := testPaths(t)
	service := NewService(paths, logging.New(paths.LogFile))

	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	projects, err := service.Projects()
	if err != nil {
		t.Fatalf("Projects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != "lokalstore" {
		t.Fatalf("unexpected seeded projects: %+v", projects)
	}
	if _, err := os.Stat(paths.ConfigFile); err != nil {
		t.Fatalf("seeded configuration was not written: %v", err)
	}
}

func TestLoadSupportsLegacyFlatPackageFields(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := []byte("projects:\n  - id: legacy\n    name: Legacy\n    components:\n      - id: web\n        name: Web\n        path: C:/Projects/Legacy/Web\n        build_command: npm run build\n        output_directory: build\n        package_enabled: true\n        package_filename: legacy-web.zip\n")
	if err := os.WriteFile(paths.ConfigFile, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(paths, logging.New(paths.LogFile))
	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	project, err := service.Project("legacy")
	if err != nil || !project.Components[0].Package.Enabled || project.Components[0].Package.Filename != "legacy-web.zip" {
		t.Fatalf("legacy package fields were not normalized: %+v, error = %v", project, err)
	}
}

func TestSaveProjectGeneratesIDsAndCreatesBackup(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ConfigFile, []byte("projects: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(paths, logging.New(paths.LogFile))
	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	saved, err := service.SaveProject(models.Project{
		Name: "Donation Platform",
		Components: []models.Component{{
			Name:            "Admin Portal",
			Path:            `C:\Projects\Donation\Admin`,
			BuildCommand:    "npm run build",
			OutputDirectory: "build",
			Package:         models.PackageConfig{Enabled: true, Filename: "donation-admin.zip"},
		}},
	})
	if err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}
	if saved.ID != "donation-platform" || saved.Components[0].ID != "admin-portal" {
		t.Fatalf("unexpected generated IDs: %+v", saved)
	}
	if _, err := os.Stat(paths.BackupFile); err != nil {
		t.Fatalf("backup was not created: %v", err)
	}
}

func TestLoadRecoversValidBackup(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := []byte("projects:\n  - id: safe\n    name: Safe\n    components: []\n")
	if err := os.WriteFile(paths.ConfigFile, []byte("projects: [invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.BackupFile, valid, 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService(paths, logging.New(paths.LogFile))
	if err := service.Load(); err != nil {
		t.Fatalf("Load() should recover backup, got %v", err)
	}
	project, err := service.Project("safe")
	if err != nil || project.Name != "Safe" {
		t.Fatalf("recovered project = %+v, error = %v", project, err)
	}
	contents, err := os.ReadFile(paths.ConfigFile)
	if err != nil || !strings.Contains(string(contents), "id: safe") {
		t.Fatalf("primary configuration was not restored: %s", contents)
	}
}

func TestValidateReportsRequiredFields(t *testing.T) {
	issues := Validate([]models.Project{{
		ID:   "project",
		Name: "Project",
		Components: []models.Component{{
			ID:      "component",
			Name:    "Component",
			Package: models.PackageConfig{Enabled: true},
		}},
	}})

	if len(issues) != 4 {
		t.Fatalf("expected 4 validation issues, got %d: %+v", len(issues), issues)
	}
	for _, issue := range issues {
		if issue.Message == "" || issue.Field == "" {
			t.Fatalf("incomplete validation issue: %+v", issue)
		}
	}
}
