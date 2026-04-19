package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHasCommandReturnsSuccessForSh(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	code, err := app.Run([]string{"has-command", "sh"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
}

func TestRunAskPrintsReplyToStdout(t *testing.T) {
	app := NewApp()
	app.SetStdin(strings.NewReader("hello\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"ask", "What's up?"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stdout.String() != "hello\n" {
		t.Fatalf("stdout = %q, want reply on stdout", stdout.String())
	}
	if !strings.Contains(stderr.String(), "What's up? ") {
		t.Fatalf("stderr = %q, want prompt on stderr", stderr.String())
	}
}

func TestRunAskUsesDefaultForEmptyReply(t *testing.T) {
	app := NewApp()
	app.SetStdin(strings.NewReader("\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"ask", "What's up?", "ok"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stdout.String() != "ok\n" {
		t.Fatalf("stdout = %q, want default reply on stdout", stdout.String())
	}
	if !strings.Contains(stderr.String(), "What's up? [ok] ") {
		t.Fatalf("stderr = %q, want prompt with default on stderr", stderr.String())
	}
}

func TestRunShellDryRunPrintsPlannedCommand(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"shell", "--dry-run", "demo", "echo", "hello"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "DRY-RUN: echo hello") {
		t.Fatalf("stdout = %q, want dry-run output", stdout.String())
	}
}

func TestRunBackupDryRunPrintsDestination(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	target := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"backup", "--dry-run", target})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "would move to") {
		t.Fatalf("stdout = %q, want dry-run backup output", stdout.String())
	}
}

func TestRunStepStatusCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "success", args: []string{"step", "success", "done"}, want: "✓ done"},
		{name: "skip", args: []string{"step", "skip", "dry-run"}, want: "○ dry-run"},
		{name: "error", args: []string{"step", "error", "failed"}, want: "✗ failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			app.SetStdout(&stdout)
			app.SetStderr(&stderr)

			code, err := app.Run(tt.args)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if code != 0 {
				t.Fatalf("Run returned code %d, want 0", code)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestRunStepStatusCommandsUseDefaultMessages(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "success", args: []string{"step", "success"}, want: "✓ done"},
		{name: "error", args: []string{"step", "error"}, want: "✗ failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			app.SetStdout(&stdout)
			app.SetStderr(&stderr)

			code, err := app.Run(tt.args)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if code != 0 {
				t.Fatalf("Run returned code %d, want 0", code)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestRunStepStartCommand(t *testing.T) {
	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"step", "start", "demo"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "◆ demo") {
		t.Fatalf("stdout = %q, want step header", stdout.String())
	}
}

func TestRunSymlinkDryRunIsVerbose(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)
	target := filepath.Join(t.TempDir(), "tmux.conf")

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"symlink", "--dry-run", "tmux.conf", target})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Linking tmux.conf") {
		t.Fatalf("stdout = %q, want linking header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "=> "+target) {
		t.Fatalf("stdout = %q, want target line", stdout.String())
	}
	if !strings.Contains(stdout.String(), "would link") {
		t.Fatalf("stdout = %q, want verbose symlink message", stdout.String())
	}
}

func TestRunMkdirDryRunIsVerbose(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)
	target := filepath.Join(t.TempDir(), "config")

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"mkdir", "--dry-run", target})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Creating "+target) {
		t.Fatalf("stdout = %q, want mkdir header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "would create") {
		t.Fatalf("stdout = %q, want verbose mkdir message", stdout.String())
	}
}

func TestRunGitCloneDryRunIsVerbose(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)
	target := filepath.Join(t.TempDir(), "repo")

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"git-clone", "--dry-run", "https://github.com/elentok/notes.git", target})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Cloning https://github.com/elentok/notes.git") {
		t.Fatalf("stdout = %q, want clone header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "=> "+target) {
		t.Fatalf("stdout = %q, want target line", stdout.String())
	}
	if !strings.Contains(stdout.String(), "would clone") {
		t.Fatalf("stdout = %q, want dry-run clone message", stdout.String())
	}
}

func TestRunGitCloneDryRunInheritsSSHForGitHubSlug(t *testing.T) {
	repoRoot := initRepoWithOrigin(t, "git@github.com:elentok/dotfiles.git")
	t.Setenv("DFL_ROOT", repoRoot)
	target := filepath.Join(t.TempDir(), "repo")

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"git-clone", "--dry-run", "elentok/stuff.nvim", target})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Cloning elentok/stuff.nvim") {
		t.Fatalf("stdout = %q, want clone header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "would clone git@github.com:elentok/stuff.nvim.git") {
		t.Fatalf("stdout = %q, want inherited ssh clone url", stdout.String())
	}
}

func TestRunGitCloneUpdatePrintsFailedPullStatus(t *testing.T) {
	repoRoot := initRepoWithOrigin(t, filepath.Join(t.TempDir(), "dotfiles-origin.git"))
	origin, worktree := createOriginAndWorktreeForCLI(t)
	target := cloneRepoForCLI(t, origin, "target")
	t.Setenv("DFL_ROOT", repoRoot)

	writeFileForCLI(t, worktree, "README.md", "remote change\n")
	runGitWithEnv(t, worktree, []string{
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	}, "add", "README.md")
	runGitWithEnv(t, worktree, []string{
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	}, "commit", "-m", "remote update")
	runGitWithEnv(t, worktree, []string{
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	}, "push", "origin", "HEAD")
	writeFileForCLI(t, target, "README.md", "local change\n")

	app := NewApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"git-clone", "--update", origin, target})
	if err == nil {
		t.Fatal("Run returned nil error, want pull failure")
	}
	if code != 1 {
		t.Fatalf("Run returned code %d, want 1", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "✗ failed to pull") {
		t.Fatalf("stdout = %q, want failed pull status", stdout.String())
	}
}

func initRepoWithOrigin(t *testing.T, origin string) string {
	t.Helper()

	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init")
	runGit(t, repoRoot, "remote", "add", "origin", origin)
	return repoRoot
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}

func runGitWithEnv(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}

func createOriginAndWorktreeForCLI(t *testing.T) (string, string) {
	t.Helper()

	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	worktree := filepath.Join(base, "worktree")

	runGit(t, base, "init", "--bare", origin)
	runGit(t, base, "clone", origin, worktree)
	writeFileForCLI(t, worktree, "README.md", "hello\n")
	runGitWithEnv(t, worktree, []string{
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	}, "add", "README.md")
	runGitWithEnv(t, worktree, []string{
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	}, "commit", "-m", "initial commit")
	runGitWithEnv(t, worktree, []string{
		"GIT_AUTHOR_NAME=DFL Test",
		"GIT_AUTHOR_EMAIL=dfl@example.com",
		"GIT_COMMITTER_NAME=DFL Test",
		"GIT_COMMITTER_EMAIL=dfl@example.com",
	}, "push", "origin", "HEAD")

	return origin, worktree
}

func cloneRepoForCLI(t *testing.T, origin, name string) string {
	t.Helper()

	target := filepath.Join(t.TempDir(), name)
	runGit(t, filepath.Dir(target), "clone", origin, target)
	return target
}

func writeFileForCLI(t *testing.T, dir, name, contents string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
