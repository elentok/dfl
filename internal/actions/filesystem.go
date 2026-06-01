package actions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dfl/internal/runctx"
)

func (o Runner) Copy(ctx runctx.Context, componentRoot, source, target string) (runctx.ResultStatus, string, error) {
	resolvedSource, resolvedTarget, err := resolvePaths(componentRoot, source, target)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	same, err := sameFileContents(resolvedSource, resolvedTarget)
	if err != nil {
		return runctx.StatusFailed, "", err
	}
	if same {
		return runctx.StatusSkipped, "already up to date", nil
	}

	if _, err := os.Stat(resolvedTarget); err == nil {
		if _, err := o.Backup(ctx, resolvedTarget); err != nil {
			return runctx.StatusFailed, "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runctx.StatusSuccess, fmt.Sprintf("would copy %s -> %s", resolvedSource, resolvedTarget), nil
	}

	data, err := os.ReadFile(resolvedSource)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runctx.StatusFailed, "", err
	}
	if err := os.WriteFile(resolvedTarget, data, sourceMode(resolvedSource)); err != nil {
		return runctx.StatusFailed, "", err
	}

	return runctx.StatusSuccess, fmt.Sprintf("copied %s", resolvedTarget), nil
}

func (o Runner) Symlink(ctx runctx.Context, componentRoot, source, target string) (runctx.ResultStatus, string, error) {
	resolvedSource, resolvedTarget, err := resolvePaths(componentRoot, source, target)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	var backupPath string
	if info, err := os.Lstat(resolvedTarget); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			current, err := os.Readlink(resolvedTarget)
			if err != nil {
				return runctx.StatusFailed, "", err
			}
			if samePath(current, resolvedSource, filepath.Dir(resolvedTarget)) {
				return runctx.StatusSkipped, "already exists", nil
			}
		}

		backupPath, err = o.Backup(ctx, resolvedTarget)
		if err != nil {
			return runctx.StatusFailed, "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}

	if ctx.DryRun {
		if backupPath != "" {
			return runctx.StatusSuccess, fmt.Sprintf("would back up to %s and link %s -> %s", backupPath, resolvedTarget, resolvedSource), nil
		}
		return runctx.StatusSuccess, fmt.Sprintf("would link %s -> %s", resolvedTarget, resolvedSource), nil
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runctx.StatusFailed, "", err
	}
	if err := os.RemoveAll(resolvedTarget); err != nil && !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}
	if err := os.Symlink(resolvedSource, resolvedTarget); err != nil {
		return runctx.StatusFailed, "", err
	}

	if backupPath != "" {
		return runctx.StatusSuccess, fmt.Sprintf("backed up to %s and linked %s -> %s", backupPath, resolvedTarget, resolvedSource), nil
	}
	return runctx.StatusSuccess, fmt.Sprintf("linked %s -> %s", resolvedTarget, resolvedSource), nil
}

func (o Runner) Mkdir(ctx runctx.Context, path string) (runctx.ResultStatus, string, error) {
	resolvedPath, err := expandPath(path)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	if info, err := os.Stat(resolvedPath); err == nil && info.IsDir() {
		return runctx.StatusSkipped, "already exists", nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runctx.StatusSuccess, fmt.Sprintf("would create %s", resolvedPath), nil
	}

	if err := os.MkdirAll(resolvedPath, 0o755); err != nil {
		return runctx.StatusFailed, "", err
	}

	return runctx.StatusSuccess, fmt.Sprintf("created %s", resolvedPath), nil
}

func (o Runner) Backup(ctx runctx.Context, path string) (string, error) {
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
