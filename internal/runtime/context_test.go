package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewContextUsesEnvRepoRootOverride(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}

	t.Setenv("DFL_ROOT", repoRoot)

	ctx, err := NewContext("")
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}
	if ctx.RepoRoot != repoRoot {
		t.Fatalf("RepoRoot = %q, want %q", ctx.RepoRoot, repoRoot)
	}
}
