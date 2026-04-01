package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Context struct {
	RepoRoot string
	OS       OSType
	DryRun   bool
}

func NewContext(startDir string) (Context, error) {
	if repoRoot, ok := repoRootFromEnv(); ok {
		return Context{
			RepoRoot: repoRoot,
			OS:       DetectOS(),
		}, nil
	}

	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return Context{}, err
		}
	}

	repoRoot, err := FindRepoRoot(startDir)
	if err != nil {
		return Context{}, err
	}

	return Context{
		RepoRoot: repoRoot,
		OS:       DetectOS(),
	}, nil
}

func repoRootFromEnv() (string, bool) {
	for _, key := range []string{"DFL_ROOT", "DOTF"} {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		absValue, err := filepath.Abs(value)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absValue); errors.Is(err, os.ErrNotExist) {
			continue
		}
		return absValue, true
	}

	return "", false
}

func DetectOS() OSType {
	switch runtime.GOOS {
	case "darwin":
		return OSMac
	case "linux":
		if isWSL() {
			return OSWSL
		}
		return OSLinux
	default:
		return OSUnknown
	}
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
