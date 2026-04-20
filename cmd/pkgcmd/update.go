package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewUpdateCmd returns the update subcommand.
func NewUpdateCmd() *cobra.Command {
	var allFlag bool

	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update a package (or all packages with --all)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			if allFlag {
				if err := pctx.Installer.UpdateAll(); err != nil {
					return err
				}
				cmd.Println("updated all packages")
				return nil
			}
			if len(args) == 0 {
				return cmd.Usage()
			}
			if err := pctx.Installer.Update(args[0]); err != nil {
				return err
			}
			cmd.Printf("updated %s\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "update all installed packages")
	return cmd
}
