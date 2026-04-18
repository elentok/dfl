package selfmgr

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/runtimecmd"
	"dfl/internal/ui"
)

type Updater struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	DryRun bool
}

func (u Updater) Run(repoOverride string) (int, error) {
	installer := Installer{DryRun: u.DryRun}

	var installedPath string
	err := ui.Step(u.stdout(), "Installing dfl", func() (runtimectx.ResultStatus, string, error) {
		result, err := installer.Install("", "")
		if err != nil {
			return "", "", err
		}
		installedPath = result.Path
		return result.Status, result.Message, nil
	})
	if err != nil {
		return 1, err
	}

	repoRoot, err := resolveRepoRoot(repoOverride)
	if err != nil {
		return 1, err
	}

	err = ui.Step(u.stdout(), "Updating dotfiles repo", func() (runtimectx.ResultStatus, string, error) {
		if u.DryRun {
			return runtimectx.StatusSuccess, fmt.Sprintf("would update %s", repoRoot), nil
		}

		if err := u.updateRepo(repoRoot); err != nil {
			return "", "", err
		}
		return runtimectx.StatusSuccess, fmt.Sprintf("updated %s", repoRoot), nil
	})
	if err != nil {
		return 1, err
	}

	err = ui.Step(u.stdout(), "Running dotfiles setup", func() (runtimectx.ResultStatus, string, error) {
		if u.DryRun {
			return runtimectx.StatusSuccess, fmt.Sprintf("would run dfl setup --repo %s --dry-run", repoRoot), nil
		}

		setupBinary, err := setupBinaryPath(installedPath)
		if err != nil {
			return "", "", err
		}

		args := []string{"setup", "--repo", repoRoot}
		cmd := exec.Command(setupBinary, args...)
		cmd.Stdout = u.stdout()
		cmd.Stderr = u.stderr()
		if err := cmd.Run(); err != nil {
			return "", "", err
		}

		return runtimectx.StatusSuccess, fmt.Sprintf("ran setup for %s", repoRoot), nil
	})
	if err != nil {
		return 1, err
	}

	return 0, nil
}

func resolveRepoRoot(repoOverride string) (string, error) {
	if repoOverride != "" {
		return filepath.Abs(repoOverride)
	}

	if ctx, err := runtimectx.NewContext(""); err == nil {
		return ctx.RepoRoot, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	fallback := filepath.Join(home, ".dotfiles")
	if _, err := os.Stat(filepath.Join(fallback, ".git")); err == nil {
		return fallback, nil
	}

	return "", fmt.Errorf("could not resolve dotfiles repo; run from inside the repo or pass --repo")
}

func setupBinaryPath(installedPath string) (string, error) {
	if installedPath != "" {
		return installedPath, nil
	}

	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return path, nil
}

func (u Updater) stdout() io.Writer {
	if u.Stdout != nil {
		return u.Stdout
	}
	return io.Discard
}

func (u Updater) stderr() io.Writer {
	if u.Stderr != nil {
		return u.Stderr
	}
	return io.Discard
}

func (u Updater) stdin() io.Reader {
	if u.Stdin != nil {
		return u.Stdin
	}
	return os.Stdin
}

func (u Updater) updateRepo(repoRoot string) error {
	output, err := u.runGit(repoRoot, "pull", "--ff-only")
	if err == nil {
		return nil
	}
	if !isPullBlockedByLocalChanges(output) || !hasTrackedLocalChanges(repoRoot) {
		return err
	}

	reply, askErr := runtimecmd.Runner{Stdin: u.stdin(), Stderr: u.stderr()}.Ask("Local changes would be overwritten by pull. Stash them before pulling?", "n")
	if askErr != nil {
		return askErr
	}
	if !isAffirmative(reply) {
		return err
	}

	if _, stashErr := u.runGit(repoRoot, "stash", "push", "--message", "dfl update auto-stash"); stashErr != nil {
		return stashErr
	}

	if _, pullErr := u.runGit(repoRoot, "pull", "--ff-only"); pullErr != nil {
		return fmt.Errorf("git pull failed after stashing changes; stash was kept: %w", pullErr)
	}

	if _, popErr := u.runGit(repoRoot, "stash", "pop", "--index"); popErr != nil {
		return fmt.Errorf("git pull succeeded but failed to restore stashed changes: %w", popErr)
	}

	return nil
}

func (u Updater) runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(u.stdout(), &output)
	cmd.Stderr = io.MultiWriter(u.stderr(), &output)
	err := cmd.Run()
	return output.String(), err
}

func hasTrackedLocalChanges(repoRoot string) bool {
	cmd := exec.Command("git", "-C", repoRoot, "status", "--porcelain", "--untracked-files=no")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func isPullBlockedByLocalChanges(output string) bool {
	return strings.Contains(output, "Please commit your changes or stash them before you merge.") ||
		strings.Contains(output, "Your local changes to the following files would be overwritten by merge:") ||
		strings.Contains(output, "cannot pull with rebase: You have unstaged changes.") ||
		strings.Contains(output, "Please commit or stash them.")
}

func isAffirmative(reply string) bool {
	switch strings.ToLower(strings.TrimSpace(reply)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
