package packagemgr

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	runtimectx "dfl/internal/runtime"
)

type GitHubInstallResult struct {
	Status  runtimectx.ResultStatus
	Message string
	Path    string
	Version string
}

type GitHubInstaller struct {
	Client         *http.Client
	DryRun         bool
	Repository     string
	BinaryName     string
	VersionArgs    []string
	ReleaseBaseURL string
	GOOS           string
	GOARCH         string
	PathEnv        string
}

func (i GitHubInstaller) Install(version, target string) (GitHubInstallResult, error) {
	binaryName, err := i.binaryName()
	if err != nil {
		return GitHubInstallResult{}, err
	}

	resolvedTarget, err := i.resolveTarget(target)
	if err != nil {
		return GitHubInstallResult{}, err
	}

	if version != "" && i.supportsInstalledVersion() {
		if current, err := i.installedVersion(resolvedTarget); err == nil && versionsMatch(current, version) {
			return GitHubInstallResult{
				Status:  runtimectx.StatusSkipped,
				Message: fmt.Sprintf("%s %s already installed at %s", binaryName, current, resolvedTarget),
				Path:    resolvedTarget,
				Version: current,
			}, nil
		}
	}

	if i.DryRun {
		versionLabel := "latest release"
		if version != "" {
			versionLabel = version
		}
		return GitHubInstallResult{
			Status:  runtimectx.StatusSuccess,
			Message: fmt.Sprintf("would install %s %s to %s", binaryName, versionLabel, resolvedTarget),
			Path:    resolvedTarget,
		}, nil
	}

	archiveURL, err := i.downloadURL(version)
	if err != nil {
		return GitHubInstallResult{}, err
	}

	binary, err := i.downloadBinary(archiveURL)
	if err != nil {
		return GitHubInstallResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return GitHubInstallResult{}, err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(resolvedTarget), binaryName+"-install-*")
	if err != nil {
		return GitHubInstallResult{}, err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(binary); err != nil {
		tempFile.Close()
		return GitHubInstallResult{}, err
	}
	if err := tempFile.Chmod(0o755); err != nil {
		tempFile.Close()
		return GitHubInstallResult{}, err
	}
	if err := tempFile.Close(); err != nil {
		return GitHubInstallResult{}, err
	}

	if err := os.Rename(tempPath, resolvedTarget); err != nil {
		return GitHubInstallResult{}, err
	}

	installed := ""
	if i.supportsInstalledVersion() {
		installed, err = i.installedVersion(resolvedTarget)
		if err != nil {
			return GitHubInstallResult{}, err
		}
		if version != "" && !versionsMatch(installed, version) {
			return GitHubInstallResult{}, fmt.Errorf("installed version %q does not match requested version %q", installed, version)
		}
	}

	message := fmt.Sprintf("installed %s to %s", binaryName, resolvedTarget)
	if installed != "" {
		message = fmt.Sprintf("installed %s %s to %s", binaryName, installed, resolvedTarget)
	}
	if !pathContainsDir(i.pathEnv(), filepath.Dir(resolvedTarget)) {
		message += fmt.Sprintf("; add %s to PATH", filepath.Dir(resolvedTarget))
	}

	return GitHubInstallResult{
		Status:  runtimectx.StatusSuccess,
		Message: message,
		Path:    resolvedTarget,
		Version: installed,
	}, nil
}

func DefaultBinaryInstallPath(binaryName string) (string, error) {
	if binaryName == "" {
		return "", fmt.Errorf("binary name is required")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin", binaryName), nil
}

func DownloadBinaryURL(repository, binaryName, version, goos, goarch, base string) (string, error) {
	asset, err := assetName(binaryName, goos, goarch)
	if err != nil {
		return "", err
	}
	if base == "" {
		base = releaseBaseURL(repository)
	}
	base = strings.TrimRight(base, "/")
	if version == "" {
		return base + "/latest/download/" + asset, nil
	}
	return base + "/download/" + version + "/" + asset, nil
}

func assetName(binaryName, goos, goarch string) (string, error) {
	if binaryName == "" {
		return "", fmt.Errorf("binary name is required")
	}

	var osName string
	switch goos {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	default:
		return "", fmt.Errorf("unsupported OS %q", goos)
	}

	var archName string
	switch goarch {
	case "amd64":
		archName = "x86_64"
	case "arm64":
		archName = "arm64"
	default:
		return "", fmt.Errorf("unsupported architecture %q", goarch)
	}

	return fmt.Sprintf("%s_%s_%s.tar.gz", binaryName, osName, archName), nil
}

func (i GitHubInstaller) pathEnv() string {
	if i.PathEnv != "" {
		return i.PathEnv
	}
	return os.Getenv("PATH")
}

func (i GitHubInstaller) httpClient() *http.Client {
	if i.Client != nil {
		return i.Client
	}
	return http.DefaultClient
}

func (i GitHubInstaller) currentGOOS() string {
	if i.GOOS != "" {
		return i.GOOS
	}
	return runtime.GOOS
}

func (i GitHubInstaller) currentGOARCH() string {
	if i.GOARCH != "" {
		return i.GOARCH
	}
	return runtime.GOARCH
}

func (i GitHubInstaller) resolveTarget(target string) (string, error) {
	if target == "" {
		binaryName, err := i.binaryName()
		if err != nil {
			return "", err
		}
		return DefaultBinaryInstallPath(binaryName)
	}
	return filepath.Abs(target)
}

func (i GitHubInstaller) downloadURL(version string) (string, error) {
	repository, err := i.repository()
	if err != nil {
		return "", err
	}
	binaryName, err := i.binaryName()
	if err != nil {
		return "", err
	}
	return DownloadBinaryURL(repository, binaryName, version, i.currentGOOS(), i.currentGOARCH(), i.ReleaseBaseURL)
}

func (i GitHubInstaller) downloadBinary(url string) ([]byte, error) {
	resp, err := i.httpClient().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	binaryName, err := i.binaryName()
	if err != nil {
		return nil, err
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		return io.ReadAll(tr)
	}

	return nil, fmt.Errorf("archive did not contain %s", binaryName)
}

func (i GitHubInstaller) installedVersion(path string) (string, error) {
	output, err := exec.Command(path, i.versionArgs()...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (i GitHubInstaller) supportsInstalledVersion() bool {
	return len(i.versionArgs()) > 0
}

func (i GitHubInstaller) repository() (string, error) {
	if i.Repository == "" {
		return "", fmt.Errorf("repository is required")
	}
	repo := strings.Trim(i.Repository, "/")
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid GitHub repository %q", i.Repository)
	}
	return parts[0] + "/" + parts[1], nil
}

func (i GitHubInstaller) binaryName() (string, error) {
	if i.BinaryName != "" {
		return i.BinaryName, nil
	}
	repo, err := i.repository()
	if err != nil {
		return "", err
	}
	return filepath.Base(repo), nil
}

func (i GitHubInstaller) versionArgs() []string {
	if i.VersionArgs != nil {
		return append([]string(nil), i.VersionArgs...)
	}
	return []string{"version"}
}

func releaseBaseURL(repository string) string {
	return "https://github.com/" + strings.Trim(repository, "/") + "/releases"
}

func versionsMatch(actual, requested string) bool {
	return strings.TrimPrefix(actual, "v") == strings.TrimPrefix(requested, "v")
}

func pathContainsDir(pathEnv, dir string) bool {
	for _, entry := range filepath.SplitList(pathEnv) {
		if entry == dir {
			return true
		}
	}
	return false
}
