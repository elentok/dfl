package cli

import (
	"dfl/internal/runtime"

	"github.com/spf13/cobra"
)

func (a *App) newOSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "os",
		Short: "OS predicate helpers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	for _, item := range []struct {
		use string
		os  runtime.OSType
	}{
		{use: "is-mac", os: runtime.OSMac},
		{use: "is-linux", os: runtime.OSLinux},
		{use: "is-wsl", os: runtime.OSWSL},
	} {
		item := item
		cmd.AddCommand(&cobra.Command{
			Use:   item.use,
			Short: item.use,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx, err := a.runtimeContext()
				if err != nil {
					return err
				}
				if ctx.OS == item.os {
					return nil
				}
				return exitError{code: 1}
			},
		})
	}

	return cmd
}
