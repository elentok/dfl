package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPkgSnapInstallDryRunPrintsPlan(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	binDir := t.TempDir()
	snapPath := filepath.Join(binDir, "snap")
	script := "#!/usr/bin/env bash\nif [ \"$1\" = \"list\" ]; then\n  printf 'Name Version Rev Tracking Publisher Notes\\n'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(snapPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile snap stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"--dry-run", "pkg", "snap", "install", "wl-clip"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "would install snap packages: wl-clip") {
		t.Fatalf("stdout = %q, want snap dry-run output", stdout.String())
	}
}

func TestRunPkgGitHubInstallDryRunPrintsPlan(t *testing.T) {
	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"--dry-run", "pkg", "github", "install", "elentok/colr", "elentok/blf"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "Installing GitHub package elentok/colr") {
		t.Fatalf("stdout = %q, want colr step header", output)
	}
	if !strings.Contains(output, "would install colr ") && !strings.Contains(output, "latest version already installed") {
		t.Fatalf("stdout = %q, want colr dry-run output or latest-version skip output", output)
	}
	if !strings.Contains(output, "would install blf ") && strings.Count(output, "latest version already installed") < 2 {
		t.Fatalf("stdout = %q, want blf dry-run output or latest-version skip output", output)
	}
}

func TestRunPkgGitHubInstallRejectsURLForm(t *testing.T) {
	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"pkg", "github", "install", "github.com/elentok/colr"})
	if err == nil {
		t.Fatalf("Run returned nil error, want validation error")
	}
	if code != 1 {
		t.Fatalf("Run returned code %d, want 1", code)
	}
	if !strings.Contains(err.Error(), `expected owner/repo`) {
		t.Fatalf("err = %q, want owner/repo guidance", err)
	}
}
