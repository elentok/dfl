package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"dfl/internal/install"
	"dfl/internal/runtime"
	"dfl/internal/runtimecmd"
)

type App struct {
	stdout io.Writer
	stderr io.Writer
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
	stdout := a.stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stderr := a.stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	ctx, err := runtime.NewContext("")
	if err != nil {
		return 1, err
	}

	if len(args) == 0 {
		printUsage(stdout)
		return 0, nil
	}

	cmd := args[0]
	rest := args[1:]
	dryRun, rest := parseDryRun(rest)
	ctx.DryRun = dryRun

	switch cmd {
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0, nil
	case "setup":
		return runPlaceholder(stdout, "setup", rest)
	case "install", "i":
		runner := install.Runner{Stdout: stdout, Stderr: stderr}
		code, err := runner.Install(ctx, rest)
		if err != nil && errors.Is(err, install.ErrManifestInstallNotImplemented) {
			fmt.Fprintln(stderr, err)
			return 1, nil
		}
		return code, err
	case "pkg":
		return runPlaceholder(stdout, "pkg", rest)
	case "os":
		return runOS(ctx, rest)
	case "has-command":
		return runHasCommand(stdout, stderr, rest)
	case "step-start":
		return runStepStart(stdout, rest)
	case "step-end":
		return runStepEnd(stdout, rest)
	case "shell":
		return runShell(ctx, stdout, stderr, rest)
	case "symlink":
		return runSymlink(ctx, stdout, stderr, rest)
	case "copy":
		return runCopy(ctx, stdout, stderr, rest)
	case "mkdir":
		return runMkdir(ctx, stdout, stderr, rest)
	case "backup":
		return runBackup(ctx, stdout, stderr, rest)
	case "version":
		fmt.Fprintln(stdout, "dfl dev")
		return 0, nil
	case "repo-root":
		fmt.Fprintln(stdout, ctx.RepoRoot)
		return 0, nil
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", cmd)
		printUsage(stderr)
		return 2, nil
	}
}

func runPlaceholder(w io.Writer, name string, args []string) (int, error) {
	if len(args) > 0 && isHelpArg(args[0]) {
		fmt.Fprintf(w, "%s command is not implemented yet\n", name)
		return 0, nil
	}

	fmt.Fprintf(w, "%s command is not implemented yet\n", name)
	return 0, nil
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func printUsage(w io.Writer) {
	lines := []string{
		"Usage:",
		"  dfl <command>",
		"",
		"Commands:",
		"  setup",
		"  install, i",
		"  pkg",
		"  os",
		"  has-command",
		"  step-start",
		"  step-end",
		"  shell",
		"  symlink",
		"  copy",
		"  mkdir",
		"  backup",
		"  repo-root",
		"  version",
	}

	fmt.Fprintln(w, strings.Join(lines, "\n"))
}

func parseDryRun(args []string) (bool, []string) {
	if len(args) > 0 && args[0] == "--dry-run" {
		return true, args[1:]
	}
	return false, args
}

func runOS(ctx runtime.Context, args []string) (int, error) {
	if len(args) != 1 {
		return 2, errors.New("os requires exactly one predicate")
	}

	switch args[0] {
	case "is-mac":
		return boolExitCode(ctx.OS == runtime.OSMac), nil
	case "is-linux":
		return boolExitCode(ctx.OS == runtime.OSLinux), nil
	case "is-wsl":
		return boolExitCode(ctx.OS == runtime.OSWSL), nil
	default:
		return 2, fmt.Errorf("unknown os predicate %q", args[0])
	}
}

func runHasCommand(stdout, stderr io.Writer, args []string) (int, error) {
	if len(args) != 1 {
		return 2, errors.New("has-command requires exactly one command name")
	}

	found, err := (runtimecmd.Runner{Stdout: stdout, Stderr: stderr}).HasCommand(args[0])
	if err != nil {
		return 1, err
	}

	return boolExitCode(found), nil
}

func runStepStart(stdout io.Writer, args []string) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("step-start requires a message")
	}

	return 0, (runtimecmd.Runner{Stdout: stdout}).StepStart(strings.Join(args, " "))
}

func runStepEnd(stdout io.Writer, args []string) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("step-end requires a status flag")
	}

	var status runtime.ResultStatus
	switch args[0] {
	case "--success":
		status = runtime.StatusSuccess
	case "--skip":
		status = runtime.StatusSkipped
	case "--error":
		status = runtime.StatusFailed
	default:
		return 2, fmt.Errorf("unknown step-end flag %q", args[0])
	}

	return 0, (runtimecmd.Runner{Stdout: stdout}).StepEnd(status, strings.Join(args[1:], " "))
}

func runShell(ctx runtime.Context, stdout, stderr io.Writer, args []string) (int, error) {
	if len(args) < 3 {
		return 2, errors.New("shell requires the form: shell <name> -- <command...>")
	}

	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator != 1 || separator == len(args)-1 {
		return 2, errors.New("shell requires the form: shell <name> -- <command...>")
	}

	return (runtimecmd.Runner{Stdout: stdout, Stderr: stderr}).Shell(ctx, args[0], args[separator+1:])
}

func runSymlink(ctx runtime.Context, stdout, stderr io.Writer, args []string) (int, error) {
	if len(args) != 2 {
		return 2, errors.New("symlink requires <source> <target>")
	}

	status, message, err := (runtimecmd.Runner{Stdout: stdout, Stderr: stderr}).Symlink(ctx, componentRoot(), args[0], args[1])
	return printOperationResult(stdout, status, message, err)
}

func runCopy(ctx runtime.Context, stdout, stderr io.Writer, args []string) (int, error) {
	if len(args) != 2 {
		return 2, errors.New("copy requires <source> <target>")
	}

	status, message, err := (runtimecmd.Runner{Stdout: stdout, Stderr: stderr}).Copy(ctx, componentRoot(), args[0], args[1])
	return printOperationResult(stdout, status, message, err)
}

func runMkdir(ctx runtime.Context, stdout, stderr io.Writer, args []string) (int, error) {
	if len(args) != 1 {
		return 2, errors.New("mkdir requires <path>")
	}

	status, message, err := (runtimecmd.Runner{Stdout: stdout, Stderr: stderr}).Mkdir(ctx, args[0])
	return printOperationResult(stdout, status, message, err)
}

func runBackup(ctx runtime.Context, stdout, stderr io.Writer, args []string) (int, error) {
	if len(args) != 1 {
		return 2, errors.New("backup requires <path>")
	}

	backupPath, err := (runtimecmd.Runner{Stdout: stdout, Stderr: stderr}).Backup(ctx, args[0])
	if err != nil {
		return 1, err
	}
	if backupPath == "" {
		fmt.Fprintln(stdout, "[skipped] path does not exist")
		return 0, nil
	}
	if ctx.DryRun {
		fmt.Fprintf(stdout, "[success] would move to %s\n", backupPath)
		return 0, nil
	}

	fmt.Fprintf(stdout, "[success] moved to %s\n", backupPath)
	return 0, nil
}

func printOperationResult(stdout io.Writer, status runtime.ResultStatus, message string, err error) (int, error) {
	if err != nil {
		return 1, err
	}

	fmt.Fprintf(stdout, "[%s] %s\n", status, message)
	return 0, nil
}

func boolExitCode(value bool) int {
	if value {
		return 0
	}
	return 1
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
