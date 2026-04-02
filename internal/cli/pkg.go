package cli

import (
	"fmt"

	"dfl/internal/packagemgr"

	"github.com/spf13/cobra"
)

func (a *App) newPkgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Package manager commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	for _, manager := range []string{"brew", "apt", "npm", "pipx", "cargo", "snap"} {
		cmd.AddCommand(a.newPkgManagerCommand(manager))
	}

	return cmd
}

func (a *App) newPkgManagerCommand(manager string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   manager,
		Short: fmt.Sprintf("%s package operations", manager),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(a.newPkgInstallCommand(manager))
	return cmd
}

func (a *App) newPkgInstallCommand(manager string) *cobra.Command {
	var tap string
	var cask bool

	cmd := &cobra.Command{
		Use:   "install <package...>",
		Short: fmt.Sprintf("Install %s packages", manager),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := a.runtimeContext()
			if err != nil {
				return err
			}

			runner := packagemgr.Runner{Stdout: a.stdoutWriter(), Stderr: a.stderrWriter()}
			code, err := runner.Install(ctx, manager, packagemgr.InstallOptions{
				Packages: args,
				Tap:      tap,
				Cask:     cask,
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

	if manager == "brew" {
		cmd.Flags().StringVar(&tap, "tap", "", "Ensure this Homebrew tap before installing packages")
		cmd.Flags().BoolVar(&cask, "cask", false, "Install brew casks instead of formulae")
	}

	return cmd
}
