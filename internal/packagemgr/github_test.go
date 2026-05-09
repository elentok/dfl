package packagemgr

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestDownloadBinaryURLUsesLatestWhenVersionMissing(t *testing.T) {
	url, err := DownloadBinaryURL("elentok/dfl", "dfl", "", "darwin", "amd64", "https://example.com/releases")
	if err != nil {
		t.Fatalf("DownloadBinaryURL returned error: %v", err)
	}
	if want := "https://example.com/releases/latest/download/dfl_Darwin_x86_64.tar.gz"; url != want {
		t.Fatalf("DownloadBinaryURL = %q, want %q", url, want)
	}
}

func TestDownloadBinaryURLUsesExplicitVersion(t *testing.T) {
	url, err := DownloadBinaryURL("elentok/dfl", "dfl", "v1.2.3", "linux", "arm64", "https://example.com/releases")
	if err != nil {
		t.Fatalf("DownloadBinaryURL returned error: %v", err)
	}
	if want := "https://example.com/releases/download/v1.2.3/dfl_Linux_arm64.tar.gz"; url != want {
		t.Fatalf("DownloadBinaryURL = %q, want %q", url, want)
	}
}

func TestDownloadBinaryURLIncludesVersionedAssetCandidate(t *testing.T) {
	urls, err := downloadBinaryURLs("example/tool-mcp", "tool-mcp", "v1.0.4", "darwin", "arm64", "https://example.com/releases")
	if err != nil {
		t.Fatalf("downloadBinaryURLs returned error: %v", err)
	}
	if len(urls) < 3 {
		t.Fatalf("len(urls) = %d, want at least 3 candidates", len(urls))
	}
	if want := "https://example.com/releases/download/v1.0.4/tool-mcp_Darwin_arm64.tar.gz"; urls[0] != want {
		t.Fatalf("urls[0] = %q, want %q", urls[0], want)
	}

	wantVersioned := "https://example.com/releases/download/v1.0.4/tool-mcp_1.0.4_darwin_arm64.tar.gz"
	if !containsString(urls, wantVersioned) {
		t.Fatalf("urls missing expected versioned asset %q; got %v", wantVersioned, urls)
	}
}

func TestDownloadBinaryURLUsesRepositoryAndBinaryName(t *testing.T) {
	url, err := DownloadBinaryURL("elentok/colr", "colr", "v1.2.3", "linux", "arm64", "")
	if err != nil {
		t.Fatalf("DownloadBinaryURL returned error: %v", err)
	}
	if want := "https://github.com/elentok/colr/releases/download/v1.2.3/colr_Linux_arm64.tar.gz"; url != want {
		t.Fatalf("DownloadBinaryURL = %q, want %q", url, want)
	}
}

func TestGitHubInstallCreatesVersionedBinaryAndSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-only in this repo")
	}

	server := newGitHubReleaseServer(t, "dfl", map[string]string{"v9.9.9": "#!/bin/sh\necho installed\n"}, "v9.9.9")
	defer server.Close()

	linkPath := filepath.Join(t.TempDir(), "bin", "dfl")
	installer := GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/dfl",
		BinaryName:     "dfl",
		ReleaseBaseURL: server.URL,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		PathEnv:        os.Getenv("PATH"),
	}

	result, err := installer.Install("v9.9.9", linkPath)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSuccess {
		t.Fatalf("Status = %q, want success", result.Status)
	}
	if result.Path != linkPath {
		t.Fatalf("Path = %q, want %q", result.Path, linkPath)
	}
	if result.Version != "v9.9.9" {
		t.Fatalf("Version = %q, want v9.9.9", result.Version)
	}

	versionedPath := managedBinaryPath(linkPath, "v9.9.9")
	if _, err := os.Stat(versionedPath); err != nil {
		t.Fatalf("versioned binary missing: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", linkPath)
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if filepath.Base(target) != "dfl-v9.9.9" {
		t.Fatalf("symlink target = %q, want basename dfl-v9.9.9", target)
	}
}

func TestGitHubInstallSkipsWhenRequestedVersionAlreadyInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-only in this repo")
	}

	linkPath := filepath.Join(t.TempDir(), "bin", "dfl")
	versionedPath := managedBinaryPath(linkPath, "v1.2.3")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(versionedPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("WriteFile versioned target: %v", err)
	}
	if err := os.Symlink(filepath.Base(versionedPath), linkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	result, err := (GitHubInstaller{Repository: "elentok/dfl", BinaryName: "dfl"}).Install("v1.2.3", linkPath)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSkipped {
		t.Fatalf("Status = %q, want skipped", result.Status)
	}
	if !strings.Contains(result.Message, "already installed") {
		t.Fatalf("Message = %q, want already installed", result.Message)
	}
}

func TestGitHubInstallSkipsWhenLatestVersionAlreadyInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-only in this repo")
	}

	server := newGitHubReleaseServer(t, "dfl", map[string]string{"v1.2.3": "#!/bin/sh\necho installed\n"}, "v1.2.3")
	defer server.Close()

	linkPath := filepath.Join(t.TempDir(), "bin", "dfl")
	versionedPath := managedBinaryPath(linkPath, "v1.2.3")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(versionedPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("WriteFile versioned target: %v", err)
	}
	if err := os.Symlink(filepath.Base(versionedPath), linkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	result, err := (GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/dfl",
		BinaryName:     "dfl",
		ReleaseBaseURL: server.URL,
	}).Install("", linkPath)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSkipped {
		t.Fatalf("Status = %q, want skipped", result.Status)
	}
	if result.Message != "latest version already installed" {
		t.Fatalf("Message = %q, want exact latest-version skip message", result.Message)
	}
}

