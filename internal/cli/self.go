package cli

import (
	"fmt"

	"dfl/internal/selfmgr"
	"dfl/internal/ui"

	"github.com/spf13/cobra"
)

func (a *App) newSelfCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self",
		Short: "Manage the dfl binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(a.newSelfInstallCommand())
	return cmd
}

func (a *App) newSelfInstallCommand() *cobra.Command {
	var version string
	var target string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install dfl to a user-writable path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			installer := selfmgr.Installer{
				Stdout: a.stdoutWriter(),
				DryRun: a.dryRun,
			}

			label := "Installing dfl"
			if version != "" {
				label = fmt.Sprintf("Installing dfl %s", version)
			}

			return ui.Step(a.stdoutWriter(), label, func() (selfmgr.Status, string, error) {
				result, err := installer.Install(version, target)
				if err != nil {
					return "", "", err
				}
				return result.Status, result.Message, nil
			})
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "Install a specific release tag")
	cmd.Flags().StringVar(&target, "path", "", "Install dfl to this exact path")
	return cmd
}

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
