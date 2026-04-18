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

func TestDownloadBinaryURLUsesRepositoryAndBinaryName(t *testing.T) {
	url, err := DownloadBinaryURL("elentok/colr", "colr", "v1.2.3", "linux", "arm64", "")
	if err != nil {
		t.Fatalf("DownloadBinaryURL returned error: %v", err)
	}
	if want := "https://github.com/elentok/colr/releases/download/v1.2.3/colr_Linux_arm64.tar.gz"; url != want {
		t.Fatalf("DownloadBinaryURL = %q, want %q", url, want)
	}
}

func TestGitHubInstallDownloadsAndValidatesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test binary is Unix-only")
	}

	archive := makeArchive(t, "dfl", "#!/bin/sh\necho v9.9.9\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "bin", "dfl")
	installer := GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/dfl",
		BinaryName:     "dfl",
		ReleaseBaseURL: server.URL,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		PathEnv:        os.Getenv("PATH"),
	}

	result, err := installer.Install("v9.9.9", target)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSuccess {
		t.Fatalf("Status = %q, want success", result.Status)
	}
	if result.Version != "v9.9.9" {
		t.Fatalf("Version = %q, want v9.9.9", result.Version)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
}

func TestGitHubInstallSkipsWhenRequestedVersionAlreadyInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test binary is Unix-only")
	}

	target := filepath.Join(t.TempDir(), "dfl")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho v1.2.3\n"), 0o755); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	result, err := (GitHubInstaller{Repository: "elentok/dfl", BinaryName: "dfl"}).Install("v1.2.3", target)
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
		t.Skip("shell script test binary is Unix-only")
	}

	target := filepath.Join(t.TempDir(), "dfl")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho v1.2.3\n"), 0o755); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			http.Redirect(w, r, "/tag/v1.2.3", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	result, err := (GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/dfl",
		BinaryName:     "dfl",
		ReleaseBaseURL: server.URL,
	}).Install("", target)
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

func TestGitHubInstallGenericBinaryWithoutVersionCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test binary is Unix-only")
	}

	archive := makeArchive(t, "colr", "#!/bin/sh\necho installed\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "bin", "colr")
	installer := GitHubInstaller{
		Client:         server.Client(),
		Repository:     "elentok/colr",
		BinaryName:     "colr",
		VersionArgs:    []string{},
		ReleaseBaseURL: server.URL,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		PathEnv:        os.Getenv("PATH"),
	}

	result, err := installer.Install("", target)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Status != runtimectx.StatusSuccess {
		t.Fatalf("Status = %q, want success", result.Status)
	}
	if result.Version != "" {
		t.Fatalf("Version = %q, want empty", result.Version)
	}
	if !strings.Contains(result.Message, "installed colr to "+target) {
		t.Fatalf("Message = %q, want generic install output", result.Message)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
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
