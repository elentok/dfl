package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithNoArgsPrintsUsage(t *testing.T) {
	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run(nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout = %q, want usage output", stdout.String())
	}
}

func TestRunWithUnknownCommandPrintsError(t *testing.T) {
	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"wat"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 2 {
		t.Fatalf("Run returned code %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "wat"`) {
		t.Fatalf("stderr = %q, want unknown command output", stderr.String())
	}
}

func TestRunInstallAliasExecutesComponentInstall(t *testing.T) {
	repoRoot := t.TempDir()
	componentRoot := filepath.Join(repoRoot, "core", "fish")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(componentRoot, "install"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile install: %v", err)
	}

	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"i", "fish"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), `component "fish": success`) {
		t.Fatalf("stdout = %q, want success summary", stdout.String())
	}
}

func TestRunOSPredicateReturnsSuccessForCurrentPlatform(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	code, err := app.Run([]string{"os", "is-mac"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0 on darwin", code)
	}
}

func TestRunHasCommandReturnsSuccessForSh(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	code, err := app.Run([]string{"has-command", "sh"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
}

func TestRunShellDryRunPrintsPlannedCommand(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"shell", "--dry-run", "demo", "--", "echo", "hello"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "DRY-RUN: echo hello") {
		t.Fatalf("stdout = %q, want dry-run output", stdout.String())
	}
}

func TestRunBackupDryRunPrintsDestination(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	target := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"backup", "--dry-run", target})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "would move to") {
		t.Fatalf("stdout = %q, want dry-run backup output", stdout.String())
	}
}
