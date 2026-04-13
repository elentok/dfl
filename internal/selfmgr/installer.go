package selfmgr

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

type Status = runtimectx.ResultStatus

type InstallResult struct {
	Status  Status
	Message string
	Path    string
	Version string
}

type Installer struct {
	Stdout         io.Writer
	Client         *http.Client
	DryRun         bool
	ReleaseBaseURL string
	GOOS           string
	GOARCH         string
	PathEnv        string
}

func (i Installer) Install(version, target string) (InstallResult, error) {
	resolvedTarget, err := i.resolveTarget(target)
	if err != nil {
		return InstallResult{}, err
	}

	if version != "" {
		if current, err := installedVersion(resolvedTarget); err == nil && versionsMatch(current, version) {
			return InstallResult{
				Status:  runtimectx.StatusSkipped,
				Message: fmt.Sprintf("dfl %s already installed at %s", current, resolvedTarget),
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
		return InstallResult{
			Status:  runtimectx.StatusSuccess,
			Message: fmt.Sprintf("would install dfl %s to %s", versionLabel, resolvedTarget),
			Path:    resolvedTarget,
		}, nil
	}

	archiveURL, err := i.downloadURL(version)
	if err != nil {
		return InstallResult{}, err
	}

	binary, err := i.downloadBinary(archiveURL)
	if err != nil {
		return InstallResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return InstallResult{}, err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(resolvedTarget), "dfl-install-*")
	if err != nil {
		return InstallResult{}, err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(binary); err != nil {
		tempFile.Close()
		return InstallResult{}, err
	}
	if err := tempFile.Chmod(0o755); err != nil {
		tempFile.Close()
		return InstallResult{}, err
	}
	if err := tempFile.Close(); err != nil {
		return InstallResult{}, err
	}

	if err := os.Rename(tempPath, resolvedTarget); err != nil {
		return InstallResult{}, err
	}

	installed, err := installedVersion(resolvedTarget)
	if err != nil {
		return InstallResult{}, err
	}
	if version != "" && !versionsMatch(installed, version) {
		return InstallResult{}, fmt.Errorf("installed version %q does not match requested version %q", installed, version)
	}

	message := fmt.Sprintf("installed dfl %s to %s", installed, resolvedTarget)
	if !pathContainsDir(i.pathEnv(), filepath.Dir(resolvedTarget)) {
		message += fmt.Sprintf("; add %s to PATH", filepath.Dir(resolvedTarget))
	}

	return InstallResult{
		Status:  runtimectx.StatusSuccess,
		Message: message,
		Path:    resolvedTarget,
		Version: installed,
	}, nil
}

func DefaultInstallPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin", "dfl"), nil
}

func DownloadURL(version, goos, goarch, base string) (string, error) {
	asset, err := assetName(goos, goarch)
	if err != nil {
		return "", err
	}
	if base == "" {
		base = "https://github.com/elentok/dfl/releases"
	}
	base = strings.TrimRight(base, "/")
	if version == "" {
		return base + "/latest/download/" + asset, nil
	}
	return base + "/download/" + version + "/" + asset, nil
}

func assetName(goos, goarch string) (string, error) {
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

	return fmt.Sprintf("dfl_%s_%s.tar.gz", osName, archName), nil
}

func (i Installer) pathEnv() string {
	if i.PathEnv != "" {
		return i.PathEnv
	}
	return os.Getenv("PATH")
}

func (i Installer) httpClient() *http.Client {
	if i.Client != nil {
		return i.Client
	}
	return http.DefaultClient
}

func (i Installer) currentGOOS() string {
	if i.GOOS != "" {
		return i.GOOS
	}
	return runtime.GOOS
}

func (i Installer) currentGOARCH() string {
	if i.GOARCH != "" {
		return i.GOARCH
	}
	return runtime.GOARCH
}

func (i Installer) resolveTarget(target string) (string, error) {
	if target == "" {
		return DefaultInstallPath()
	}
	return filepath.Abs(target)
}

func (i Installer) downloadURL(version string) (string, error) {
	return DownloadURL(version, i.currentGOOS(), i.currentGOARCH(), i.ReleaseBaseURL)
}

func (i Installer) downloadBinary(url string) ([]byte, error) {
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
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) != "dfl" {
			continue
		}
		return io.ReadAll(tr)
	}

	return nil, fmt.Errorf("archive did not contain dfl")
}

func installedVersion(path string) (string, error) {
	output, err := exec.Command(path, "version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
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
