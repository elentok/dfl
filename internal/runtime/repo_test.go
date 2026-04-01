package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRootFromNestedDirectory(t *testing.T) {
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	nestedDir := filepath.Join(repoRoot, "a", "b", "c")

	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll nested dir: %v", err)
	}

	got, err := FindRepoRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindRepoRoot returned error: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("FindRepoRoot = %q, want %q", got, repoRoot)
	}
}

func TestFindRepoRootReturnsErrorOutsideRepo(t *testing.T) {
	_, err := FindRepoRoot(t.TempDir())
	if err == nil {
		t.Fatal("FindRepoRoot returned nil error, want error")
	}
	if !errors.Is(err, ErrRepoRootNotFound) {
		t.Fatalf("FindRepoRoot error = %v, want ErrRepoRootNotFound", err)
	}
}
