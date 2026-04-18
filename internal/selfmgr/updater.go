package selfmgr

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

type Updater struct {
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

		cmd := exec.Command("git", "-C", repoRoot, "pull", "--ff-only")
		cmd.Stdout = u.stdout()
		cmd.Stderr = u.stderr()
		if err := cmd.Run(); err != nil {
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
