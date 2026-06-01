package actions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"dfl/internal/runctx"
)

func (o Runner) GitClone(ctx runctx.Context, origin, target string, update bool) (runctx.ResultStatus, string, error) {
	resolvedOrigin, err := resolveCloneOrigin(ctx.RepoRoot, origin)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	resolvedTarget, err := expandPath(target)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	info, err := os.Stat(resolvedTarget)
	if err == nil && info.IsDir() {
		gitDir := filepath.Join(resolvedTarget, ".git")
		if gitInfo, gitErr := os.Stat(gitDir); gitErr == nil && gitInfo.IsDir() {
			currentOrigin, gitErr := gitOrigin(resolvedTarget)
			if gitErr != nil {
				return runctx.StatusFailed, "", gitErr
			}
			if sameCloneOrigin(currentOrigin, resolvedOrigin) {
				if update {
					if ctx.DryRun {
						return runctx.StatusSuccess, fmt.Sprintf("already cloned, would update %s", resolvedTarget), nil
					}
					pullResult, err := gitPull(resolvedTarget)
					if err != nil {
						return runctx.StatusFailed, "failed to pull", err
					}
					if pullResult.upToDate {
						return runctx.StatusSkipped, "up-to-date", nil
					}
					return runctx.StatusSuccess, commitsPulledMessage(pullResult.commitCount), nil
				}
				return runctx.StatusSkipped, fmt.Sprintf("already cloned at %s", resolvedTarget), nil
			}
		}

		backupPath, err := o.Backup(ctx, resolvedTarget)
		if err != nil {
			return runctx.StatusFailed, "", err
		}
		if ctx.DryRun {
			return runctx.StatusSuccess, fmt.Sprintf("would back up to %s and clone %s into %s", backupPath, resolvedOrigin, resolvedTarget), nil
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}

	if ctx.DryRun {
		return runctx.StatusSuccess, fmt.Sprintf("would clone %s into %s", resolvedOrigin, resolvedTarget), nil
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runctx.StatusFailed, "", err
	}

	cmd := exec.Command("git", "clone", resolvedOrigin, resolvedTarget)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(o.stdout(), &stdoutBuf)
	cmd.Stderr = io.MultiWriter(o.stderr(), &stderrBuf)
	if err := cmd.Run(); err != nil {
		return runctx.StatusFailed, "", &OutputError{Err: err, Output: combinedOutput(stdoutBuf.String(), stderrBuf.String())}
	}
	return runctx.StatusSuccess, fmt.Sprintf("cloned %s into %s", resolvedOrigin, resolvedTarget), nil
}

func gitOrigin(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &OutputError{
			Err:    fmt.Errorf("git remote get-url origin failed: %w: %s", err, strings.TrimSpace(string(output))),
			Output: string(output),
		}
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
		return gitPullResult{}, &OutputError{
			Err:    fmt.Errorf("git pull failed: %w: %s", err, strings.TrimSpace(string(output))),
			Output: string(output),
		}
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
		return "", &OutputError{
			Err:    fmt.Errorf("git rev-parse HEAD failed: %w: %s", err, strings.TrimSpace(string(output))),
			Output: string(output),
		}
	}
	return strings.TrimSpace(string(output)), nil
}

func gitCommitCount(dir, before, after string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", before+".."+after)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, &OutputError{
			Err:    fmt.Errorf("git rev-list --count failed: %w: %s", err, strings.TrimSpace(string(output))),
			Output: string(output),
		}
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
