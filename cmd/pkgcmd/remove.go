package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewRemoveCmd returns the remove subcommand (removes links, store record, and files).
func NewRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a package and all its projections",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			if err := pctx.Installer.Remove(args[0]); err != nil {
				return err
			}
			cmd.Printf("removed %s\n", args[0])
			return nil
		},
	}
}
