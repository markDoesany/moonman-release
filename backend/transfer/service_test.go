package transfer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"release-launcher/backend/logging"
	"release-launcher/backend/models"
	"release-launcher/backend/packaging"
)

type fakeRunner struct {
	output    string
	exitCode  int
	cancelled bool
	args      []string
}

func (r *fakeRunner) Run(ctx context.Context, _ string, args []string, emit func(string, string)) processOutcome {
	r.args = args
	started := time.Now()
	if r.cancelled || ctx.Err() != nil {
		return processOutcome{StartTime: started, EndTime: time.Now(), ExitCode: -1, Cancelled: true, Err: context.Canceled, Technical: context.Canceled}
	}
	if emit != nil {
		emit("stdout", r.output)
	}
	return processOutcome{StartTime: started, EndTime: time.Now(), ExitCode: r.exitCode}
}

func transferFixture(t *testing.T) (models.Project, string, *packaging.Service) {
	t.Helper()
	root := t.TempDir()
	releaseRoot := filepath.Join(root, "releases")
	componentRoot := filepath.Join(root, "admin")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	packager := packaging.NewService(logging.New(filepath.Join(base, "app.log")), logging.NewJSONLWriter(filepath.Join(base, "packaging.jsonl")), releaseRoot)
	project := models.Project{ID: "lokalstore", Name: "LokalStore", Components: []models.Component{{ID: "admin", Name: "Admin", Path: componentRoot, OutputDirectory: "build", Package: models.PackageConfig{Enabled: true, Filename: "admin.zip"}}}}
	plan, err := packager.Plan(project, []string{"admin"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(plan.Components[0].PackagePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.Components[0].PackagePath, []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	return project, plan.Components[0].PackagePath, packager
}

func TestSendOneRequiresDaliSuccessConfirmation(t *testing.T) {
	project, packagePath, packager := transferFixture(t)
	runner := &fakeRunner{output: "Connecting...\n✓ File sent successfully!"}
	service := newServiceWithRunner(logging.New(filepath.Join(t.TempDir(), "app.log")), nil, packager, runner, func(string) (string, error) { return "dali.exe", nil })
	result := service.SendOne(context.Background(), "run-1", project, project.Components[0], packagePath, models.DaliConfig{Executable: "dali", PeerName: "DevOps"}, nil)
	if !result.Success || result.Status != models.TransferStatusSuccess {
		t.Fatalf("unexpected transfer result: %+v", result)
	}
	if strings.Join(runner.args, " ") != "send file="+packagePath+" for=DevOps" {
		t.Fatalf("unexpected Dali arguments: %v", runner.args)
	}
}

func TestSendOneRejectsUnconfirmedSuccessfulExit(t *testing.T) {
	project, packagePath, packager := transferFixture(t)
	runner := &fakeRunner{output: "No peers found."}
	service := newServiceWithRunner(nil, nil, packager, runner, func(string) (string, error) { return "dali.exe", nil })
	result := service.SendOne(context.Background(), "run-1", project, project.Components[0], packagePath, models.DaliConfig{Executable: "dali", Auto: true}, nil)
	if result.Success || result.Status != models.TransferStatusFailed || result.Error == "" {
		t.Fatalf("unexpected unconfirmed transfer result: %+v", result)
	}
}

func TestExecuteStopsAfterTransferFailure(t *testing.T) {
	project, _, packager := transferFixture(t)
	project.Components = append(project.Components, project.Components[0])
	project.Components[1].ID = "customer"
	project.Components[1].Name = "Customer"
	project.Components[1].Package.Filename = "customer.zip"
	plan, err := packager.Plan(project, []string{"admin", "customer"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range plan.Components {
		if item.Selected {
			if err := os.WriteFile(item.PackagePath, []byte("zip"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	runner := &fakeRunner{output: "No peers found."}
	service := newServiceWithRunner(nil, nil, packager, runner, func(string) (string, error) { return "dali.exe", nil })
	run, err := service.PrepareRun("run-1", project, models.TransferRequest{ProjectID: project.ID, ComponentIDs: []string{"admin", "customer"}, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	final := service.Execute(context.Background(), run, project, models.TransferRequest{ProjectID: project.ID, ComponentIDs: []string{"admin", "customer"}, Version: "1.0.0"}, models.DaliConfig{Executable: "dali", Auto: true}, nil)
	if final.Status != models.TransferRunStatusFailed || final.Components[1].Status != models.TransferStatusSkipped {
		t.Fatalf("unexpected transfer run: %+v", final)
	}
}
