package setup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"dfl/internal/manifest"
	runtimectx "dfl/internal/runtime"
)

type gitRunner interface {
	run(dir string, args ...string) (string, string, error)
}

type execGitRunner struct{}

func (execGitRunner) run(dir string, args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

type repoSyncer struct {
	git gitRunner
}

func (s repoSyncer) Sync(ctx runtimectx.Context, defaults manifest.RepoDefaults, repo manifest.RepoSpec) (runtimectx.ResultStatus, string, error) {
	targetPath, err := expandRepoPath(repo.Path)
	if err != nil {
		return "", "", err
	}

	repoURL, err := s.resolveRepoURL(ctx.RepoRoot, defaults, repo)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(targetPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}

	if errors.Is(err, os.ErrNotExist) {
		return s.cloneRepo(ctx, repo, repoURL, targetPath)
	}

	if !info.IsDir() {
		return runtimectx.StatusFailed, fmt.Sprintf("%s path exists but is not a directory: %s", repo.Name, targetPath), nil
	}

	isGitRepo, err := s.isGitCheckout(targetPath)
	if err != nil {
		return "", "", err
	}
	if !isGitRepo {
		return runtimectx.StatusFailed, fmt.Sprintf("%s path exists but is not a Git checkout: %s", repo.Name, targetPath), nil
	}

	return s.pullRepo(ctx, repo, targetPath)
}

func (s repoSyncer) cloneRepo(ctx runtimectx.Context, repo manifest.RepoSpec, repoURL, targetPath string) (runtimectx.ResultStatus, string, error) {
	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would clone %s to %s", repoURL, targetPath), nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", "", err
	}

	if _, stderr, err := s.git.run(ctx.RepoRoot, "clone", repoURL, targetPath); err != nil {
		return runtimectx.StatusFailed, fmt.Sprintf("failed to clone %s: %s", repo.Name, commandErrorMessage(stderr, err)), nil
	}

	return runtimectx.StatusSuccess, fmt.Sprintf("cloned into %s", targetPath), nil
}

func (s repoSyncer) pullRepo(ctx runtimectx.Context, repo manifest.RepoSpec, targetPath string) (runtimectx.ResultStatus, string, error) {
	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would pull --ff-only in %s", targetPath), nil
	}

	stdout, stderr, err := s.git.run(targetPath, "pull", "--ff-only")
	if err == nil {
		if message := strings.TrimSpace(firstNonEmpty(stdout, stderr)); message != "" {
			return runtimectx.StatusSuccess, message, nil
		}
		return runtimectx.StatusSuccess, "pull completed", nil
	}

	diverged, divergeErr := s.hasDiverged(targetPath)
	if divergeErr != nil {
		return "", "", divergeErr
	}
	if diverged {
		return runtimectx.StatusFailed, fmt.Sprintf("%s diverged from its upstream branch; resolve it manually before rerunning setup", repo.Name), nil
	}

	return runtimectx.StatusFailed, fmt.Sprintf("failed to pull %s: %s", repo.Name, commandErrorMessage(firstNonEmpty(stderr, stdout), err)), nil
}

func (s repoSyncer) resolveRepoURL(repoRoot string, defaults manifest.RepoDefaults, repo manifest.RepoSpec) (string, error) {
	if repo.URL != "" {
		return repo.URL, nil
	}

	transport, err := s.resolveTransport(repoRoot, defaults, repo)
	if err != nil {
		return "", err
	}

	return githubURL(repo.GitHub, transport)
}

func (s repoSyncer) resolveTransport(repoRoot string, defaults manifest.RepoDefaults, repo manifest.RepoSpec) (string, error) {
	transport := repo.Transport
	if transport == "" {
		transport = defaults.Transport
	}
	if transport == "" || transport == "inherit" {
		return s.detectOriginTransport(repoRoot), nil
	}
	return transport, nil
}

func (s repoSyncer) detectOriginTransport(repoRoot string) string {
	stdout, _, err := s.git.run(repoRoot, "remote", "get-url", "origin")
	if err != nil {
		return "https"
	}

	remote := strings.TrimSpace(stdout)
	switch {
	case isGitHubSSHRemote(remote):
		return "ssh"
	case isGitHubHTTPSRemote(remote):
		return "https"
	default:
		return "https"
	}
}

func githubURL(slug, transport string) (string, error) {
	if !strings.Contains(slug, "/") {
		return "", fmt.Errorf("invalid GitHub repo %q", slug)
	}

	parts := strings.SplitN(slug, "/", 2)
	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if owner == "" || name == "" {
		return "", fmt.Errorf("invalid GitHub repo %q", slug)
	}

	switch transport {
	case "ssh":
		return fmt.Sprintf("git@github.com:%s/%s.git", owner, name), nil
	case "https":
		return fmt.Sprintf("https://github.com/%s/%s.git", owner, name), nil
	default:
		return "", fmt.Errorf("unsupported repo transport %q", transport)
	}
}

func expandRepoPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return filepath.Abs(path)
}

func (s repoSyncer) isGitCheckout(path string) (bool, error) {
	stdout, _, err := s.git.run(path, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(stdout) == "true", nil
}

func (s repoSyncer) hasDiverged(path string) (bool, error) {
	stdout, stderr, err := s.git.run(path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		if strings.Contains(firstNonEmpty(stderr, stdout), "no upstream configured") {
			return false, nil
		}
		return false, nil
	}

	fields := strings.Fields(stdout)
	if len(fields) != 2 {
		return false, fmt.Errorf("unexpected rev-list output %q", stdout)
	}

	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return false, err
	}
	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return false, err
	}
	return ahead > 0 && behind > 0, nil
}

func isGitHubSSHRemote(remote string) bool {
	return strings.HasPrefix(remote, "git@github.com:") || strings.HasPrefix(remote, "ssh://git@github.com/")
}

func isGitHubHTTPSRemote(remote string) bool {
	return strings.HasPrefix(remote, "https://github.com/") || strings.HasPrefix(remote, "http://github.com/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func commandErrorMessage(message string, err error) string {
	if strings.TrimSpace(message) != "" {
		return strings.TrimSpace(message)
	}
	return err.Error()
}
