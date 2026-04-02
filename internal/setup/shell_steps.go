package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"dfl/internal/manifest"
	runtimectx "dfl/internal/runtime"
)

func (s shellStepExecutor) Execute(ctx runtimectx.Context, repoRoot string, step manifest.StepSpec) (runtimectx.ResultStatus, string, error) {
	if step.Run == "" {
		return runtimectx.StatusSkipped, "no command", nil
	}
	if step.If != "" {
		ok, err := runPredicate(repoRoot, step.CWD, step.If)
		if err != nil {
			return "", "", err
		}
		if !ok {
			return runtimectx.StatusSkipped, "if condition failed", nil
		}
	}
	if step.IfNot != "" {
		ok, err := runPredicate(repoRoot, step.CWD, step.IfNot)
		if err != nil {
			return "", "", err
		}
		if ok {
			return runtimectx.StatusSkipped, "if_not condition matched", nil
		}
	}

	if ctx.DryRun {
		return runtimectx.StatusSuccess, fmt.Sprintf("would run %s", step.Run), nil
	}

	command := []string{"sh", "-c", step.Run}
	if step.Sudo {
		command = append([]string{"sudo"}, command...)
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	cmd.Dir = resolveStepDir(repoRoot, step.CWD)
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return runtimectx.StatusFailed, "command failed", nil
	}
	return runtimectx.StatusSuccess, "done", nil
}

func runPredicate(repoRoot, cwd, expr string) (bool, error) {
	cmd := exec.Command("sh", "-c", expr)
	cmd.Dir = resolveStepDir(repoRoot, cwd)
	cmd.Env = os.Environ()
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, err
}

func resolveStepDir(repoRoot, cwd string) string {
	if cwd == "" {
		return repoRoot
	}
	if filepath.IsAbs(cwd) {
		return cwd
	}
	return filepath.Join(repoRoot, cwd)
}
