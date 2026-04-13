package cli

import (
	"dfl/internal/selfmgr"

	"github.com/spf13/cobra"
)

func (a *App) newUpdateCommand() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update dfl, update the dotfiles repo, and run setup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			updater := selfmgr.Updater{
				Stdout: a.stdoutWriter(),
				Stderr: a.stderrWriter(),
				DryRun: a.dryRun,
			}

			code, err := updater.Run(repo)
			if err != nil {
				return err
			}
			if code != 0 {
				return exitError{code: code}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Update and configure this dotfiles repo")
	return cmd
}
