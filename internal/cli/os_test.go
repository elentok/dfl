package cli

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
)

func TestRunOSPredicateReturnsSuccessForCurrentPlatform(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	t.Setenv("DFL_ROOT", repoRoot)

	app := NewApp()
	predicate := "is-linux"
	if goruntime.GOOS == "darwin" {
		predicate = "is-mac"
	}

	code, err := app.Run([]string{"os", predicate})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0 for %s", code, predicate)
	}
}
