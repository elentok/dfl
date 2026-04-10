package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"dfl/internal/components"
	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Install(ctx runtimectx.Context, names []string) (int, error) {
	if len(names) == 0 {
		return 2, errors.New("install requires at least one component")
	}

	stdout := r.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := r.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	for _, name := range names {
		component, err := components.Resolve(ctx.RepoRoot, name)
		if err != nil {
			if errors.Is(err, components.ErrComponentNotFound) {
				if _, writeErr := fmt.Fprintln(stderr); writeErr != nil {
					return 1, writeErr
				}
				if writeErr := ui.Error(stderr, fmt.Sprintf("component %q not found", name)); writeErr != nil {
					return 1, writeErr
				}
				return 1, nil
			}
			return 1, err
		}

		if err := ui.SectionHeader(stdout, fmt.Sprintf("Installing %s (%s/%s)", component.Name, component.Kind, component.InstallerType)); err != nil {
			return 1, err
		}

		if err := r.installComponent(ctx, component); err != nil {
			if _, writeErr := fmt.Fprintln(stderr); writeErr != nil {
				return 1, writeErr
			}
			if writeErr := ui.Error(stderr, fmt.Sprintf("%s failed to install: %v", component.Name, err)); writeErr != nil {
				return 1, writeErr
			}
			return 1, nil
		}

		if _, err := fmt.Fprintln(stdout); err != nil {
			return 1, err
		}
		if err := ui.Success(stdout, fmt.Sprintf("%s installed successfully", component.Name)); err != nil {
			return 1, err
		}
	}

	return 0, nil
}

func (r Runner) installComponent(ctx runtimectx.Context, component components.Component) error {
	if component.InstallerType != components.InstallerScript {
		return fmt.Errorf("unsupported installer type %q", component.InstallerType)
	}
	return r.runScript(ctx, component)
}

func (r Runner) runScript(ctx runtimectx.Context, component components.Component) error {
	cmd := exec.Command(component.Entrypoint)
	cmd.Dir = component.Root
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.Env = scriptEnv(ctx, component)
	return cmd.Run()
}

func scriptEnv(ctx runtimectx.Context, component components.Component) []string {
	env := os.Environ()
	env = append(env, "DFL_ROOT="+ctx.RepoRoot)
	env = append(env, "DFL_COMPONENT_ROOT="+component.Root)
	env = append(env, "DOTF="+ctx.RepoRoot)
	return env
}
