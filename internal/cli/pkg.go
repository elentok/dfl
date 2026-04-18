package cli

import (
	"fmt"
	"strings"

	"dfl/internal/packagemgr"
	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"

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
	cmd.AddCommand(a.newPkgGitHubCommand())

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

func (a *App) newPkgGitHubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "GitHub release package operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(a.newPkgGitHubInstallCommand())
	return cmd
}

func (a *App) newPkgGitHubInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <owner/repo...>",
		Short: "Install binaries from GitHub releases",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				repo, err := normalizeGitHubRepo(arg)
				if err != nil {
					return err
				}

				installer := packagemgr.GitHubInstaller{
					DryRun:      a.dryRun,
					Repository:  repo,
					VersionArgs: []string{},
				}

				stepLabel := fmt.Sprintf("Installing GitHub package %s", repo)
				err = ui.Step(a.stdoutWriter(), stepLabel, func() (runtimectx.ResultStatus, string, error) {
					result, err := installer.Install("", "")
					if err != nil {
						return "", "", err
					}
					return result.Status, result.Message, nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func normalizeGitHubRepo(arg string) (string, error) {
	repo := strings.TrimSpace(arg)
	repo = strings.Trim(repo, "/")

	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid GitHub package %q; expected owner/repo", arg)
	}
	return parts[0] + "/" + parts[1], nil
}
