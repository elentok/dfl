package cli

import (
	"fmt"
	"io"
	"strings"

	"dfl/internal/runtime"
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
		stdout = io.Discard
	}

	stderr := a.stderr
	if stderr == nil {
		stderr = io.Discard
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

	switch cmd {
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0, nil
	case "setup":
		return runPlaceholder(stdout, "setup", rest)
	case "install", "i":
		return runPlaceholder(stdout, "install", rest)
	case "pkg":
		return runPlaceholder(stdout, "pkg", rest)
	case "os":
		return runPlaceholder(stdout, "os", rest)
	case "has-command":
		return runPlaceholder(stdout, "has-command", rest)
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
		"  repo-root",
		"  version",
	}

	fmt.Fprintln(w, strings.Join(lines, "\n"))
}
