package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewListCmd returns the list subcommand.
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed packages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			pkgs, err := pctx.Installer.Store().ListPackages()
			if err != nil {
				return err
			}
			if len(pkgs) == 0 {
				cmd.Println("no packages installed")
				return nil
			}
			for _, p := range pkgs {
				cmd.Printf("%-30s  %-5s  %s\n", p.ID, p.Kind, p.Revision)
			}
			return nil
		},
	}
}
