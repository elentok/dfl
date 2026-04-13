package setup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestRunExecutesRepoSetupScriptFromRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "core"), 0o755); err != nil {
		t.Fatalf("MkdirAll core: %v", err)
	}
	marker := filepath.Join(repoRoot, "setup-ran")
	script := fmt.Sprintf("#!/bin/sh\npwd > pwd.txt\ntouch %q\n", marker)
	if err := os.WriteFile(filepath.Join(repoRoot, "core", "setup"), []byte(script), 0o755); err != nil {
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
	if err := os.MkdirAll(filepath.Join(repoRoot, "core"), 0o755); err != nil {
		t.Fatalf("MkdirAll core: %v", err)
	}
	outputFile := filepath.Join(repoRoot, "bash.txt")
	script := fmt.Sprintf("#!/usr/bin/env bash\nfunction hello-world() {\n  echo bash > %q\n}\nhello-world\n", outputFile)
	if err := os.WriteFile(filepath.Join(repoRoot, "core", "setup"), []byte(script), 0o755); err != nil {
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
	if err := os.MkdirAll(filepath.Join(repoRoot, "core"), 0o755); err != nil {
		t.Fatalf("MkdirAll core: %v", err)
	}
	outputFile := filepath.Join(repoRoot, "env.txt")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n%%s\\n%%s\\n%%s\\n%%s\\n' \"$DFL_ROOT\" \"$DFL_COMPONENT_ROOT\" \"$DOTF\" \"$DFL_DRY_RUN\" \"$PATH\" > %q\n", outputFile)
	if err := os.WriteFile(filepath.Join(repoRoot, "core", "setup"), []byte(script), 0o755); err != nil {
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

	parts := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(parts) != 5 {
		t.Fatalf("env output lines = %d, want 5", len(parts))
	}
	if parts[0] != repoRoot || parts[1] != filepath.Join(repoRoot, "core") || parts[2] != repoRoot || parts[3] != "1" {
		t.Fatalf("unexpected setup env output: %q", parts[:4])
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	exeDir := filepath.Dir(exe)
	if !strings.HasPrefix(parts[4], exeDir+string(os.PathListSeparator)) && parts[4] != exeDir {
		t.Fatalf("PATH = %q, want it to start with %q", parts[4], exeDir)
	}
}
