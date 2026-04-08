package packagemgr

import (
	"bytes"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestInstallBrewEnsuresTapAndInstallsMissingPackages(t *testing.T) {
	exec := &FakeExecutor{
		Outputs: map[string][]byte{
			commandKey("brew", "list", "--full-name"): []byte("git\n"),
			commandKey("brew", "tap"):                 []byte("homebrew/core\n"),
		},
	}

	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, Exec: exec}

	code, err := runner.Install(runtimectx.Context{}, "brew", InstallOptions{
		Packages: []string{"git", "gx"},
		Tap:      "elentok/stuff",
	})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Install returned code %d, want 0", code)
	}
	if len(exec.Runs) != 2 {
		t.Fatalf("run count = %d, want 2", len(exec.Runs))
	}
	if exec.Runs[0].Name != "brew" || strings.Join(exec.Runs[0].Args, " ") != "tap elentok/stuff" {
		t.Fatalf("first run = %#v, want brew tap", exec.Runs[0])
	}
	if exec.Runs[1].Name != "brew" || strings.Join(exec.Runs[1].Args, " ") != "install gx" {
		t.Fatalf("second run = %#v, want brew install gx", exec.Runs[1])
	}
}

func TestInstallBrewDryRunPrintsPlannedActions(t *testing.T) {
	exec := &FakeExecutor{
		Outputs: map[string][]byte{
			commandKey("brew", "list", "--full-name"): []byte(""),
		},
	}

	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, Exec: exec}

	code, err := runner.Install(runtimectx.Context{DryRun: true}, "brew", InstallOptions{
		Packages: []string{"gx"},
		Tap:      "elentok/stuff",
	})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Install returned code %d, want 0", code)
	}
	if len(exec.Runs) != 0 {
		t.Fatalf("run count = %d, want 0 in dry-run", len(exec.Runs))
	}
	if !strings.Contains(stdout.String(), "would ensure tap elentok/stuff") {
		t.Fatalf("stdout = %q, want tap dry-run output", stdout.String())
	}
}

func TestInstallNPMShapesGlobalInstallCommand(t *testing.T) {
	exec := &FakeExecutor{
		Outputs: map[string][]byte{
			commandKey("npm", "list", "-g", "--depth=0", "--json"): []byte(`{"dependencies":{"fx":{}}}`),
		},
	}

	runner := Runner{Exec: exec}
	code, err := runner.Install(runtimectx.Context{}, "npm", InstallOptions{Packages: []string{"fx", "json"}})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Install returned code %d, want 0", code)
	}
	if len(exec.Runs) != 1 {
		t.Fatalf("run count = %d, want 1", len(exec.Runs))
	}
	if exec.Runs[0].Name != "npm" || strings.Join(exec.Runs[0].Args, " ") != "install -g json" {
		t.Fatalf("run = %#v, want npm install -g json", exec.Runs[0])
	}
}

func TestInstallSkipsWhenAllPackagesPresent(t *testing.T) {
	exec := &FakeExecutor{
		Outputs: map[string][]byte{
			commandKey("pipx", "list", "--short"): []byte("httpie 3.2.4\n"),
		},
	}

	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, Exec: exec}
	code, err := runner.Install(runtimectx.Context{}, "pipx", InstallOptions{Packages: []string{"httpie"}})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Install returned code %d, want 0", code)
	}
	if len(exec.Runs) != 0 {
		t.Fatalf("run count = %d, want 0", len(exec.Runs))
	}
	if !strings.Contains(stdout.String(), "skipped: already installed") {
		t.Fatalf("stdout = %q, want skipped output", stdout.String())
	}
}
