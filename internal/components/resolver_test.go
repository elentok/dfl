package components

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrefersCoreManifestOverScript(t *testing.T) {
	repoRoot := t.TempDir()
	componentRoot := filepath.Join(repoRoot, "core", "tmux")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, filepath.Join(componentRoot, "install"), "#!/usr/bin/env bash\n")
	writeFile(t, filepath.Join(componentRoot, "install.toml"), "name = \"tmux\"\n")

	component, err := Resolve(repoRoot, "tmux")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if component.Kind != KindCore {
		t.Fatalf("Kind = %q, want %q", component.Kind, KindCore)
	}
	if component.InstallerType != InstallerManifest {
		t.Fatalf("InstallerType = %q, want %q", component.InstallerType, InstallerManifest)
	}
}

func TestResolveFallsBackToExtraScript(t *testing.T) {
	repoRoot := t.TempDir()
	componentRoot := filepath.Join(repoRoot, "extra", "ssh")
	if err := os.MkdirAll(componentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	entrypoint := filepath.Join(componentRoot, "install")
	writeFile(t, entrypoint, "#!/usr/bin/env bash\n")

	component, err := Resolve(repoRoot, "ssh")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if component.Kind != KindExtra {
		t.Fatalf("Kind = %q, want %q", component.Kind, KindExtra)
	}
	if component.InstallerType != InstallerScript {
		t.Fatalf("InstallerType = %q, want %q", component.InstallerType, InstallerScript)
	}
	if component.Entrypoint != entrypoint {
		t.Fatalf("Entrypoint = %q, want %q", component.Entrypoint, entrypoint)
	}
}

func TestResolveMissingComponentReturnsError(t *testing.T) {
	_, err := Resolve(t.TempDir(), "missing")
	if !errors.Is(err, ErrComponentNotFound) {
		t.Fatalf("Resolve error = %v, want ErrComponentNotFound", err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
