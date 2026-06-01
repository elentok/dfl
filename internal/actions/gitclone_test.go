package actions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"dfl/internal/runctx"
)

func TestGitCloneSkipsWhenAlreadyClonedFromSameOrigin(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	status, message, err := Runner{}.GitClone(runctx.Context{}, origin, target, false)
	if err != nil {
		t.Fatalf("GitClone returned error: %v", err)
	}
	if status != runctx.StatusSkipped {
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

	status, message, err := Runner{}.GitClone(runctx.Context{}, origin, target, true)
	if err != nil {
		t.Fatalf("GitClone returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if message != "1 commit pulled" {
		t.Fatalf("message = %q, want 1 commit pulled", message)
	}

	data, err := os.ReadFile(filepath.Join(target, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "updated\n" {
		t.Fatalf("README.md = %q, want updated content", string(data))
	}
}

func TestGitCloneReportsUpToDateWhenPullHasNoChanges(t *testing.T) {
	origin, _ := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	status, message, err := Runner{}.GitClone(runctx.Context{}, origin, target, true)
	if err != nil {
		t.Fatalf("GitClone returned error: %v", err)
	}
	if status != runctx.StatusSkipped {
		t.Fatalf("status = %q, want skipped", status)
	}
	if message != "up-to-date" {
		t.Fatalf("message = %q, want up-to-date", message)
	}
}

func TestGitCloneReportsMultiplePulledCommits(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	writeFile(t, worktree, "README.md", "updated once\n")
	gitRun(t, worktree, "add", "README.md")
	gitRun(t, worktree, "commit", "-m", "update readme once")

	writeFile(t, worktree, "README.md", "updated twice\n")
	gitRun(t, worktree, "add", "README.md")
	gitRun(t, worktree, "commit", "-m", "update readme twice")
	gitRun(t, worktree, "push", "origin", "HEAD")

	status, message, err := Runner{}.GitClone(runctx.Context{}, origin, target, true)
	if err != nil {
		t.Fatalf("GitClone returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if message != "2 commits pulled" {
		t.Fatalf("message = %q, want 2 commits pulled", message)
	}
}

func TestGitCloneReportsFailedToPull(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	writeFile(t, worktree, "README.md", "remote change\n")
	gitRun(t, worktree, "add", "README.md")
	gitRun(t, worktree, "commit", "-m", "remote update")
	gitRun(t, worktree, "push", "origin", "HEAD")

	writeFile(t, target, "README.md", "local change\n")

	status, message, err := Runner{}.GitClone(runctx.Context{}, origin, target, true)
	if err == nil {
		t.Fatal("GitClone returned nil error, want pull failure")
	}
	if status != runctx.StatusFailed {
		t.Fatalf("status = %q, want failed", status)
	}
	if message != "failed to pull" {
		t.Fatalf("message = %q, want failed to pull", message)
	}
}

func TestResolveCloneOriginDefaultsGitHubSlugToHTTPS(t *testing.T) {
	resolved, err := resolveCloneOrigin("", "elentok/stuff.nvim")
	if err != nil {
		t.Fatalf("resolveCloneOrigin returned error: %v", err)
	}
	if resolved != "https://github.com/elentok/stuff.nvim.git" {
		t.Fatalf("resolved = %q, want https github url", resolved)
	}
}

func TestResolveCloneOriginInheritsSSHFromRepoOrigin(t *testing.T) {
	repoRoot := createGitRepoWithOrigin(t, "git@github.com:elentok/dotfiles.git")

	resolved, err := resolveCloneOrigin(repoRoot, "elentok/stuff.nvim")
	if err != nil {
		t.Fatalf("resolveCloneOrigin returned error: %v", err)
	}
	if resolved != "git@github.com:elentok/stuff.nvim.git" {
		t.Fatalf("resolved = %q, want ssh github url", resolved)
	}
}

func TestResolveCloneOriginKeepsExplicitHTTPSOrigin(t *testing.T) {
	repoRoot := createGitRepoWithOrigin(t, "git@github.com:elentok/dotfiles.git")

	resolved, err := resolveCloneOrigin(repoRoot, "https://github.com/elentok/stuff.nvim")
	if err != nil {
		t.Fatalf("resolveCloneOrigin returned error: %v", err)
	}
	if resolved != "https://github.com/elentok/stuff.nvim" {
		t.Fatalf("resolved = %q, want explicit https origin", resolved)
	}
}

func TestSameCloneOriginTreatsGitHubTransportVariantsAsSameRepo(t *testing.T) {
	if !sameCloneOrigin("git@github.com:elentok/stuff.nvim.git", "https://github.com/elentok/stuff.nvim.git") {
		t.Fatal("sameCloneOrigin returned false, want true for same github repo")
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

func createGitRepoWithOrigin(t *testing.T, origin string) string {
	t.Helper()

	repoRoot := t.TempDir()
	gitRun(t, repoRoot, "init")
	gitRun(t, repoRoot, "remote", "add", "origin", origin)
	return repoRoot
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
