package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dfl/internal/runctx"
)

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

	status, message, err := Runner{}.Symlink(runctx.Context{}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	if status != runctx.StatusSkipped {
		t.Fatalf("status = %q, want skipped", status)
	}
	if message != "already exists" {
		t.Fatalf("message = %q, want exact skip output", message)
	}
}

func TestSymlinkDryRunReportsBackupAndLink(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source")
	target := filepath.Join(tempDir, "target")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Symlink(runctx.Context{DryRun: true}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "would back up to") || !strings.Contains(message, "and link") {
		t.Fatalf("message = %q, want verbose dry-run output", message)
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

	status, message, err := Runner{}.Copy(runctx.Context{DryRun: true}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
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

func TestBackupUsesTimestampWhenDefaultBackupExists(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "target")
	if err := os.WriteFile(target, []byte("current"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}
	if err := os.WriteFile(target+".backup", []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	backupPath, err := Runner{}.Backup(runctx.Context{}, target)
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
