package setup

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/runtimecmd"
	"dfl/internal/setuplog"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Run(ctx runtimectx.Context) (int, error) {
	logFile, err := os.CreateTemp("", "dfl-setup-*.jsonl")
	if err != nil {
		return 1, err
	}
	logPath := logFile.Name()
	if err := logFile.Close(); err != nil {
		return 1, err
	}
	defer os.Remove(logPath)

	setupPath := filepath.Join(ctx.RepoRoot, "core", "setup")
	cmd := exec.Command(setupPath)
	cmd.Dir = ctx.RepoRoot
	cmd.Stdout = r.stdout()
	cmd.Stderr = r.stderr()
	cmd.Env = setupEnv(ctx, logPath)

	runErr := cmd.Run()
	if renderErr := r.renderSummary(logPath); renderErr != nil && runErr == nil {
		return 1, renderErr
	}
	if runErr != nil {
		return 1, runErr
	}

	return 0, nil
}

func setupEnv(ctx runtimectx.Context, logPath string) []string {
	env := runtimecmd.WithExecutableOnPath(os.Environ())
	env = append(env, "DFL_ROOT="+ctx.RepoRoot)
	env = append(env, "DFL_COMPONENT_ROOT="+filepath.Join(ctx.RepoRoot, "core"))
	env = append(env, "DOTF="+ctx.RepoRoot)
	env = append(env, "DFL_LOG="+logPath)
	if ctx.DryRun {
		env = append(env, "DFL_DRY_RUN=1")
	}
	return env
}

func (r Runner) renderSummary(logPath string) error {
	steps, err := setuplog.Read(logPath)
	if err != nil {
		return nil
	}
	return setuplog.RenderSummary(r.stdout(), steps)
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
