package setup

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/runtimecmd"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Run(ctx runtimectx.Context) (int, error) {
	setupPath := filepath.Join(ctx.RepoRoot, "core", "setup")
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
	env := runtimecmd.WithExecutableOnPath(os.Environ())
	env = append(env, "DFL_ROOT="+ctx.RepoRoot)
	env = append(env, "DFL_COMPONENT_ROOT="+filepath.Join(ctx.RepoRoot, "core"))
	env = append(env, "DOTF="+ctx.RepoRoot)
	if ctx.DryRun {
		env = append(env, "DFL_DRY_RUN=1")
	}
	return env
}

func (r Runner) stdout() io.Writer {
	return writerOrDiscard(r.Stdout)
}

func (r Runner) stderr() io.Writer {
	return writerOrDiscard(r.Stderr)
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return io.Discard
}
