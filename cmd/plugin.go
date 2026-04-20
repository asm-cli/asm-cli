package cmd

import (
	"github.com/6xiaowu9/asm/cmd/pkgcmd"
	"github.com/6xiaowu9/asm/internal/store"
	"github.com/spf13/cobra"
)

func newPluginsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage plugin packages",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return buildPkgContext(cmd, store.PackageKindPlugin)
		},
	}

	cmd.AddCommand(
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
	rootCmd.AddCommand(newPluginsCmd())
}
