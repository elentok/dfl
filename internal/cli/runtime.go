package cli

import (
	"fmt"
	"strings"

	"dfl/internal/runtime"
	"dfl/internal/runtimecmd"
	"dfl/internal/ui"

	"github.com/spf13/cobra"
)

func (a *App) newHasCommandCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "has-command <name>",
		Short: "Exit successfully if a command exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).HasCommand(args[0])
			if err != nil {
				return err
			}
			if found {
				return nil
			}
			return exitError{code: 1}
		},
	}
}

func (a *App) newAskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question> [default]",
		Short: "Prompt for a value and print the result",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			defaultValue := ""
			if len(args) == 2 {
				defaultValue = args[1]
			}
			value, err := (runtimecmd.Runner{
				Stdin:  a.stdinReader(),
				Stdout: a.stdoutWriter(),
				Stderr: a.stderrWriter(),
			}).Ask(args[0], defaultValue)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.stdoutWriter(), value)
			return err
		},
	}
}

func (a *App) newStepCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "step",
		Short: "Print step output",
	}
	cmd.AddCommand(
		a.newStepStartCommand(),
		a.newStepStatusCommand("success", runtime.StatusSuccess, "Print a success step line"),
		a.newStepStatusCommand("skip", runtime.StatusSkipped, "Print a skipped step line"),
		a.newStepStatusCommand("error", runtime.StatusFailed, "Print an error step line"),
	)
	return cmd
}

func (a *App) newStepStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <message...>",
		Short: "Print a step start line",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := strings.Join(args, " ")
			if err := (runtimecmd.Runner{Stdout: a.stdoutWriter()}).StepStart(message); err != nil {
				return err
			}
			logStepStart(message)
			return nil
		},
	}
}

func (a *App) newStepStatusCommand(name string, status runtime.ResultStatus, short string) *cobra.Command {
	return &cobra.Command{
		Use:   name + " [message...]",
		Short: short,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			message := strings.Join(args, " ")
			if message == "" {
				switch status {
				case runtime.StatusSuccess:
					message = "done"
				case runtime.StatusFailed:
					message = "failed"
				}
			}
			if err := (runtimecmd.Runner{Stdout: a.stdoutWriter()}).StepEnd(status, message); err != nil {
				return err
			}
			logStepEnd(status, message)
			return nil
		},
	}
}

func (a *App) newShellCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "shell <name> -- <command...>",
		Short: "Run a shell command with standard step formatting",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			code, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Shell(ctx, args[0], args[1:])
			if err != nil {
				if code != 0 {
					return exitError{code: code, err: err}
				}
				return err
			}
			if code != 0 {
				return exitError{code: code}
			}
			return nil
		},
	}
}

func (a *App) newGitCloneCommand() *cobra.Command {
	var update bool
	cmd := &cobra.Command{
		Use:   "git-clone <origin> <target>",
		Short: "Clone a git repository, backing up conflicting targets",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			label := fmt.Sprintf("Cloning %s", args[0])
			if err := ui.StepStart(a.stdoutWriter(), label); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(a.stdoutWriter(), "       => %s\n", args[1]); err != nil {
				return err
			}
			status, message, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).GitClone(ctx, args[0], args[1], update)
			if err != nil {
				logStepResult(label, status, message, err)
				if stepErr := ui.StepEnd(a.stdoutWriter(), status, message); stepErr != nil {
					return stepErr
				}
				return exitError{code: 1, err: err}
			}
			logStepResult(label, status, message, nil)
			return ui.StepEnd(a.stdoutWriter(), status, message)
		},
	}
	cmd.Flags().BoolVar(&update, "update", false, "Pull an already cloned repo if the origin matches")
	return cmd
}

func (a *App) newSymlinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "symlink <source> <target>",
		Short: "symlink",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			if err := ui.StepStart(a.stdoutWriter(), fmt.Sprintf("Linking %s", args[0])); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(a.stdoutWriter(), "       => %s\n", args[1]); err != nil {
				return err
			}
			status, message, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Symlink(ctx, componentRoot(), args[0], args[1])
			if err != nil {
				return err
			}
			logStepResult(fmt.Sprintf("Linking %s", args[0]), status, message, nil)
			return ui.StepEnd(a.stdoutWriter(), status, message)
		},
	}
}

func (a *App) newCopyCommand() *cobra.Command {
	return a.newFilesystemCommand("copy", "<source> <target>", cobra.ExactArgs(2), func(ctx runtime.Context, args []string) (runtime.ResultStatus, string, error) {
		return (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Copy(ctx, componentRoot(), args[0], args[1])
	})
}

func (a *App) newInjectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inject <source-file> <target-file>",
		Short: "inject",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			label := fmt.Sprintf("Injecting %s into %s...", args[0], args[1])
			if err := ui.StepStart(a.stdoutWriter(), label); err != nil {
				return err
			}
			status, message, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Inject(ctx, componentRoot(), args[0], args[1])
			if err != nil {
				return err
			}
			logStepResult(label, status, message, nil)
			return ui.StepEnd(a.stdoutWriter(), status, message)
		},
	}
}

func (a *App) newMkdirCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mkdir <path>",
		Short: "mkdir",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			if err := ui.StepStart(a.stdoutWriter(), fmt.Sprintf("Creating %s", args[0])); err != nil {
				return err
			}
			status, message, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Mkdir(ctx, args[0])
			if err != nil {
				return err
			}
			logStepResult(fmt.Sprintf("Creating %s", args[0]), status, message, nil)
			return ui.StepEnd(a.stdoutWriter(), status, message)
		},
	}
}

func (a *App) newBackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup <path>",
		Short: "Move a path to its backup location",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			backupPath, err := (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Backup(ctx, args[0])
			if err != nil {
				return err
			}
			if backupPath == "" {
				logStepResult(fmt.Sprintf("Backing up %s", args[0]), runtime.StatusSkipped, "path does not exist", nil)
				return ui.StepEnd(a.stdoutWriter(), runtime.StatusSkipped, "path does not exist")
			}
			if ctx.DryRun {
				logStepResult(fmt.Sprintf("Backing up %s", args[0]), runtime.StatusSuccess, fmt.Sprintf("would move to %s", backupPath), nil)
				return ui.StepEnd(a.stdoutWriter(), runtime.StatusSuccess, fmt.Sprintf("would move to %s", backupPath))
			}
			logStepResult(fmt.Sprintf("Backing up %s", args[0]), runtime.StatusSuccess, fmt.Sprintf("moved to %s", backupPath), nil)
			return ui.StepEnd(a.stdoutWriter(), runtime.StatusSuccess, fmt.Sprintf("moved to %s", backupPath))
		},
	}
}

func (a *App) newFilesystemCommand(name, argUse string, args cobra.PositionalArgs, run func(runtime.Context, []string) (runtime.ResultStatus, string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   strings.TrimSpace(name + " " + argUse),
		Short: name,
		Args:  args,
		RunE: func(cmd *cobra.Command, argv []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			status, message, err := run(ctx, argv)
			if err != nil {
				return err
			}
			logStepResult(strings.TrimSpace(name+" "+strings.Join(argv, " ")), status, message, nil)
			return ui.StepEnd(a.stdoutWriter(), status, message)
		},
	}
}
