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
		a.newPkgCommand(),
		a.newOSCommand(),
		a.newHasCommandCommand(),
		a.newStepStartCommand(),
		a.newStepEndCommand(),
		a.newShellCommand(),
		a.newSymlinkCommand(),
		a.newCopyCommand(),
		a.newMkdirCommand(),
		a.newBackupCommand(),
		a.newRepoRootCommand(),
		a.newVersionCommand(),
	)

	return cmd
}

func (a *App) newSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Install the default machine setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runPlaceholder("setup", args)
		},
	}
}
