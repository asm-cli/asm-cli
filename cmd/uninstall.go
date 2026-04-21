package cmd

import (
	"os"
	"path/filepath"

	"github.com/asm-cli/asm-cli/internal/config"
	"github.com/asm-cli/asm-cli/internal/ejector"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Restore all packages to agent directories and remove the ASM store",
		Long: `Uninstalls ASM in two steps:

  1. Eject: copies every managed skill, plugin, and MCP back into
     the respective agent directory (symlinks become real directories,
     local MCP binaries are restored to <agentHome>/bin/).
  2. Remove: deletes the ~/.asm directory entirely.

Use --dry-run to preview what would be restored before committing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			asmHome := resolveAsmHome()
			cfgPath := filepath.Join(asmHome, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			if dryRun {
				cmd.Println("Dry run — no changes will be made:")
			} else {
				cmd.Println("Step 1/2: restoring packages to agent directories...")
			}

			result, err := ejector.Eject(asmHome, cfg, dryRun)
			if err != nil {
				return err
			}

			for _, item := range result.Items {
				cmd.Printf("  [%s] %s (%s): %s\n", item.Kind, item.PackageID, item.Agent, item.Detail)
			}

			if dryRun {
				cmd.Printf("\n%d item(s) would be restored. Run without --dry-run to apply.\n", len(result.Items))
				return nil
			}

			cmd.Printf("\n%d item(s) restored.\n", len(result.Items))
			cmd.Println("\nStep 2/2: removing ASM store...")
			if err := os.RemoveAll(asmHome); err != nil {
				return err
			}
			cmd.Printf("Removed %s\n", asmHome)
			cmd.Println("\nASM has been uninstalled.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
	return cmd
}

func init() {
	rootCmd.AddCommand(newUninstallCmd())
}
