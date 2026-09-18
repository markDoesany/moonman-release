package packaging

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"release-launcher/backend/models"
	"release-launcher/backend/pathutil"
)

func TestResolveFilenameExpandsAllTokensAndAddsZip(t *testing.T) {
	stamp := time.Date(2026, time.September, 18, 14, 5, 6, 0, time.UTC)
	got, err := ResolveFilename("{project}-{component}-v{version}-{date}-{time}-{datetime}", "Lokal/Store", "Admin", "1.2.3", stamp)
	if err != nil {
		t.Fatal(err)
	}
	want := "Lokal_Store-Admin-v1.2.3-20260918-140506-20260918-140506.zip"
	if got != want {
		t.Fatalf("ResolveFilename() = %q, want %q", got, want)
	}
}

func TestResolveFilenameKeepsStaticNamesAndRejectsInvalidNames(t *testing.T) {
	static, err := ResolveFilename("lokalstore-admin.zip", "Project", "Admin", "1.0.0", time.Now())
	if err != nil || static != "lokalstore-admin.zip" {
		t.Fatalf("static filename = %q, error = %v", static, err)
	}
	for _, name := range []string{"bad/name", "bad:name", "CON.txt", "bad. ", "{unknown}"} {
		if _, err := ResolveFilename(name, "Project", "Admin", "1.0.0", time.Now()); err == nil {
			t.Errorf("ResolveFilename(%q) accepted invalid name", name)
		}
	}
}

func TestPlanRequestUsesOverridesAndRejectsDuplicates(t *testing.T) {
	root := t.TempDir()
	project := models.Project{ID: "p", Name: "Project", Components: []models.Component{
		{ID: "a", Name: "Admin", Path: filepath.Join(root, "a"), Package: models.PackageConfig{Enabled: true, Filename: "admin.zip"}},
		{ID: "b", Name: "Customer", Path: filepath.Join(root, "b"), Package: models.PackageConfig{Enabled: true, Filename: "customer.zip"}},
	}}
	service := NewService(nil, nil, filepath.Join(root, "release"))
	plan, err := service.PlanRequest(project, models.PackageRequest{ComponentIDs: []string{"a", "b"}, Version: "1.0.0", FilenameTemplate: "{project}-{component}-{date}"})
	if err != nil || !strings.HasSuffix(plan.Components[0].ResolvedFilename, ".zip") {
		t.Fatalf("dynamic plan = %+v, error = %v", plan, err)
	}
	_, err = service.PlanRequest(project, models.PackageRequest{ComponentIDs: []string{"a", "b"}, Version: "1.0.0", PackageNames: map[string]string{"a": "same.zip", "b": "same.zip"}})
	if err == nil {
		t.Fatal("duplicate package names were accepted")
	}
	plan, err = service.PlanRequest(project, models.PackageRequest{ComponentIDs: []string{"a"}, Version: "1.0.0", FilenameTemplate: "{invalid}", PackageNames: map[string]string{"a": "approved.zip"}})
	if err != nil || plan.Components[0].ResolvedFilename != "approved.zip" {
		t.Fatalf("per-component override did not take precedence: %+v, error = %v", plan, err)
	}
}

func TestRelativeIfInsideAndResolveSupportBothOutputForms(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "dist")
	if got := pathutil.RelativeIfInside(base, inside); got != "dist" {
		t.Fatalf("relative path = %q, want dist", got)
	}
	outside := filepath.Join(t.TempDir(), "dist")
	if got := pathutil.RelativeIfInside(base, outside); got != outside {
		t.Fatalf("outside path = %q, want %q", got, outside)
	}
	if got := pathutil.Resolve(base, inside); got != inside {
		t.Fatalf("absolute output resolve = %q", got)
	}
	if got := pathutil.Resolve(base, "dist"); got != filepath.Join(base, "dist") {
		t.Fatalf("relative output resolve = %q", got)
	}
}
