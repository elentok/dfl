package cli

import (
	"dfl/internal/runtime"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type App struct {
	stdout io.Writer
	stderr io.Writer
	dryRun bool
}

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func NewApp() *App {
	return &App{}
}

func (a *App) SetStdout(w io.Writer) {
	a.stdout = w
}

func (a *App) SetStderr(w io.Writer) {
	a.stderr = w
}

func (a *App) Run(args []string) (int, error) {
	a.dryRun = false

	root := a.newRootCommand()
	root.SetArgs(args)
	root.SetOut(a.stdoutWriter())
	root.SetErr(a.stderrWriter())

	err := root.Execute()
	if err == nil {
		return 0, nil
	}

	var codeErr exitError
	if errors.As(err, &codeErr) {
		return codeErr.code, codeErr.err
	}

	if strings.HasPrefix(err.Error(), "unknown command ") {
		fmt.Fprintln(a.stderrWriter(), err)
		return 2, nil
	}

	return 1, err
}

func (a *App) runPlaceholder(name string, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		fmt.Fprintf(a.stdoutWriter(), "%s command is not implemented yet\n", name)
		return nil
	}
	fmt.Fprintf(a.stdoutWriter(), "%s command is not implemented yet\n", name)
	return nil
}

func (a *App) runtimeContext() (runtime.Context, error) {
	ctx, err := runtime.NewContext("")
	if err != nil {
		return runtime.Context{}, err
	}
	ctx.DryRun = a.dryRun
	return ctx, nil
}

func (a *App) stdoutWriter() io.Writer {
	if a.stdout != nil {
		return a.stdout
	}
	return os.Stdout
}

func (a *App) stderrWriter() io.Writer {
	if a.stderr != nil {
		return a.stderr
	}
	return os.Stderr
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func statusFromKind(kind string) (runtime.ResultStatus, error) {
	switch kind {
	case "success":
		return runtime.StatusSuccess, nil
	case "skip":
		return runtime.StatusSkipped, nil
	case "error":
		return runtime.StatusFailed, nil
	default:
		return "", fmt.Errorf("step-end requires one of --status success|skip|error or the matching shortcut flag")
	}
}

func componentRoot() string {
	if value := os.Getenv("DFL_COMPONENT_ROOT"); value != "" {
		return value
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
