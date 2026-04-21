package cmd

import (
	"github.com/asm-cli/asm-cli/cmd/pkgcmd"
	"github.com/asm-cli/asm-cli/internal/store"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP packages",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return buildPkgContext(cmd, store.PackageKindMCP)
		},
	}

	cmd.AddCommand(
		pkgcmd.NewAddCmd(),
		pkgcmd.NewInstallCmd(),
		pkgcmd.NewListCmd(),
		pkgcmd.NewStatusCmd(),
		pkgcmd.NewMigrateCmd(),
		pkgcmd.NewLinkCmd(),
		pkgcmd.NewUnlinkCmd(),
		pkgcmd.NewEnableCmd(),
		pkgcmd.NewDisableCmd(),
		pkgcmd.NewUseCmd(),
		pkgcmd.NewUninstallCmd(),
		pkgcmd.NewRemoveCmd(),
		pkgcmd.NewUpdateCmd(),
		pkgcmd.NewDoctorCmd(),
		pkgcmd.NewSyncCmd(),
	)

	return cmd
}

func init() {
	rootCmd.AddCommand(newMCPCmd())
}
