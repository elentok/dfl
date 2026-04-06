package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"dfl/internal/manifest"
	runtimectx "dfl/internal/runtime"
)

func TestResolveRepoURLInheritsSSHFromGitHubOrigin(t *testing.T) {
	syncer := repoSyncer{
		git: fakeGitRunner{
			outputs: map[string]gitResult{
				key("", "remote", "get-url", "origin"): {stdout: "git@github.com:elentok/dotfiles.git"},
			},
		},
	}

	url, err := syncer.resolveRepoURL("", manifest.RepoDefaults{Transport: "inherit"}, manifest.RepoSpec{GitHub: "elentok/notes"})
	if err != nil {
		t.Fatalf("resolveRepoURL: %v", err)
	}
	if url != "git@github.com:elentok/notes.git" {
		t.Fatalf("url = %q, want SSH GitHub URL", url)
	}
}

func TestResolveRepoURLFallsBackToHTTPSForNonGitHubOrigin(t *testing.T) {
	syncer := repoSyncer{
		git: fakeGitRunner{
			outputs: map[string]gitResult{
				key("", "remote", "get-url", "origin"): {stdout: "git@gitlab.com:elentok/dotfiles.git"},
			},
		},
	}

	url, err := syncer.resolveRepoURL("", manifest.RepoDefaults{Transport: "inherit"}, manifest.RepoSpec{GitHub: "elentok/notes"})
	if err != nil {
		t.Fatalf("resolveRepoURL: %v", err)
	}
	if url != "https://github.com/elentok/notes.git" {
		t.Fatalf("url = %q, want HTTPS GitHub URL", url)
	}
}

func TestSyncClonesMissingRepo(t *testing.T) {
	origin := createOriginRepo(t)
	target := filepath.Join(t.TempDir(), "notes")

	syncer := repoSyncer{git: execGitRunner{}}
	status, msg, err := syncer.Sync(
		runtimectx.Context{RepoRoot: origin},
		manifest.RepoDefaults{},
		manifest.RepoSpec{Name: "notes", URL: origin, Path: target},
	)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != runtimectx.StatusSuccess {
		t.Fatalf("status = %q, want success (%s)", status, msg)
	}
	if _, err := os.Stat(filepath.Join(target, "README.md")); err != nil {
		t.Fatalf("cloned repo missing README.md: %v", err)
	}
}

func TestSyncPullsExistingCheckout(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	writeFile(t, worktree, "README.md", "updated\n")
	gitRun(t, worktree, "add", "README.md")
	gitRun(t, worktree, "commit", "-m", "update readme")
	gitRun(t, worktree, "push", "origin", "HEAD")

	syncer := repoSyncer{git: execGitRunner{}}
	status, msg, err := syncer.Sync(
		runtimectx.Context{RepoRoot: worktree},
		manifest.RepoDefaults{},
		manifest.RepoSpec{Name: "notes", URL: origin, Path: target},
	)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != runtimectx.StatusSuccess {
		t.Fatalf("status = %q, want success (%s)", status, msg)
	}

	data, err := os.ReadFile(filepath.Join(target, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "updated\n" {
		t.Fatalf("README.md = %q, want updated content", string(data))
	}
}

func TestSyncReportsDivergedCheckout(t *testing.T) {
	origin, worktree := createOriginAndWorktree(t)
	target := cloneRepo(t, origin, "target")

	writeFile(t, target, "local.txt", "local\n")
	gitRun(t, target, "add", "local.txt")
	gitRun(t, target, "commit", "-m", "local change")

	writeFile(t, worktree, "remote.txt", "remote\n")
	gitRun(t, worktree, "add", "remote.txt")
	gitRun(t, worktree, "commit", "-m", "remote change")
	gitRun(t, worktree, "push", "origin", "HEAD")

	syncer := repoSyncer{git: execGitRunner{}}
	status, msg, err := syncer.Sync(
		runtimectx.Context{RepoRoot: worktree},
		manifest.RepoDefaults{},
		manifest.RepoSpec{Name: "notes", URL: origin, Path: target},
	)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != runtimectx.StatusFailed {
		t.Fatalf("status = %q, want failed (%s)", status, msg)
	}
	if !strings.Contains(msg, "diverged from its upstream branch") {
		t.Fatalf("msg = %q, want divergence message", msg)
	}
}

func TestSyncDryRunReportsNonGitPreconditionFailure(t *testing.T) {
	target := filepath.Join(t.TempDir(), "notes")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	syncer := repoSyncer{git: execGitRunner{}}
	status, msg, err := syncer.Sync(
		runtimectx.Context{RepoRoot: t.TempDir(), DryRun: true},
		manifest.RepoDefaults{},
		manifest.RepoSpec{Name: "notes", URL: "https://github.com/elentok/notes.git", Path: target},
	)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != runtimectx.StatusFailed {
		t.Fatalf("status = %q, want failed", status)
	}
	if !strings.Contains(msg, "not a Git checkout") {
		t.Fatalf("msg = %q, want precondition failure", msg)
	}
}

type fakeGitRunner struct {
	outputs map[string]gitResult
}

type gitResult struct {
	stdout string
	stderr string
	err    error
}

func (f fakeGitRunner) run(dir string, args ...string) (string, string, error) {
	result, ok := f.outputs[key(dir, args...)]
	if !ok {
		return "", "", nil
	}
	return result.stdout, result.stderr, result.err
}

func key(dir string, args ...string) string {
	return dir + "::" + strings.Join(args, "\x00")
}

func createOriginRepo(t *testing.T) string {
	origin, worktree := createOriginAndWorktree(t)
	_ = worktree
	return origin
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
