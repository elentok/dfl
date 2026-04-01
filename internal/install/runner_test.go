package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestInstallRunsScriptWithExpectedEnvironment(t *testing.T) {
	repoRoot := t.TempDir()
	componentRoot := filepath.Join(repoRoot, "core", "fish")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	outputFile := filepath.Join(repoRoot, "env.txt")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\n%s\n%s\n' \"$DFL_ROOT\" \"$DFL_COMPONENT_ROOT\" \"$DOTF\" > \"" + outputFile + "\"\n"
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

	want := strings.Join([]string{repoRoot, componentRoot, repoRoot, ""}, "\n")
	if string(data) != want {
		t.Fatalf("script env output = %q, want %q", string(data), want)
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
	if !strings.Contains(stderr.String(), `component "missing" not found`) {
		t.Fatalf("stderr = %q, want missing component output", stderr.String())
	}
}
