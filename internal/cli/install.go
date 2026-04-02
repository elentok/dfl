package cli

import (
	"errors"
	"fmt"

	"dfl/internal/install"

	"github.com/spf13/cobra"
)

func (a *App) newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "install <component...>",
		Aliases: []string{"i"},
		Short:   "Install one or more components",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}

			runner := install.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}
			code, err := runner.Install(ctx, args)
			if err != nil && errors.Is(err, install.ErrManifestInstallNotImplemented) {
				fmt.Fprintln(a.stderrWriter(), err)
				return exitError{code: 1}
			}
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
