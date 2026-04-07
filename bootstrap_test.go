package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBootstrapScriptSyntax(t *testing.T) {
	cmd := exec.Command("sh", "-n", "bootstrap")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sh -n bootstrap failed: %v\n%s", err, string(output))
	}
}

func TestBootstrapInstallsDFLAndRunsSetup(t *testing.T) {
	home := t.TempDir()
	fixtures := t.TempDir()
	binDir := filepath.Join(fixtures, "bin")
	releaseDir := filepath.Join(fixtures, "release")

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll binDir: %v", err)
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("MkdirAll releaseDir: %v", err)
	}

	fakeDFL := filepath.Join(releaseDir, "dfl")
	setupMarker := filepath.Join(fixtures, "setup-ran")
	writeExecutable(t, fakeDFL, "#!/bin/sh\n[ \"$*\" = \"setup\" ] || { echo \"unexpected dfl args: $*\" >&2; exit 1; }\ntouch "+shellQuote(setupMarker)+"\n")

	assetName := "dfl_" + bootstrapOS() + "_" + bootstrapArch(t) + ".tar.gz"
	run(t, releaseDir, "tar", "-czf", filepath.Join(releaseDir, assetName), "dfl")
	if err := os.Remove(fakeDFL); err != nil {
		t.Fatalf("Remove fake dfl: %v", err)
	}

	archivePath := shellQuote(filepath.Join(releaseDir, assetName))
	curl := "#!/bin/sh\ncase \"$*\" in\n  *" + assetName + "*) cp " + archivePath + " \"$4\" ;;\n  *) echo \"unexpected curl args: $*\" >&2; exit 1 ;;\nesac\n"
	wget := "#!/bin/sh\ncase \"$*\" in\n  *" + assetName + "*) cp " + archivePath + " \"$2\" ;;\n  *) echo \"unexpected wget args: $*\" >&2; exit 1 ;;\nesac\n"
	writeExecutable(t, filepath.Join(binDir, "curl"), curl)
	writeExecutable(t, filepath.Join(binDir, "wget"), wget)

	cmd := exec.Command("sh", "./bootstrap")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
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
