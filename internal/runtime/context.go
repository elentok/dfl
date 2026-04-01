package runtime

import (
	"os"
	"runtime"
	"strings"
)

type Context struct {
	RepoRoot string
	OS       OSType
	DryRun   bool
}

func NewContext(startDir string) (Context, error) {
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
