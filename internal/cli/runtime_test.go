package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	code, err := app.Run([]string{"shell", "--dry-run", "demo", "echo", "hello"})
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

func TestRunStepEndShortcutFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "success", args: []string{"step-end", "--success", "done"}, want: "success: done"},
		{name: "skip", args: []string{"step-end", "--skip", "dry-run"}, want: "skipped: dry-run"},
		{name: "error", args: []string{"step-end", "--error", "failed"}, want: "failed: failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			app.SetStdout(&stdout)
			app.SetStderr(&stderr)

			code, err := app.Run(tt.args)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if code != 0 {
				t.Fatalf("Run returned code %d, want 0", code)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}
