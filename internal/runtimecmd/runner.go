package runtimecmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

type Runner struct {
	Stdin  io.Reader
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

func (o Runner) Ask(question, defaultValue string) (string, error) {
	promptWriter := o.stderr()
	if _, err := fmt.Fprintf(promptWriter, "%s ", question); err != nil {
		return "", err
	}
	if defaultValue != "" {
		if _, err := fmt.Fprintf(promptWriter, "[%s] ", defaultValue); err != nil {
			return "", err
		}
	}

	reader := bufio.NewReader(o.stdin())
	reply, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := fmt.Fprintln(promptWriter); err != nil {
		return "", err
	}

	reply = strings.TrimSpace(reply)
	if reply == "" {
		reply = defaultValue
	}
	return reply, nil
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

func (o Runner) GitClone(ctx runtimectx.Context, origin, target string, update bool) (runtimectx.ResultStatus, string, error) {
	resolvedOrigin, err := resolveCloneOrigin(ctx.RepoRoot, origin)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	resolvedTarget, err := expandPath(target)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	info, err := os.Stat(resolvedTarget)
	if err == nil && info.IsDir() {
		gitDir := filepath.Join(resolvedTarget, ".git")
		if gitInfo, gitErr := os.Stat(gitDir); gitErr == nil && gitInfo.IsDir() {
			currentOrigin, gitErr := gitOrigin(resolvedTarget)
			if gitErr != nil {
				return runtimectx.StatusFailed, "", gitErr
			}
			if sameCloneOrigin(currentOrigin, resolvedOrigin) {
				if update {
					if ctx.DryRun {
						return runtimectx.StatusSuccess, fmt.Sprintf("already cloned, would update %s", resolvedTarget), nil
					}
					pullResult, err := gitPull(resolvedTarget)
					if err != nil {
						return runtimectx.StatusFailed, "failed to pull", err
					}
					if pullResult.upToDate {
						return runtimectx.StatusSkipped, "up-to-date", nil
					}
					return runtimectx.StatusSuccess, commitsPulledMessage(pullResult.commitCount), nil
				}
				return runtimectx.StatusSkipped, fmt.Sprintf("already cloned at %s", resolvedTarget), nil
			}
		}

		backupPath, err := o.Backup(ctx, resolvedTarget)
		if err != nil {
			return runtimectx.StatusFailed, "", err
		}
		if ctx.DryRun {
			return runtimectx.StatusSuccess, fmt.Sprintf("would back up to %s and clone %s into %s", backupPath, resolvedOrigin, resolvedTarget), nil
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runtimectx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would clone %s into %s", resolvedOrigin, resolvedTarget), nil
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runtimectx.StatusFailed, "", err
	}

	cmd := exec.Command("git", "clone", resolvedOrigin, resolvedTarget)
	cmd.Stdout = o.stdout()
	cmd.Stderr = o.stderr()
	if err := cmd.Run(); err != nil {
		return runtimectx.StatusFailed, "", err
	}
	return runtimectx.StatusSuccess, fmt.Sprintf("cloned %s into %s", resolvedOrigin, resolvedTarget), nil
}

func (o Runner) Symlink(ctx runtimectx.Context, componentRoot, source, target string) (runtimectx.ResultStatus, string, error) {
	resolvedSource, resolvedTarget, err := resolvePaths(componentRoot, source, target)
	if err != nil {
		return runtimectx.StatusFailed, "", err
	}

	var backupPath string
	if info, err := os.Lstat(resolvedTarget); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			current, err := os.Readlink(resolvedTarget)
			if err != nil {
				return runtimectx.StatusFailed, "", err
			}
			if samePath(current, resolvedSource, filepath.Dir(resolvedTarget)) {
				return runtimectx.StatusSkipped, fmt.Sprintf("already exists at %s", resolvedTarget), nil
			}
		}

		backupPath, err = o.Backup(ctx, resolvedTarget)
		if err != nil {
			return runtimectx.StatusFailed, "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return runtimectx.StatusFailed, "", err
	}

	if ctx.DryRun {
		if backupPath != "" {
			return runtimectx.StatusSuccess, fmt.Sprintf("would back up to %s and link %s -> %s", backupPath, resolvedTarget, resolvedSource), nil
		}
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

	if backupPath != "" {
		return runtimectx.StatusSuccess, fmt.Sprintf("backed up to %s and linked %s -> %s", backupPath, resolvedTarget, resolvedSource), nil
	}
	return runtimectx.StatusSuccess, fmt.Sprintf("linked %s -> %s", resolvedTarget, resolvedSource), nil
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

func gitOrigin(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func resolveCloneOrigin(repoRoot, origin string) (string, error) {
	if origin == "" {
		return "", errors.New("origin is required")
	}

	if isGitHubSSHOrigin(origin) || isGitHubHTTPSOrigin(origin) {
		return origin, nil
	}

	if owner, repo, ok := githubRepoFromOrigin(origin); ok {
		transport := githubTransportHTTPS
		if inherited, ok := githubTransportFromRepo(repoRoot); ok {
			transport = inherited
		}
		return githubCloneURL(owner, repo, transport), nil
	}

	return origin, nil
}

func sameCloneOrigin(currentOrigin, desiredOrigin string) bool {
	if currentOrigin == desiredOrigin {
		return true
	}

	currentOwner, currentRepo, currentOK := githubRepoFromOrigin(currentOrigin)
	desiredOwner, desiredRepo, desiredOK := githubRepoFromOrigin(desiredOrigin)
	return currentOK && desiredOK && currentOwner == desiredOwner && currentRepo == desiredRepo
}

type githubTransport string

const (
	githubTransportSSH   githubTransport = "ssh"
	githubTransportHTTPS githubTransport = "https"
)

func githubTransportFromRepo(repoRoot string) (githubTransport, bool) {
	if repoRoot == "" {
		return "", false
	}

	origin, err := gitOrigin(repoRoot)
	if err != nil {
		return "", false
	}

	switch {
	case isGitHubSSHOrigin(origin):
		return githubTransportSSH, true
	case isGitHubHTTPSOrigin(origin):
		return githubTransportHTTPS, true
	default:
		return "", false
	}
}

func githubCloneURL(owner, repo string, transport githubTransport) string {
	switch transport {
	case githubTransportSSH:
		return fmt.Sprintf("git@github.com:%s/%s.git", owner, repo)
	default:
		return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	}
}

func githubRepoFromOrigin(origin string) (string, string, bool) {
	switch {
	case isGitHubSSHOrigin(origin):
		return parseGitHubRepo(strings.TrimPrefix(origin, "git@github.com:"))
	case isGitHubHTTPSOrigin(origin):
		return parseGitHubRepo(strings.TrimPrefix(origin, "https://github.com/"))
	default:
		return parseGitHubRepo(origin)
	}
}

func parseGitHubRepo(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ".git")
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func isGitHubSSHOrigin(origin string) bool {
	return strings.HasPrefix(origin, "git@github.com:")
}

func isGitHubHTTPSOrigin(origin string) bool {
	return strings.HasPrefix(origin, "https://github.com/")
}

type gitPullResult struct {
	upToDate    bool
	commitCount int
}

func gitPull(dir string) (gitPullResult, error) {
	before, err := gitHead(dir)
	if err != nil {
		return gitPullResult{}, err
	}

	cmd := exec.Command("git", "pull")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return gitPullResult{}, fmt.Errorf("git pull failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	after, err := gitHead(dir)
	if err != nil {
		return gitPullResult{}, err
	}
	if before == after {
		return gitPullResult{upToDate: true}, nil
	}

	commitCount, err := gitCommitCount(dir, before, after)
	if err != nil {
		return gitPullResult{}, err
	}
	return gitPullResult{commitCount: commitCount}, nil
}

func gitHead(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func gitCommitCount(dir, before, after string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", before+".."+after)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("git rev-list --count failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	countText := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countText)
	if err != nil {
		return 0, fmt.Errorf("parse git rev-list --count output %q: %w", countText, err)
	}
	return count, nil
}

func commitsPulledMessage(count int) string {
	if count == 1 {
		return "1 commit pulled"
	}
	return fmt.Sprintf("%d commits pulled", count)
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

func (o Runner) stdin() io.Reader {
	if o.Stdin == nil {
		return os.Stdin
	}
	return o.Stdin
}
