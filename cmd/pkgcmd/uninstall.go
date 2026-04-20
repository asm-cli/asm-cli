package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewUninstallCmd returns the uninstall subcommand (removes store record and files, keeps links).
func NewUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <id>",
		Short: "Remove a package from the store (links are not removed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			if err := pctx.Installer.Uninstall(args[0]); err != nil {
				return err
			}
			cmd.Printf("uninstalled %s\n", args[0])
			return nil
		},
	}
}
