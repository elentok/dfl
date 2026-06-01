package actions

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func resolvePaths(componentRoot, source, target string) (string, string, error) {
	resolvedTarget, err := expandPath(target)
	if err != nil {
		return "", "", err
	}

	resolvedSource := source
	if !filepath.IsAbs(source) && !strings.HasPrefix(source, "~") {
		resolvedSource = filepath.Join(componentRoot, source)
	}
	resolvedSource, err = expandPath(resolvedSource)
	if err != nil {
		return "", "", err
	}

	return resolvedSource, resolvedTarget, nil
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}

	return filepath.Abs(path)
}

func samePath(currentLink, expected, linkDir string) bool {
	if filepath.IsAbs(currentLink) {
		return filepath.Clean(currentLink) == filepath.Clean(expected)
	}

	return filepath.Clean(filepath.Join(linkDir, currentLink)) == filepath.Clean(expected)
}

func sameFileContents(source, target string) (bool, error) {
	sourceData, err := os.ReadFile(source)
	if err != nil {
		return false, err
	}

	targetData, err := os.ReadFile(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return string(sourceData) == string(targetData), nil
}

func sourceMode(path string) fs.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return 0o644
	}
	return info.Mode().Perm()
}

func targetMode(target, source string) fs.FileMode {
	info, err := os.Stat(target)
	if err == nil {
		return info.Mode().Perm()
	}
	return sourceMode(source)
}

func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	cleanHome := filepath.Clean(home)
	cleanPath := filepath.Clean(path)
	if cleanPath == cleanHome {
		return "~"
	}
	if strings.HasPrefix(cleanPath, cleanHome+string(filepath.Separator)) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(cleanPath, cleanHome+string(filepath.Separator))
	}
	return cleanPath
}
