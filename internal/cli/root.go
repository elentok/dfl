package cli

import "github.com/spf13/cobra"

func (a *App) newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "dfl",
		Short:         "Dotfiles runtime",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&a.dryRun, "dry-run", false, "Show what would happen without modifying the machine")

	cmd.AddCommand(
		a.newSetupCommand(),
		a.newInstallCommand(),
		a.newUpdateCommand(),
		a.newPkgCommand(),
		a.newOSCommand(),
		a.newHasCommandCommand(),
		a.newAskCommand(),
		a.newStepCommand(),
		a.newShellCommand(),
		a.newGitCloneCommand(),
		a.newSymlinkCommand(),
		a.newCopyCommand(),
		a.newMkdirCommand(),
		a.newBackupCommand(),
		a.newRepoRootCommand(),
		a.newVersionCommand(),
	)

	return cmd
}
