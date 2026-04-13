package selfmgr

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

func TestDownloadURLUsesLatestWhenVersionMissing(t *testing.T) {
	url, err := DownloadURL("", "darwin", "amd64", "https://example.com/releases")
	if err != nil {
		t.Fatalf("DownloadURL returned error: %v", err)
	}
	if want := "https://example.com/releases/latest/download/dfl_Darwin_x86_64.tar.gz"; url != want {
		t.Fatalf("DownloadURL = %q, want %q", url, want)
	}
}

func TestDownloadURLUsesExplicitVersion(t *testing.T) {
	url, err := DownloadURL("v1.2.3", "linux", "arm64", "https://example.com/releases")
	if err != nil {
		t.Fatalf("DownloadURL returned error: %v", err)
	}
	if want := "https://example.com/releases/download/v1.2.3/dfl_Linux_arm64.tar.gz"; url != want {
		t.Fatalf("DownloadURL = %q, want %q", url, want)
	}
}

func TestInstallDownloadsAndValidatesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test binary is Unix-only")
	}

	archive := makeArchive(t, "#!/bin/sh\necho v9.9.9\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "bin", "dfl")
	installer := Installer{
		Client:         server.Client(),
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

func TestInstallSkipsWhenRequestedVersionAlreadyInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test binary is Unix-only")
	}

	target := filepath.Join(t.TempDir(), "dfl")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho v1.2.3\n"), 0o755); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	result, err := (Installer{}).Install("v1.2.3", target)
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

func makeArchive(t *testing.T, contents string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	data := []byte(contents)
	if err := tw.WriteHeader(&tar.Header{Name: "dfl", Mode: 0o755, Size: int64(len(data))}); err != nil {
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
