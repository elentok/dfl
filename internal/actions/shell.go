package actions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"dfl/internal/runctx"
	"dfl/internal/setuplog"
)

func (o Runner) Shell(ctx runctx.Context, name string, command []string) (int, error) {
	if len(command) == 0 {
		return 2, errors.New("shell requires a command after --")
	}

	if err := o.StepStart(name); err != nil {
		return 1, err
	}

	if ctx.DryRun {
		if _, err := fmt.Fprintf(o.stdout(), "DRY-RUN: %s\n", strings.Join(command, " ")); err != nil {
			return 1, err
		}
		if err := o.StepEnd(runctx.StatusSkipped, "dry-run"); err != nil {
			return 1, err
		}
		_ = setuplog.AppendResult(os.Getenv("DFL_LOG"), name, runctx.StatusSkipped, "dry-run", "")
		return 0, nil
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = io.MultiWriter(o.stdout(), &stdoutBuf)
	cmd.Stderr = io.MultiWriter(o.stderr(), &stderrBuf)
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		_ = o.StepEnd(runctx.StatusFailed, "command failed")
		output := combinedOutput(stdoutBuf.String(), stderrBuf.String())
		_ = setuplog.AppendResult(os.Getenv("DFL_LOG"), name, runctx.StatusFailed, "command failed", output)

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), &OutputError{Err: err, Output: output}
		}

		return 1, &OutputError{Err: err, Output: output}
	}

	if err := o.StepEnd(runctx.StatusSuccess, "done"); err != nil {
		return 1, err
	}
	_ = setuplog.AppendResult(os.Getenv("DFL_LOG"), name, runctx.StatusSuccess, "done", "")

	return 0, nil
}
