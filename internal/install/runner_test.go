package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/setuplog"
)

func TestInstallRunsScriptWithExpectedEnvironment(t *testing.T) {
	repoRoot := t.TempDir()
	componentRoot := filepath.Join(repoRoot, "core", "fish")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	outputFile := filepath.Join(repoRoot, "env.txt")
	script := fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\nprintf '%%s\\n%%s\\n%%s\\n%%s\\n' \"$DFL_ROOT\" \"$DFL_COMPONENT_ROOT\" \"$DOTF\" \"$PATH\" > %q\n", outputFile)
	if err := os.WriteFile(filepath.Join(componentRoot, "install"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile install: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := Runner{Stdout: &stdout, Stderr: &stderr}

	code, err := runner.Install(runtimectx.Context{RepoRoot: repoRoot}, []string{"fish"})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Install returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile env output: %v", err)
	}

	parts := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(parts) != 4 {
		t.Fatalf("env output lines = %d, want 4", len(parts))
	}
	if parts[0] != repoRoot || parts[1] != componentRoot || parts[2] != repoRoot {
		t.Fatalf("script env output = %q, want repo/component roots", parts[:3])
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	exeDir := filepath.Dir(exe)
	if !strings.HasPrefix(parts[3], exeDir+string(os.PathListSeparator)) && parts[3] != exeDir {
		t.Fatalf("PATH = %q, want it to start with %q", parts[3], exeDir)
	}
}

func TestInstallReturnsErrorForMissingComponent(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := Runner{Stdout: &stdout, Stderr: &stderr}

	code, err := runner.Install(runtimectx.Context{RepoRoot: t.TempDir()}, []string{"missing"})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 1 {
		t.Fatalf("Install returned code %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `✘ component "missing" not found`) {
		t.Fatalf("stderr = %q, want missing component output", stderr.String())
	}
}

func TestInstallLogsComponentHeader(t *testing.T) {
	repoRoot := t.TempDir()
	componentRoot := filepath.Join(repoRoot, "core", "fish")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(componentRoot, "install"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile install: %v", err)
	}

	logPath := filepath.Join(t.TempDir(), "setup.jsonl")
	t.Setenv("DFL_LOG", logPath)

	runner := Runner{}
	code, err := runner.Install(runtimectx.Context{RepoRoot: repoRoot}, []string{"fish"})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Install returned code %d, want 0", code)
	}

	steps, err := setuplog.Read(logPath)
	if err != nil {
		t.Fatalf("Read log: %v", err)
	}
	if len(steps) == 0 {
		t.Fatal("steps is empty, want component header record")
	}
	if !steps[0].IsHeader || steps[0].Text != "Installing fish (core/script)" {
		t.Fatalf("steps[0] = %#v, want first record to be component header", steps[0])
	}
}
