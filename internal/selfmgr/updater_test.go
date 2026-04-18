package selfmgr

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateDryRunDoesNotRequireInstalledBinary(t *testing.T) {
	repoRoot := t.TempDir()

	var stdout bytes.Buffer
	updater := Updater{
		Stdout: &stdout,
		DryRun: true,
	}

	code, err := updater.Run(repoRoot)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "would install dfl ") && !strings.Contains(output, "latest version already installed") {
		t.Fatalf("stdout = %q, want dry-run install output or latest-version skip output", output)
	}
	if !strings.Contains(output, "would update "+repoRoot) {
		t.Fatalf("stdout = %q, want dry-run repo update output", output)
	}
	if !strings.Contains(output, "would run dfl setup --repo "+repoRoot+" --dry-run") {
		t.Fatalf("stdout = %q, want dry-run setup output", output)
	}
}

func TestUpdateRepoOffersToStashTrackedChangesAndRestoresThem(t *testing.T) {
	repoRoot := t.TempDir()
	stubDir := t.TempDir()
	stateDir := t.TempDir()
	writeGitStub(t, stubDir)
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DFL_GIT_STUB_STATE_DIR", stateDir)

	var stderr bytes.Buffer
	updater := Updater{
		Stdin:  strings.NewReader("y\n"),
		Stderr: &stderr,
	}

	if err := updater.updateRepo(repoRoot); err != nil {
		t.Fatalf("updateRepo returned error: %v", err)
	}

	if !strings.Contains(stderr.String(), "Stash them before pulling? [n]") {
		t.Fatalf("stderr = %q, want stash prompt", stderr.String())
	}
	assertFileExists(t, filepath.Join(stateDir, "stash-push"))
	assertFileExists(t, filepath.Join(stateDir, "stash-pop"))
}

func TestUpdateRepoLeavesStashWhenPullStillFails(t *testing.T) {
	repoRoot := t.TempDir()
	stubDir := t.TempDir()
	stateDir := t.TempDir()
	writeGitStub(t, stubDir)
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DFL_GIT_STUB_STATE_DIR", stateDir)
	t.Setenv("DFL_GIT_STUB_FAIL_AFTER_STASH", "1")

	updater := Updater{
		Stdin: strings.NewReader("y\n"),
	}

	err := updater.updateRepo(repoRoot)
	if err == nil {
		t.Fatalf("updateRepo returned nil error, want failure after stashing")
	}
	if !strings.Contains(err.Error(), "stash was kept") {
		t.Fatalf("err = %q, want stash-kept message", err)
	}
	assertFileExists(t, filepath.Join(stateDir, "stash-push"))
	if _, statErr := os.Stat(filepath.Join(stateDir, "stash-pop")); !os.IsNotExist(statErr) {
		t.Fatalf("stash pop should not run on failed pull, err=%v", statErr)
	}
}

func TestIsPullBlockedByLocalChangesMatchesRebaseStyleGitMessage(t *testing.T) {
	output := "error: cannot pull with rebase: You have unstaged changes.\nerror: Please commit or stash them.\n"
	if !isPullBlockedByLocalChanges(output) {
		t.Fatalf("expected rebase-style git message to trigger stash prompt detection")
	}
}

func writeGitStub(t *testing.T, dir string) {
	t.Helper()

	script := `#!/usr/bin/env bash
set -euo pipefail

state_dir="${DFL_GIT_STUB_STATE_DIR:?}"
fail_after_stash="${DFL_GIT_STUB_FAIL_AFTER_STASH:-}"

if [ "${1:-}" = "-C" ]; then
  shift 2
fi

cmd="${1:-}"
shift || true

case "$cmd" in
  status)
    printf ' M tracked.txt\n'
    ;;
  pull)
    if [ ! -f "$state_dir/stash-push" ] || [ -n "$fail_after_stash" ]; then
      printf 'error: Your local changes to the following files would be overwritten by merge:\n' >&2
      printf 'Please commit your changes or stash them before you merge.\n' >&2
      exit 1
    fi
    printf 'Already up to date.\n'
    ;;
  stash)
    subcmd="${1:-}"
    shift || true
    case "$subcmd" in
      push)
        : > "$state_dir/stash-push"
        printf 'Saved working directory and index state\n'
        ;;
      pop)
        : > "$state_dir/stash-pop"
        printf 'Dropped refs/stash@{0}\n'
        ;;
      *)
        printf 'unexpected stash subcommand: %s\n' "$subcmd" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    printf 'unexpected git command: %s\n' "$cmd" >&2
    exit 1
    ;;
esac
`

	path := filepath.Join(dir, "git")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile git stub: %v", err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
