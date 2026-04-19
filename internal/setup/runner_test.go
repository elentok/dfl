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
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n%%s\\n%%s\\n%%s\\n%%s\\n%%s\\n' \"$DFL_ROOT\" \"$DFL_COMPONENT_ROOT\" \"$DOTF\" \"$DFL_DRY_RUN\" \"$DFL_LOG\" \"$PATH\" > %q\n", outputFile)
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
	if len(parts) != 6 {
		t.Fatalf("env output lines = %d, want 6", len(parts))
	}
	if parts[0] != repoRoot || parts[1] != filepath.Join(repoRoot, "core") || parts[2] != repoRoot || parts[3] != "1" || parts[4] == "" {
		t.Fatalf("unexpected setup env output: %q", parts[:5])
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	exeDir := filepath.Dir(exe)
	if !strings.HasPrefix(parts[5], exeDir+string(os.PathListSeparator)) && parts[5] != exeDir {
		t.Fatalf("PATH = %q, want it to start with %q", parts[5], exeDir)
	}
}

func TestRunPrintsSetupSummary(t *testing.T) {
	repoRoot := t.TempDir()
	coreDir := filepath.Join(repoRoot, "core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("MkdirAll core: %v", err)
	}

	dflPath := filepath.Join(repoRoot, "dfl")
	dflScript := `#!/bin/sh
case "$1:$2" in
  step:start)
    printf '{"type":"step_start","text":"%s"}\n' "$3" >> "$DFL_LOG"
    ;;
  step:success)
    printf '{"type":"step_end","status":"success","message":"%s"}\n' "$3" >> "$DFL_LOG"
    ;;
  step:error)
    printf '{"type":"step_end","status":"failed","message":"%s"}\n' "$3" >> "$DFL_LOG"
    ;;
esac
`
	if err := os.WriteFile(dflPath, []byte(dflScript), 0o755); err != nil {
		t.Fatalf("WriteFile dfl shim: %v", err)
	}

	setupScript := fmt.Sprintf("#!/bin/sh\nPATH=%q:$PATH\nDFL_LOG=${DFL_LOG:?}\nprintf '{\"type\":\"component_header\",\"text\":\"Installing fish (core/script)\"}\\n' >> \"$DFL_LOG\"\ndfl step start 'create directory X'\ndfl step success 'already exists'\nprintf '{\"type\":\"step_result\",\"text\":\"git-clone this repo\",\"status\":\"failed\",\"message\":\"failed\",\"output\":\"line 1\\\\nline 2\\\\n\"}\\n' >> \"$DFL_LOG\"\n", repoRoot)
	if err := os.WriteFile(filepath.Join(coreDir, "setup"), []byte(setupScript), 0o755); err != nil {
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
	output := stdout.String()
	if !strings.Contains(output, "Setup Summary") {
		t.Fatalf("stdout = %q, want setup summary", output)
	}
	if !strings.Contains(output, "◆ Installing fish (core/script)") {
		t.Fatalf("stdout = %q, want component header", output)
	}
	if !strings.Contains(output, "  ✓ create directory X... already exists") {
		t.Fatalf("stdout = %q, want successful summary line", output)
	}
	if !strings.Contains(output, "  ✗ git-clone this repo... failed") {
		t.Fatalf("stdout = %q, want failed summary line", output)
	}
	if !strings.Contains(output, "    line 1") || !strings.Contains(output, "    line 2") {
		t.Fatalf("stdout = %q, want failure details", output)
	}
	if !strings.Contains(output, "\n\n✗ dfl setup failed: 1 of 2 steps failed\n") {
		t.Fatalf("stdout = %q, want final setup summary line", output)
	}
}
