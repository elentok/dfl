package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if !strings.Contains(stdout.String(), `Installing fish (core/script)`) {
		t.Fatalf("stdout = %q, want install header", stdout.String())
	}
	if !strings.Contains(stdout.String(), `component "fish": success`) {
		t.Fatalf("stdout = %q, want success summary", stdout.String())
	}
}
