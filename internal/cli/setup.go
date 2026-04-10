package cli

import (
	"dfl/internal/setup"

	"github.com/spf13/cobra"
)

func (a *App) newSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Run the repo setup script",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return cobra.ExactArgs(0)(cmd, args)
			}

			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}

			runner := setup.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}
			code, err := runner.Run(ctx)
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
