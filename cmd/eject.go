package cmd

import (
	"path/filepath"

	"github.com/asm-cli/asm-cli/internal/config"
	"github.com/asm-cli/asm-cli/internal/ejector"
	"github.com/spf13/cobra"
)

func newEjectCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "eject",
		Short: "Restore all managed packages back to agent directories",
		Long: `Copies every ASM-managed skill, plugin, and MCP back into the
respective agent directory, reversing the effect of migration.

After ejection:
  - Skills/plugins: symlinks are replaced with real directory copies
  - MCPs with local binaries: binary is restored to <agentHome>/bin/
    and the agent config command path is updated accordingly
  - MCPs using npx/http/sse: no file changes needed

The ASM store is NOT removed automatically. Once you have verified the
result, you can safely delete ~/.asm.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			asmHome := resolveAsmHome()
			cfgPath := filepath.Join(asmHome, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			if !dryRun {
				cmd.Println("Restoring packages to agent directories...")
			} else {
				cmd.Println("Dry run — no changes will be made:")
			}

			result, err := ejector.Eject(asmHome, cfg, dryRun)
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				cmd.Println("Nothing to eject.")
				return nil
			}

			for _, item := range result.Items {
				cmd.Printf("  [%s] %s (%s): %s\n", item.Kind, item.PackageID, item.Agent, item.Detail)
			}

			if !dryRun {
				cmd.Printf("\n%d item(s) restored.\n", len(result.Items))
				cmd.Println("You can now safely remove ~/.asm if desired.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be restored without making changes")
	return cmd
}

func init() {
	rootCmd.AddCommand(newEjectCmd())
}
