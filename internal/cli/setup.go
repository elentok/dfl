package cli

import (
	"dfl/internal/install"
	"dfl/internal/setup"

	"github.com/spf13/cobra"
)

func (a *App) newSetupCommand() *cobra.Command {
	var components []string
	var skipPackages bool
	var skipRepos bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install the default machine setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return cobra.ExactArgs(0)(cmd, args)
			}

			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}

			runner := setup.Runner{
				Stdout:             a.stdoutWriter(),
				Stderr:             a.stderrWriter(),
				ComponentInstaller: install.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()},
			}

			code, err := runner.Run(ctx, setup.Options{
				Components:   components,
				SkipPackages: skipPackages,
				SkipRepos:    skipRepos,
			})
			if err != nil {
				return err
			}
			if code != 0 {
				return exitError{code: code}
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&components, "component", nil, "Limit setup to selected components")
	cmd.Flags().BoolVar(&skipPackages, "skip-packages", false, "Skip setup package installation")
	cmd.Flags().BoolVar(&skipRepos, "skip-repos", false, "Skip setup repo synchronization")
	return cmd
}
