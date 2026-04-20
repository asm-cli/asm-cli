package pkgcmd

import (
	"fmt"

	"github.com/6xiaowu9/asm/internal/installer"
	"github.com/spf13/cobra"
)

// NewInstallCmd returns the install subcommand.
func NewInstallCmd() *cobra.Command {
	var (
		idFlag     string
		subdirFlag string
		refFlag    string
		agentsFlag []string
	)

	cmd := &cobra.Command{
		Use:   "install <source>",
		Short: "Install a package from a local path or git URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			rec, err := pctx.Installer.Install(args[0], installer.Options{
				ID:     idFlag,
				Subdir: subdirFlag,
				Ref:    refFlag,
			})
			if err != nil {
				return err
			}
			cmd.Printf("installed %s (%s)\n", rec.ID, rec.Revision)

			for _, a := range agentsFlag {
				if err := pctx.Linker.Enable(rec.ID, a); err != nil {
					return fmt.Errorf("enable for %s: %w", a, err)
				}
				cmd.Printf("enabled for %s\n", a)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&idFlag, "id", "", "override package ID")
	cmd.Flags().StringVar(&subdirFlag, "subdir", "", "subdirectory within source")
	cmd.Flags().StringVar(&refFlag, "ref", "", "git ref (branch/tag/commit)")
	cmd.Flags().StringSliceVar(&agentsFlag, "agents", nil, "agents to enable after install")

	return cmd
}
