package runtimecmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestHasCommandFindsExecutable(t *testing.T) {
	ops := Runner{}
	found, err := ops.HasCommand("sh")
	if err != nil {
		t.Fatalf("HasCommand returned error: %v", err)
	}
	if !found {
		t.Fatal("HasCommand returned false, want true for sh")
	}
}

func TestShellDryRunSkipsExecution(t *testing.T) {
	var stdout bytes.Buffer
	ops := Runner{Stdout: &stdout}

	code, err := ops.Shell(runtimectx.Context{DryRun: true}, "demo", []string{"echo", "hi"})
	if err != nil {
		t.Fatalf("Shell returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Shell returned code %d, want 0", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "◆ demo") {
		t.Fatalf("stdout = %q, want step start", output)
	}
	if !strings.Contains(output, "DRY-RUN: echo hi") {
		t.Fatalf("stdout = %q, want dry-run command", output)
	}
	if !strings.Contains(output, "skipped: dry-run") {
		t.Fatalf("stdout = %q, want skipped step end", output)
	}
}

func TestSymlinkSkipsWhenAlreadyCorrect(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source")
	target := filepath.Join(tempDir, "target")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	status, _, err := Runner{}.Symlink(runtimectx.Context{}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	if status != runtimectx.StatusSkipped {
		t.Fatalf("status = %q, want skipped", status)
	}
}

func TestBackupUsesTimestampWhenDefaultBackupExists(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "target")
	if err := os.WriteFile(target, []byte("current"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}
	if err := os.WriteFile(target+".backup", []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	backupPath, err := Runner{}.Backup(runtimectx.Context{}, target)
	if err != nil {
		t.Fatalf("Backup returned error: %v", err)
	}
	if !strings.HasPrefix(backupPath, target+".backup.") {
		t.Fatalf("backupPath = %q, want timestamped backup", backupPath)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("Stat timestamped backup: %v", err)
	}
}

func TestCopyDryRunDoesNotModifyTarget(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source")
	target := filepath.Join(tempDir, "target")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Copy(runtimectx.Context{DryRun: true}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if status != runtimectx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "would copy") {
		t.Fatalf("message = %q, want dry-run copy output", message)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("target contents = %q, want unchanged", string(data))
	}
}
