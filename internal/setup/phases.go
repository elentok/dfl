package setup

import (
	"fmt"

	"dfl/internal/manifest"
	"dfl/internal/packagemgr"
	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

func (r Runner) runRepos(ctx runtimectx.Context, m manifest.SetupManifest, machine manifest.MachineContext, opts Options) error {
	if opts.SkipRepos {
		return ui.Step(r.stdout(), "Syncing setup repos", func() (runtimectx.ResultStatus, string, error) {
			return runtimectx.StatusSkipped, "skipped by flag", nil
		})
	}

	syncer := r.repoSyncer()
	return ui.Step(r.stdout(), "Syncing setup repos", func() (runtimectx.ResultStatus, string, error) {
		count := 0
		for _, repo := range m.Repos {
			if !manifest.RepoMatches(repo, machine) {
				continue
			}
			count++
			status, msg, err := syncer.Sync(ctx, m.RepoDefaults, repo)
			if err != nil {
				return "", "", err
			}
			if status == runtimectx.StatusFailed {
				return status, msg, nil
			}
		}
		if count == 0 {
			return runtimectx.StatusSkipped, "no matching repos", nil
		}
		return runtimectx.StatusSkipped, "repo sync not implemented yet", nil
	})
}

func (r Runner) runPackages(ctx runtimectx.Context, m manifest.SetupManifest, machine manifest.MachineContext, opts Options) error {
	if opts.SkipPackages {
		return ui.Step(r.stdout(), "Installing setup packages", func() (runtimectx.ResultStatus, string, error) {
			return runtimectx.StatusSkipped, "skipped by flag", nil
		})
	}

	pkgRunner := r.packageInstaller()
	return ui.Step(r.stdout(), "Installing setup packages", func() (runtimectx.ResultStatus, string, error) {
		count := 0
		for _, pkg := range m.Packages {
			if !manifest.PackageMatches(pkg, machine) {
				continue
			}
			count++
			code, err := pkgRunner.Install(ctx, pkg.Manager, packagemgr.InstallOptions{
				Packages: pkg.Names,
				Tap:      pkg.Tap,
				Cask:     pkg.Cask,
			})
			if err != nil {
				return "", "", err
			}
			if code != 0 {
				return runtimectx.StatusFailed, fmt.Sprintf("package install for %s failed", pkg.Manager), nil
			}
		}
		if count == 0 {
			return runtimectx.StatusSkipped, "no matching package groups", nil
		}
		return runtimectx.StatusSuccess, "setup packages processed", nil
	})
}

func (r Runner) runComponents(ctx runtimectx.Context, m manifest.SetupManifest, machine manifest.MachineContext, opts Options) error {
	selected, err := filterComponents(m.Components, machine, opts.Components)
	if err != nil {
		return err
	}

	return ui.Step(r.stdout(), "Installing setup components", func() (runtimectx.ResultStatus, string, error) {
		if len(selected) == 0 {
			return runtimectx.StatusSkipped, "no matching components", nil
		}
		code, err := r.componentInstaller().Install(ctx, selected)
		if err != nil {
			return "", "", err
		}
		if code != 0 {
			return runtimectx.StatusFailed, "component installation failed", nil
		}
		return runtimectx.StatusSuccess, fmt.Sprintf("processed %d components", len(selected)), nil
	})
}

func (r Runner) runSteps(ctx runtimectx.Context, m manifest.SetupManifest, machine manifest.MachineContext) error {
	executor := r.stepExecutor()
	for _, step := range m.Steps {
		step := step
		if !manifest.StepMatches(step, machine) {
			if err := ui.Step(r.stdout(), step.Name, func() (runtimectx.ResultStatus, string, error) {
				return runtimectx.StatusSkipped, "not applicable on this machine", nil
			}); err != nil {
				return err
			}
			continue
		}
		if err := ui.Step(r.stdout(), step.Name, func() (runtimectx.ResultStatus, string, error) {
			return executor.Execute(ctx, ctx.RepoRoot, step)
		}); err != nil {
			return err
		}
	}
	return nil
}
