package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBootstrapScriptSyntax(t *testing.T) {
	for _, script := range []string{"bootstrap", "install-dfl.sh"} {
		cmd := exec.Command("sh", "-n", script)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sh -n %s failed: %v\n%s", script, err, string(output))
		}
	}
}

func TestBootstrapInstallsDFLAndRunsSetup(t *testing.T) {
	home := t.TempDir()
	fixtures := t.TempDir()
	binDir := filepath.Join(fixtures, "bin")
	releaseDir := filepath.Join(fixtures, "release")
	repoRoot := t.TempDir()
	repoCore := filepath.Join(repoRoot, "core")

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll binDir: %v", err)
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("MkdirAll releaseDir: %v", err)
	}
	if err := os.MkdirAll(repoCore, 0o755); err != nil {
		t.Fatalf("MkdirAll repo core: %v", err)
	}

	fakeDFL := filepath.Join(releaseDir, "dfl")
	setupMarker := filepath.Join(fixtures, "setup-ran")
	writeExecutable(t, fakeDFL, fmt.Sprintf(`#!/bin/sh
if [ "$1" = "self" ] && [ "$2" = "install" ]; then
  exit 0
fi
if [ "$1" = "setup" ]; then
  touch %s
  exit 0
fi
echo "unexpected dfl args: $*" >&2
exit 1
`, shellQuote(setupMarker)))

	assetName := "dfl_" + bootstrapOS() + "_" + bootstrapArch(t) + ".tar.gz"
	run(t, releaseDir, "tar", "-czf", filepath.Join(releaseDir, assetName), "dfl")
	if err := os.Remove(fakeDFL); err != nil {
		t.Fatalf("Remove fake dfl: %v", err)
	}

	archivePath := shellQuote(filepath.Join(releaseDir, assetName))
	curl := fmt.Sprintf(`#!/bin/sh
case "$*" in
  *%s*) cp %s "$4" ;;
  *) echo "unexpected curl args: $*" >&2; exit 1 ;;
esac
`, assetName, archivePath)
	wget := fmt.Sprintf(`#!/bin/sh
case "$*" in
  *%s*) cp %s "$2" ;;
  *) echo "unexpected wget args: $*" >&2; exit 1 ;;
esac
`, assetName, archivePath)
	writeExecutable(t, filepath.Join(binDir, "curl"), curl)
	writeExecutable(t, filepath.Join(binDir, "wget"), wget)

	writeExecutable(t, filepath.Join(repoCore, "setup"), "#!/bin/sh\nexit 0\n")

	cmd := exec.Command("sh", "/Users/david/dev/dfl/bootstrap")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bootstrap failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "dfl")); err != nil {
		t.Fatalf("installed dfl missing: %v", err)
	}
	if _, err := os.Stat(setupMarker); err != nil {
		t.Fatalf("setup marker missing: %v", err)
	}
}

func bootstrapOS() string {
	if runtime.GOOS == "darwin" {
		return "Darwin"
	}
	return "Linux"
}

func bootstrapArch(t *testing.T) string {
	t.Helper()

	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		t.Fatalf("unsupported test architecture %q", runtime.GOARCH)
		return ""
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
