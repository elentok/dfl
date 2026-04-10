package setup

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	runtimectx "dfl/internal/runtime"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Run(ctx runtimectx.Context) (int, error) {
	setupPath := filepath.Join(ctx.RepoRoot, "setup")
	cmd := exec.Command(setupPath)
	cmd.Dir = ctx.RepoRoot
	cmd.Stdout = r.stdout()
	cmd.Stderr = r.stderr()
	cmd.Env = setupEnv(ctx)
	if err := cmd.Run(); err != nil {
		return 1, err
	}

	return 0, nil
}

func setupEnv(ctx runtimectx.Context) []string {
	env := os.Environ()
	env = append(env, "DFL_ROOT="+ctx.RepoRoot)
	env = append(env, "DOTF="+ctx.RepoRoot)
	if ctx.DryRun {
		env = append(env, "DFL_DRY_RUN=1")
	}
	return env
}

func (r Runner) stdout() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return io.Discard
}

func (r Runner) stderr() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return io.Discard
}
