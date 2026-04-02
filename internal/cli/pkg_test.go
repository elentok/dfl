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
