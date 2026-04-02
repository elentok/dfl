package cli

import (
	"fmt"

	"dfl/internal/buildinfo"

	"github.com/spf13/cobra"
)

func (a *App) newRepoRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "repo-root",
		Short: "Print the resolved repo root",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}
			fmt.Fprintln(a.stdoutWriter(), ctx.RepoRoot)
			return nil
		},
	}
}

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(a.stdoutWriter(), buildinfo.Version)
			return nil
		},
	}
}
