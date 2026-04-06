package setup

import (
	"errors"
	"io"

	"dfl/internal/manifest"
	"dfl/internal/packagemgr"
	runtimectx "dfl/internal/runtime"
)

type PackageInstaller interface {
	Install(ctx runtimectx.Context, manager string, opts packagemgr.InstallOptions) (int, error)
}

type ComponentInstaller interface {
	Install(ctx runtimectx.Context, names []string) (int, error)
}

type RepoSyncer interface {
	Sync(ctx runtimectx.Context, defaults manifest.RepoDefaults, repo manifest.RepoSpec) (runtimectx.ResultStatus, string, error)
}

type StepExecutor interface {
	Execute(ctx runtimectx.Context, repoRoot string, step manifest.StepSpec) (runtimectx.ResultStatus, string, error)
}

func (r Runner) packageInstaller() PackageInstaller {
	if r.PackageInstaller != nil {
		return r.PackageInstaller
	}
	return packagemgr.Runner{Stdout: r.stdout(), Stderr: r.Stderr}
}

func (r Runner) componentInstaller() ComponentInstaller {
	if r.ComponentInstaller != nil {
		return r.ComponentInstaller
	}
	return componentInstallAdapter{}
}

func (r Runner) repoSyncer() RepoSyncer {
	if r.RepoSyncer != nil {
		return r.RepoSyncer
	}
	return repoSyncer{git: execGitRunner{}}
}

func (r Runner) stepExecutor() StepExecutor {
	if r.StepExecutor != nil {
		return r.StepExecutor
	}
	return shellStepExecutor{stdout: r.stdout(), stderr: r.Stderr}
}

type componentInstallAdapter struct{}

func (componentInstallAdapter) Install(runtimectx.Context, []string) (int, error) {
	return 1, errors.New("component installer is not configured")
}

type shellStepExecutor struct {
	stdout io.Writer
	stderr io.Writer
}
