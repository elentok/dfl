package setup

import (
	"io"
	"path/filepath"

	"dfl/internal/manifest"
	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

type Options struct {
	Components   []string
	SkipPackages bool
	SkipRepos    bool
}

type Runner struct {
	Stdout             io.Writer
	Stderr             io.Writer
	PackageInstaller   PackageInstaller
	ComponentInstaller ComponentInstaller
	RepoSyncer         RepoSyncer
	StepExecutor       StepExecutor
}

func (r Runner) Run(ctx runtimectx.Context, opts Options) (int, error) {
	setupPath := filepath.Join(ctx.RepoRoot, "setup", "default.yaml")
	setupManifest, err := manifest.ParseSetupFile(setupPath)
	if err != nil {
		return 1, err
	}

	machine := detectMachineContext(ctx)
	if !manifest.MatchesWhen(setupManifest.When, machine) {
		if err := ui.Step(r.stdout(), "Evaluating setup constraints", func() (runtimectx.ResultStatus, string, error) {
			return runtimectx.StatusSkipped, "setup does not apply to this machine", nil
		}); err != nil {
			return 1, err
		}
		return 0, nil
	}

	if err := r.runRepos(ctx, setupManifest, machine, opts); err != nil {
		return 1, err
	}
	if err := r.runPackages(ctx, setupManifest, machine, opts); err != nil {
		return 1, err
	}
	if err := r.runComponents(ctx, setupManifest, machine, opts); err != nil {
		return 1, err
	}
	if err := r.runSteps(ctx, setupManifest, machine); err != nil {
		return 1, err
	}

	return 0, nil
}

func (r Runner) stdout() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return io.Discard
}
