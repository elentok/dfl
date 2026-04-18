package selfmgr

import (
	"dfl/internal/packagemgr"
	runtimectx "dfl/internal/runtime"
)

type Status = runtimectx.ResultStatus

type InstallResult struct {
	Status  Status
	Message string
	Path    string
	Version string
}

type Installer struct {
	DryRun         bool
	ReleaseBaseURL string
	GOOS           string
	GOARCH         string
	PathEnv        string
}

func (i Installer) Install(version, target string) (InstallResult, error) {
	result, err := packagemgr.GitHubInstaller{
		DryRun:         i.DryRun,
		Repository:     "elentok/dfl",
		BinaryName:     "dfl",
		VersionArgs:    nil,
		ReleaseBaseURL: i.ReleaseBaseURL,
		GOOS:           i.GOOS,
		GOARCH:         i.GOARCH,
		PathEnv:        i.PathEnv,
	}.Install(version, target)
	if err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Status:  result.Status,
		Message: result.Message,
		Path:    result.Path,
		Version: result.Version,
	}, nil
}

func DefaultInstallPath() (string, error) {
	return packagemgr.DefaultBinaryInstallPath("dfl")
}

func DownloadURL(version, goos, goarch, base string) (string, error) {
	return packagemgr.DownloadBinaryURL("elentok/dfl", "dfl", version, goos, goarch, base)
}
