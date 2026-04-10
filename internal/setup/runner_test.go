package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestRunExecutesRepoSetupScriptFromRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	marker := filepath.Join(repoRoot, "setup-ran")
	script := "#!/bin/sh\npwd > pwd.txt\ntouch \"" + marker + "\"\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "setup"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile setup: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := Runner{Stdout: &stdout, Stderr: &stderr}

	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("setup marker missing: %v", err)
	}

	wd, err := os.ReadFile(filepath.Join(repoRoot, "pwd.txt"))
	if err != nil {
		t.Fatalf("ReadFile pwd.txt: %v", err)
	}
	gotPath := strings.TrimSpace(string(wd))
	wantPath, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks(repoRoot): %v", err)
	}
	gotPath, err = filepath.EvalSymlinks(gotPath)
	if err != nil {
		t.Fatalf("EvalSymlinks(script cwd): %v", err)
	}
	if gotPath != wantPath {
		t.Fatalf("script cwd = %q, want %q", gotPath, wantPath)
	}
}

func TestRunRespectsSetupShebang(t *testing.T) {
	repoRoot := t.TempDir()
	outputFile := filepath.Join(repoRoot, "bash.txt")
	script := "#!/usr/bin/env bash\nfunction hello-world() {\n  echo bash > \"" + outputFile + "\"\n}\nhello-world\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "setup"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile setup: %v", err)
	}

	runner := Runner{}
	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile bash.txt: %v", err)
	}
	if strings.TrimSpace(string(data)) != "bash" {
		t.Fatalf("script output = %q, want bash", strings.TrimSpace(string(data)))
	}
}

func TestRunSetsSetupEnvironment(t *testing.T) {
	repoRoot := t.TempDir()
	outputFile := filepath.Join(repoRoot, "env.txt")
	script := "#!/bin/sh\nprintf '%s\\n%s\\n%s\\n' \"$DFL_ROOT\" \"$DOTF\" \"$DFL_DRY_RUN\" > \"" + outputFile + "\"\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "setup"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile setup: %v", err)
	}

	runner := Runner{}
	code, err := runner.Run(runtimectx.Context{RepoRoot: repoRoot, DryRun: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile env.txt: %v", err)
	}

	want := strings.Join([]string{repoRoot, repoRoot, "1", ""}, "\n")
	if string(data) != want {
		t.Fatalf("setup env output = %q, want %q", string(data), want)
	}
}