func TestGitHubInstallReplacesPlainFileWithManagedSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-only in this repo")
	}

	server := newGitHubReleaseServer(t, "colr", map[string]string{"v2.0.0": "#!/bin/sh\necho installed\n"}, "v2.0.0")
	defer server.Close()

	linkPath := filepath.Join(t.TempDir(), "bin", "colr")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(linkPath, []byte("old"), 0o755); err != nil {
		t.Fatalf("WriteFile plain file: %v", err)
	}

	result, err := (GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/colr",
		BinaryName:     "colr",
		ReleaseBaseURL: server.URL,
	}).Install("", linkPath)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSuccess {
		t.Fatalf("Status = %q, want success", result.Status)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink after install", linkPath)
	}
	if _, err := os.Stat(managedBinaryPath(linkPath, "v2.0.0")); err != nil {
		t.Fatalf("managed binary missing: %v", err)
	}
}

func TestGitHubInstallPrunesOlderManagedVersions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-only in this repo")
	}

	server := newGitHubReleaseServer(t, "gx", map[string]string{
		"v1.0.0": "#!/bin/sh\necho v1.0.0\n",
		"v1.1.0": "#!/bin/sh\necho v1.1.0\n",
		"v1.2.0": "#!/bin/sh\necho v1.2.0\n",
		"v1.3.0": "#!/bin/sh\necho v1.3.0\n",
	}, "v1.3.0")
	defer server.Close()

	linkPath := filepath.Join(t.TempDir(), "bin", "gx")
	installer := GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/gx",
		BinaryName:     "gx",
		ReleaseBaseURL: server.URL,
	}

	for _, version := range []string{"v1.0.0", "v1.1.0", "v1.2.0", "v1.3.0"} {
		if _, err := installer.Install(version, linkPath); err != nil {
			t.Fatalf("Install %s returned error: %v", version, err)
		}
	}

	for _, version := range []string{"v1.1.0", "v1.2.0", "v1.3.0"} {
		if _, err := os.Stat(managedBinaryPath(linkPath, version)); err != nil {
			t.Fatalf("expected %s to remain: %v", version, err)
		}
	}
	if _, err := os.Stat(managedBinaryPath(linkPath, "v1.0.0")); !os.IsNotExist(err) {
		t.Fatalf("expected oldest version to be pruned, err=%v", err)
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if filepath.Base(target) != "gx-v1.3.0" {
		t.Fatalf("symlink target = %q, want basename gx-v1.3.0", target)
	}
}

func TestGitHubInstallFallsBackToVersionedAssetName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-only in this repo")
	}

	server := newVersionedAssetOnlyReleaseServer(t, "tool-mcp", "v1.0.4", "#!/bin/sh\necho installed\n")
	defer server.Close()

	linkPath := filepath.Join(t.TempDir(), "bin", "tool-mcp")
	installer := GitHubInstaller{
		Client:         server.Client(),
		Repository:     "example/tool-mcp",
		BinaryName:     "tool-mcp",
		ReleaseBaseURL: server.URL,
		GOOS:           "darwin",
		GOARCH:         "arm64",
		PathEnv:        os.Getenv("PATH"),
	}

	result, err := installer.Install("v1.0.4", linkPath)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSuccess {
		t.Fatalf("Status = %q, want success", result.Status)
	}
	if _, err := os.Stat(managedBinaryPath(linkPath, "v1.0.4")); err != nil {
		t.Fatalf("managed binary missing: %v", err)
	}
}

func newGitHubReleaseServer(t *testing.T, binaryName string, versions map[string]string, latest string) *httptest.Server {
	t.Helper()

	archives := map[string][]byte{}
	for version, contents := range versions {
		archives[version] = makeArchive(t, binaryName, contents)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/latest":
			http.Redirect(w, r, "/tag/"+latest, http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/download/"):
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) < 3 {
				http.NotFound(w, r)
				return
			}
			version := parts[1]
			archive, ok := archives[version]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
}

func newVersionedAssetOnlyReleaseServer(t *testing.T, binaryName, version, contents string) *httptest.Server {
	t.Helper()

	versionedAsset := binaryName + "_" + strings.TrimPrefix(version, "v") + "_darwin_arm64.tar.gz"
	archive := makeArchive(t, binaryName, contents)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			http.Redirect(w, r, "/tag/"+version, http.StatusFound)
			return
		}
		want := "/download/" + version + "/" + versionedAsset
		if r.URL.Path != want {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(archive)
	}))
}

func makeArchive(t *testing.T, binaryName, contents string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	data := []byte(contents)
	if err := tw.WriteHeader(&tar.Header{Name: binaryName, Mode: 0o755, Size: int64(len(data))}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tw.Close: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzw.Close: %v", err)
	}
	return buf.Bytes()
}

func containsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
