package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dfl/internal/manifest"
	"dfl/internal/packagemgr"
	runtimectx "dfl/internal/runtime"
)

func TestRunLoadsSetupManifestAndFiltersComponents(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "setup"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	setupFile := `
[[components]]
names = ["fish", "nvim"]
`
	if err := os.WriteFile(filepath.Join(repoRoot, "setup", "default.toml"), []byte(setupFile), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	component := &fakeComponentInstaller{}
	runner := Runner{Stdout: &stdout, ComponentInstaller: component, PackageInstaller: fakePackageInstaller{}, RepoSyncer: fakeRepoSyncer{}, StepExecutor: fakeStepExecutor{}}

	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot, OS: runtimectx.OSMac}, Options{Components: []string{"nvim"}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if strings.Join(component.names, " ") != "nvim" {
		t.Fatalf("component names = %v, want [nvim]", component.names)
	}
}

func TestRunSupportsSkipPackagesAndSkipRepos(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "setup"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	setupFile := `
[[packages]]
manager = "brew"
names = ["gx"]

[[repos]]
name = "notes"
github = "elentok/notes"
path = "~/notes"
`
	if err := os.WriteFile(filepath.Join(repoRoot, "setup", "default.toml"), []byte(setupFile), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	pkgs := &trackingPackageInstaller{}
	repos := &trackingRepoSyncer{}
	runner := Runner{Stdout: &stdout, ComponentInstaller: &fakeComponentInstaller{}, PackageInstaller: pkgs, RepoSyncer: repos, StepExecutor: fakeStepExecutor{}}

	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot, OS: runtimectx.OSMac}, Options{SkipPackages: true, SkipRepos: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if pkgs.called {
		t.Fatal("package installer was called, want skipped")
	}
	if repos.called {
		t.Fatal("repo syncer was called, want skipped")
	}
	if !strings.Contains(stdout.String(), "skipped by flag") {
		t.Fatalf("stdout = %q, want skip message", stdout.String())
	}
}

func TestRunEvaluatesSetupWhenCondition(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "setup"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	setupFile := `
[when]
os = ["linux"]
`
	if err := os.WriteFile(filepath.Join(repoRoot, "setup", "default.toml"), []byte(setupFile), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, ComponentInstaller: &fakeComponentInstaller{}, PackageInstaller: fakePackageInstaller{}, RepoSyncer: fakeRepoSyncer{}, StepExecutor: fakeStepExecutor{}}

	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot, OS: runtimectx.OSMac}, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "setup does not apply to this machine") {
		t.Fatalf("stdout = %q, want setup skip output", stdout.String())
	}
}

func TestRunExecutesMatchingSetupStepsWithDryRun(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "setup"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	setupFile := `
[[steps]]
name = "cache"
run = "echo hello"

[[steps]]
name = "mac only"
os = ["mac"]
run = "echo mac"
`
	if err := os.WriteFile(filepath.Join(repoRoot, "setup", "default.toml"), []byte(setupFile), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	steps := &trackingStepExecutor{}
	runner := Runner{Stdout: &stdout, ComponentInstaller: &fakeComponentInstaller{}, PackageInstaller: fakePackageInstaller{}, RepoSyncer: fakeRepoSyncer{}, StepExecutor: steps}

	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot, OS: runtimectx.OSLinux, DryRun: true}, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if len(steps.steps) != 1 || steps.steps[0] != "cache" {
		t.Fatalf("executed steps = %v, want [cache]", steps.steps)
	}
	if !strings.Contains(stdout.String(), "not applicable on this machine") {
		t.Fatalf("stdout = %q, want skipped step output", stdout.String())
	}
}

type fakePackageInstaller struct{}

func (fakePackageInstaller) Install(runtimectx.Context, string, packagemgr.InstallOptions) (int, error) {
	return 0, nil
}

type trackingPackageInstaller struct{ called bool }

func (t *trackingPackageInstaller) Install(runtimectx.Context, string, packagemgr.InstallOptions) (int, error) {
	t.called = true
	return 0, nil
}

type fakeComponentInstaller struct{ names []string }

func (f *fakeComponentInstaller) Install(_ runtimectx.Context, names []string) (int, error) {
	f.names = append([]string(nil), names...)
	return 0, nil
}

type fakeRepoSyncer struct{}

func (fakeRepoSyncer) Sync(runtimectx.Context, manifest.RepoDefaults, manifest.RepoSpec) (runtimectx.ResultStatus, string, error) {
	return runtimectx.StatusSuccess, "done", nil
}

type trackingRepoSyncer struct{ called bool }

func (t *trackingRepoSyncer) Sync(runtimectx.Context, manifest.RepoDefaults, manifest.RepoSpec) (runtimectx.ResultStatus, string, error) {
	t.called = true
	return runtimectx.StatusSuccess, "done", nil
}

type fakeStepExecutor struct{}

func (fakeStepExecutor) Execute(runtimectx.Context, string, manifest.StepSpec) (runtimectx.ResultStatus, string, error) {
	return runtimectx.StatusSuccess, "done", nil
}

type trackingStepExecutor struct{ steps []string }

func (t *trackingStepExecutor) Execute(_ runtimectx.Context, _ string, step manifest.StepSpec) (runtimectx.ResultStatus, string, error) {
	t.steps = append(t.steps, step.Name)
	return runtimectx.StatusSuccess, "done", nil
}
