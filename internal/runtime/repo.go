package runtime

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrRepoRootNotFound = errors.New("could not find repo root")

func FindRepoRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(current)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		if looksLikeRepoRoot(current) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrRepoRootNotFound
		}
		current = parent
	}
}

func looksLikeRepoRoot(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return false
	}
	return true
}
