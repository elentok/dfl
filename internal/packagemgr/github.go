package packagemgr

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	runtimectx "dfl/internal/runtime"
)

const managedVersionsToKeep = 3

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

	linkPath, err := i.resolveTarget(target)
	if err != nil {
		return GitHubInstallResult{}, err
	}

	desiredVersion := version
	if desiredVersion == "" {
		desiredVersion, err = i.latestVersion()
		if err != nil {
			if i.DryRun {
				return GitHubInstallResult{
					Status:  runtimectx.StatusSuccess,
					Message: fmt.Sprintf("would install %s latest release to %s", binaryName, linkPath),
					Path:    linkPath,
				}, nil
			}
			return GitHubInstallResult{}, err
		}
	}

	currentVersion, currentManaged, err := currentLinkedVersion(linkPath, binaryName)
	if err != nil {
		return GitHubInstallResult{}, err
	}
	if currentManaged && versionsMatch(currentVersion, desiredVersion) {
		message := fmt.Sprintf("%s %s already installed at %s", binaryName, currentVersion, linkPath)
		if version == "" {
			message = "latest version already installed"
		}
		return GitHubInstallResult{
			Status:  runtimectx.StatusSkipped,
			Message: message,
			Path:    linkPath,
			Version: currentVersion,
		}, nil
	}

	versionedPath := managedBinaryPath(linkPath, desiredVersion)
	if i.DryRun {
		return GitHubInstallResult{
			Status:  runtimectx.StatusSuccess,
			Message: fmt.Sprintf("would install %s %s to %s", binaryName, desiredVersion, linkPath),
			Path:    linkPath,
			Version: desiredVersion,
		}, nil
	}

	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return GitHubInstallResult{}, err
	}

	if _, err := os.Stat(versionedPath); err != nil {
		if !os.IsNotExist(err) {
			return GitHubInstallResult{}, err
		}

		archiveURL, err := i.downloadURL(desiredVersion)
		if err != nil {
			return GitHubInstallResult{}, err
		}

		binary, err := i.downloadBinary(archiveURL)
		if err != nil {
			return GitHubInstallResult{}, err
		}

		if err := writeInstalledBinary(versionedPath, binaryName, binary); err != nil {
			return GitHubInstallResult{}, err
		}
	}

	if err := updateSymlink(linkPath, versionedPath); err != nil {
		return GitHubInstallResult{}, err
	}

	if err := pruneOldVersions(linkPath, binaryName, versionedPath, managedVersionsToKeep); err != nil {
		return GitHubInstallResult{}, err
	}

	message := fmt.Sprintf("installed %s %s to %s", binaryName, desiredVersion, linkPath)
	if !pathContainsDir(i.pathEnv(), filepath.Dir(linkPath)) {
		message += fmt.Sprintf("; add %s to PATH", filepath.Dir(linkPath))
	}

	return GitHubInstallResult{
		Status:  runtimectx.StatusSuccess,
		Message: message,
		Path:    linkPath,
		Version: desiredVersion,
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

func (i GitHubInstaller) latestVersion() (string, error) {
	repository, err := i.repository()
	if err != nil {
		return "", err
	}

	base := i.ReleaseBaseURL
	if base == "" {
		base = releaseBaseURL(repository)
	}
	resp, err := i.httpClient().Get(strings.TrimRight(base, "/") + "/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Request == nil || resp.Request.URL == nil {
		return "", fmt.Errorf("could not resolve latest release URL")
	}

	version := versionFromReleasePath(resp.Request.URL.Path)
	if version == "" {
		return "", fmt.Errorf("could not determine latest version from %q", resp.Request.URL.Path)
	}
	return version, nil
}

func writeInstalledBinary(path, binaryName string, contents []byte) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), binaryName+"-install-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(contents); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0o755); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func managedBinaryPath(linkPath, version string) string {
	return linkPath + "-" + version
}

func currentLinkedVersion(linkPath, binaryName string) (string, bool, error) {
	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "", false, nil
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", false, err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	base := filepath.Base(target)
	prefix := binaryName + "-"
	if !strings.HasPrefix(base, prefix) {
		return "", false, nil
	}
	version := strings.TrimPrefix(base, prefix)
	if version == "" {
		return "", false, nil
	}
	return version, true, nil
}

func updateSymlink(linkPath, targetPath string) error {
	info, err := os.Lstat(linkPath)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", linkPath)
		}
		if err := os.Remove(linkPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	relativeTarget, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
	if err != nil {
		return err
	}
	return os.Symlink(relativeTarget, linkPath)
}

func pruneOldVersions(linkPath, binaryName, currentTarget string, keep int) error {
	dir := filepath.Dir(linkPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	prefix := binaryName + "-"
	var managed []managedVersion
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		version := strings.TrimPrefix(name, prefix)
		if version == "" {
			continue
		}
		managed = append(managed, managedVersion{
			path:    filepath.Join(dir, name),
			version: version,
		})
	}

	slices.SortStableFunc(managed, func(a, b managedVersion) int {
		return compareVersionsDesc(a.version, b.version)
	})

	keepPaths := map[string]struct{}{}
	if currentTarget != "" {
		keepPaths[currentTarget] = struct{}{}
	}
	for _, item := range managed {
		if len(keepPaths) >= keep {
			break
		}
		keepPaths[item.path] = struct{}{}
	}

	for _, item := range managed {
		if _, ok := keepPaths[item.path]; ok {
			continue
		}
		if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

type managedVersion struct {
	path    string
	version string
}

func releaseBaseURL(repository string) string {
	return "https://github.com/" + strings.Trim(repository, "/") + "/releases"
}

func versionFromReleasePath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for idx := 0; idx < len(parts)-1; idx++ {
		if parts[idx] == "tag" || parts[idx] == "download" {
			return parts[idx+1]
		}
	}
	return ""
}

func versionsMatch(actual, requested string) bool {
	return strings.TrimPrefix(actual, "v") == strings.TrimPrefix(requested, "v")
}

func compareVersionsDesc(left, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	limit := len(leftParts)
	if len(rightParts) > limit {
		limit = len(rightParts)
	}
	for idx := 0; idx < limit; idx++ {
		if idx >= len(leftParts) {
			return 1
		}
		if idx >= len(rightParts) {
			return -1
		}
		cmp := compareVersionPart(leftParts[idx], rightParts[idx])
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}

func versionParts(version string) []string {
	trimmed := strings.TrimPrefix(version, "v")
	return strings.FieldsFunc(trimmed, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z')
	})
}

func compareVersionPart(left, right string) int {
	leftNum, leftErr := strconv.Atoi(left)
	rightNum, rightErr := strconv.Atoi(right)
	if leftErr == nil && rightErr == nil {
		switch {
		case leftNum > rightNum:
			return -1
		case leftNum < rightNum:
			return 1
		default:
			return 0
		}
	}
	switch {
	case left > right:
		return -1
	case left < right:
		return 1
	default:
		return 0
	}
}

func pathContainsDir(pathEnv, dir string) bool {
	for _, entry := range filepath.SplitList(pathEnv) {
		if entry == dir {
			return true
		}
	}
	return false
}
