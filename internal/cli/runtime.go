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

func (a *App) newStepStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "step-start <message...>",
		Short: "Print a step start line",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return (runtimecmd.Runner{Stdout: a.stdoutWriter()}).StepStart(strings.Join(args, " "))
		},
	}
}

func (a *App) newStepEndCommand() *cobra.Command {
	kind := ""
	cmd := &cobra.Command{
		Use:   "step-end [message...]",
		Short: "Print a step end line",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := statusFromKind(kind)
			if err != nil {
				return exitError{code: 2, err: err}
			}
			return (runtimecmd.Runner{Stdout: a.stdoutWriter()}).StepEnd(status, strings.Join(args, " "))
		},
	}
	cmd.Flags().StringVar(&kind, "status", "", "Status: success, skip, or error")
	_ = cmd.RegisterFlagCompletionFunc("status", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"success", "skip", "error"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().Bool("success", false, "Shortcut for --status success")
	cmd.Flags().Bool("skip", false, "Shortcut for --status skip")
	cmd.Flags().Bool("error", false, "Shortcut for --status error")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		shortcuts := map[string]string{
			"success": "success",
			"skip":    "skip",
			"error":   "error",
		}
		for flag, value := range shortcuts {
			on, err := cmd.Flags().GetBool(flag)
			if err != nil {
				return err
			}
			if on {
				if kind != "" && kind != value {
					return exitError{code: 2, err: fmt.Errorf("conflicting step-end status flags")}
				}
				kind = value
			}
		}
		return nil
	}
	return cmd
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
				return err
			}
			if code != 0 {
				return exitError{code: code}
			}
			return nil
		},
	}
}

func (a *App) newSymlinkCommand() *cobra.Command {
	return a.newFilesystemCommand("symlink", "<source> <target>", cobra.ExactArgs(2), func(ctx runtime.Context, args []string) (runtime.ResultStatus, string, error) {
		return (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Symlink(ctx, componentRoot(), args[0], args[1])
	})
}

func (a *App) newCopyCommand() *cobra.Command {
	return a.newFilesystemCommand("copy", "<source> <target>", cobra.ExactArgs(2), func(ctx runtime.Context, args []string) (runtime.ResultStatus, string, error) {
		return (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Copy(ctx, componentRoot(), args[0], args[1])
	})
}

func (a *App) newMkdirCommand() *cobra.Command {
	return a.newFilesystemCommand("mkdir", "<path>", cobra.ExactArgs(1), func(ctx runtime.Context, args []string) (runtime.ResultStatus, string, error) {
		return (runtimecmd.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}).Mkdir(ctx, args[0])
	})
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
				return ui.StepEnd(a.stdoutWriter(), runtime.StatusSkipped, "path does not exist")
			}
			if ctx.DryRun {
				return ui.StepEnd(a.stdoutWriter(), runtime.StatusSuccess, fmt.Sprintf("would move to %s", backupPath))
			}
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
			return ui.StepEnd(a.stdoutWriter(), status, message)
		},
	}
}
