package runtimecmd

import (
	"bytes"
	"os"
	"os/exec"
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

func TestAskUsesDefaultForEmptyReply(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := Runner{
		Stdin:  strings.NewReader("\n"),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	reply, err := runner.Ask("What's up?", "ok")
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	if reply != "ok" {
		t.Fatalf("reply = %q, want default reply", reply)
	}
	if !strings.Contains(stderr.String(), "What's up? [ok] ") {
		t.Fatalf("stderr = %q, want prompt", stderr.String())
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
	if !strings.Contains(output, "○ dry-run") {
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

	status, message, err := Runner{}.Symlink(runtimectx.Context{}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	if status != runtimectx.StatusSkipped {
		t.Fatalf("status = %q, want skipped", status)
	}
	if !strings.Contains(message, "already exists at") {
		t.Fatalf("message = %q, want verbose skip output", message)
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

	status, message, err := Runner{}.Symlink(runtimectx.Context{DryRun: true}, tempDir, source, target)
	if err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	if status != runtimectx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "would back up to") || !strings.Contains(message, "and link") {
		t.Fatalf("message = %q, want verbose dry-run output", message)
	}
}

func TestGitCloneSkipsWhenAlreadyClonedFromSameOrigin(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	status, message, err := Runner{}.GitClone(runtimectx.Context{}, origin, target, false)
	if err != nil {
		t.Fatalf("GitClone returned error: %v", err)
	}
	if status != runtimectx.StatusSkipped {
		t.Fatalf("status = %q, want skipped", status)
	}
	if !strings.Contains(message, "already cloned at") {
		t.Fatalf("message = %q, want skip output", message)
	}
	_ = worktree
}

func TestGitCloneUpdatesWhenRequested(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	writeFile(t, worktree, "README.md", "updated\n")
	gitRun(t, worktree, "add", "README.md")
	gitRun(t, worktree, "commit", "-m", "update readme")
	gitRun(t, worktree, "push", "origin", "HEAD")

	status, message, err := Runner{}.GitClone(runtimectx.Context{}, origin, target, true)
	if err != nil {
		t.Fatalf("GitClone returned error: %v", err)
	}
	if status != runtimectx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "already cloned, updated") {
		t.Fatalf("message = %q, want update output", message)
	}

	data, err := os.ReadFile(filepath.Join(target, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "updated\n" {
		t.Fatalf("README.md = %q, want updated content", string(data))
	}
}

func createOriginAndWorktree(t *testing.T) (string, string) {
	t.Helper()

	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	worktree := filepath.Join(base, "worktree")

	gitRun(t, base, "init", "--bare", origin)
	gitRun(t, base, "clone", origin, worktree)
	writeFile(t, worktree, "README.md", "hello\n")
	gitRun(t, worktree, "add", "README.md")
	gitRun(t, worktree, "commit", "-m", "initial commit")
	gitRun(t, worktree, "push", "origin", "HEAD")

	return origin, worktree
}

func cloneRepo(t *testing.T, origin, name string) string {
	t.Helper()

	target := filepath.Join(t.TempDir(), name)
	gitRun(t, filepath.Dir(target), "clone", origin, target)
	return target
}

func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}
