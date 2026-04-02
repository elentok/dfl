package runtimecmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (o Runner) HasCommand(name string) (bool, error) {
	_, err := exec.LookPath(name)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, exec.ErrNotFound) {
		return false, nil
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return false, nil
	}

	return false, err
}

func (o Runner) StepStart(message string) error {
	return ui.StepStart(o.stdout(), message)
}

func (o Runner) StepEnd(status runtimectx.ResultStatus, message string) error {
	return ui.StepEnd(o.stdout(), status, message)
}

func (o Runner) Shell(ctx runtimectx.Context, name string, command []string) (int, error) {
	if len(command) == 0 {
		return 2, errors.New("shell requires a command after --")
	}

	if err := o.StepStart(name); err != nil {
		return 1, err
	}

	if ctx.DryRun {
		if _, err := fmt.Fprintf(o.stdout(), "DRY-RUN: %s\n", strings.Join(command, " ")); err != nil {
			return 1, err
		}
		if err := o.StepEnd(runtimectx.StatusSkipped, "dry-run"); err != nil {
			return 1, err
		}
		return 0, nil
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = o.stdout()
	cmd.Stderr = o.stderr()
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		_ = o.StepEnd(runtimectx.StatusFailed, "command failed")

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}

		return 1, err
	}

	if err := o.StepEnd(runtimectx.StatusSuccess, "done"); err != nil {
		return 1, err
	}

	return 0, nil
}

func (o Runner) Symlink(ctx runtimectx.Context, componentRoot, source, target string) (runtimectx.ResultStatus, string, error) {
	resolvedSource, resolvedTarget, err := resolvePaths(componentRoot, source, target)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	if info, err := os.Lstat(resolvedTarget); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			current, err := os.Readlink(resolvedTarget)
			if err != nil {
				return runtimectx.StatusFailed, "", err
			}
			if samePath(current, resolvedSource, filepath.Dir(resolvedTarget)) {
				return runtimectx.StatusSkipped, "already linked", nil
			}
		}

		if _, err := o.Backup(ctx, resolvedTarget); err != nil {
			return runtimectx.StatusFailed, "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return runtimectx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would link %s -> %s", resolvedTarget, resolvedSource), nil
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runtimectx.StatusFailed, "", err
	}
	if err := os.RemoveAll(resolvedTarget); err != nil && !errors.Is(err, os.ErrNotExist) {
		return runtimectx.StatusFailed, "", err
	}
	if err := os.Symlink(resolvedSource, resolvedTarget); err != nil {
		return runtimectx.StatusFailed, "", err
	}

	return runtimectx.StatusSuccess, fmt.Sprintf("linked %s", resolvedTarget), nil
}

func (o Runner) Copy(ctx runtimectx.Context, componentRoot, source, target string) (runtimectx.ResultStatus, string, error) {
	resolvedSource, resolvedTarget, err := resolvePaths(componentRoot, source, target)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	same, err := sameFileContents(resolvedSource, resolvedTarget)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}
	if same {
		return runtimectx.StatusSkipped, "already up to date", nil
	}

	if _, err := os.Stat(resolvedTarget); err == nil {
		if _, err := o.Backup(ctx, resolvedTarget); err != nil {
			return runtimectx.StatusFailed, "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return runtimectx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would copy %s -> %s", resolvedSource, resolvedTarget), nil
	}

	data, err := os.ReadFile(resolvedSource)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runtimectx.StatusFailed, "", err
	}
	if err := os.WriteFile(resolvedTarget, data, sourceMode(resolvedSource)); err != nil {
		return runtimectx.StatusFailed, "", err
	}

	return runtimectx.StatusSuccess, fmt.Sprintf("copied %s", resolvedTarget), nil
}

func (o Runner) Mkdir(ctx runtimectx.Context, path string) (runtimectx.ResultStatus, string, error) {
	resolvedPath, err := expandPath(path)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	if info, err := os.Stat(resolvedPath); err == nil && info.IsDir() {
		return runtimectx.StatusSkipped, "already exists", nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runtimectx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would create %s", resolvedPath), nil
	}

	if err := os.MkdirAll(resolvedPath, 0o755); err != nil {
		return runtimectx.StatusFailed, "", err
	}

	return runtimectx.StatusSuccess, fmt.Sprintf("created %s", resolvedPath), nil
}

func (o Runner) Backup(ctx runtimectx.Context, path string) (string, error) {
	resolvedPath, err := expandPath(path)
	if err != nil {
		return "", err
	}

	if _, err := os.Lstat(resolvedPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	backupPath, err := nextBackupPath(resolvedPath)
	if err != nil {
		return "", err
	}

	if ctx.DryRun {
		return backupPath, nil
	}

	if err := os.Rename(resolvedPath, backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}

func resolvePaths(componentRoot, source, target string) (string, string, error) {
	resolvedTarget, err := expandPath(target)
	if err != nil {
		return "", "", err
	}

	resolvedSource := source
	if !filepath.IsAbs(source) && !strings.HasPrefix(source, "~") {
		resolvedSource = filepath.Join(componentRoot, source)
	}
	resolvedSource, err = expandPath(resolvedSource)
	if err != nil {
		return "", "", err
	}

	return resolvedSource, resolvedTarget, nil
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}

	return filepath.Abs(path)
}

func samePath(currentLink, expected, linkDir string) bool {
	if filepath.IsAbs(currentLink) {
		return filepath.Clean(currentLink) == filepath.Clean(expected)
	}

	return filepath.Clean(filepath.Join(linkDir, currentLink)) == filepath.Clean(expected)
}

func sameFileContents(source, target string) (bool, error) {
	sourceData, err := os.ReadFile(source)
	if err != nil {
		return false, err
	}

	targetData, err := os.ReadFile(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return string(sourceData) == string(targetData), nil
}

func sourceMode(path string) fs.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return 0o644
	}
	return info.Mode().Perm()
}

func nextBackupPath(path string) (string, error) {
	defaultPath := path + ".backup"
	if _, err := os.Lstat(defaultPath); errors.Is(err, os.ErrNotExist) {
		return defaultPath, nil
	} else if err != nil {
		return "", err
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf("%s.backup.%s", path, timestamp), nil
}

func (o Runner) stdout() io.Writer {
	if o.Stdout == nil {
		return io.Discard
	}
	return o.Stdout
}

func (o Runner) stderr() io.Writer {
	if o.Stderr == nil {
		return io.Discard
	}
	return o.Stderr
}
