package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewSyncCmd returns the sync subcommand.
func NewSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Recreate any missing projections tracked by the store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			if err := pctx.Linker.Sync(); err != nil {
				return err
			}
			cmd.Println("sync complete")
			return nil
		},
	}
}
