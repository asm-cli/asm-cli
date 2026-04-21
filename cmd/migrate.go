package cmd

import (
	"path/filepath"

	"github.com/asm-cli/asm-cli/cmd/pkgcmd"
	"github.com/asm-cli/asm-cli/internal/agent"
	"github.com/asm-cli/asm-cli/internal/config"
	"github.com/asm-cli/asm-cli/internal/installer"
	"github.com/asm-cli/asm-cli/internal/linker"
	"github.com/asm-cli/asm-cli/internal/store"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	var dryRunFlag bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import all unmanaged skills, MCP, and plugins from all agents into ASM",
		Long: `Scans every configured agent's skills, mcp, and plugin directories for
packages not tracked by ASM, deduplicates across agents, then imports each
unique package into the store and links it to every agent that had it.

This command handles skills, MCP, and plugins in one pass.
Use 'asm skills migrate', 'asm mcp migrate', or 'asm plugins migrate' to handle each kind separately.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			asmHome := resolveAsmHome()
			cfgPath := filepath.Join(asmHome, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			// Collect all detected agent names.
			var agentNames []string
			for _, a := range agent.Supported() {
				if _, ok := cfg.AgentPaths[string(a)]; ok {
					agentNames = append(agentNames, string(a))
				}
			}

			gitCacheDir := filepath.Join(asmHome, "cache", "git")

			// Migrate skills.
			skillsStore := store.New(asmHome, store.PackageKindSkill)
			skillsInst := installer.New(skillsStore, asmHome, gitCacheDir)
			skillsLnk := linker.New(skillsStore, cfg.AgentPaths)
			skillsPctx := pkgcmd.Context{
				Installer: skillsInst,
				Linker:    skillsLnk,
				Cfg:       cfg,
			}
			if err := pkgcmd.MigrateKind(cmd, skillsPctx, agentNames, dryRunFlag); err != nil {
				return err
			}

			cmd.Println()

			// Migrate MCP.
			mcpStore := store.New(asmHome, store.PackageKindMCP)
			mcpInst := installer.New(mcpStore, asmHome, gitCacheDir)
			mcpLnk := linker.New(mcpStore, cfg.AgentPaths)
			mcpPctx := pkgcmd.Context{
				Installer: mcpInst,
				Linker:    mcpLnk,
				Cfg:       cfg,
			}
			if err := pkgcmd.MigrateKind(cmd, mcpPctx, agentNames, dryRunFlag); err != nil {
				return err
			}

			cmd.Println()

			// Migrate plugins.
			pluginStore := store.New(asmHome, store.PackageKindPlugin)
			pluginInst := installer.New(pluginStore, asmHome, gitCacheDir)
			pluginLnk := linker.New(pluginStore, cfg.AgentPaths)
			pluginPctx := pkgcmd.Context{
				Installer: pluginInst,
				Linker:    pluginLnk,
				Cfg:       cfg,
			}
			return pkgcmd.MigrateKind(cmd, pluginPctx, agentNames, dryRunFlag)
		},
	}

	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "show what would be imported without making changes")
	return cmd
}

func init() {
	rootCmd.AddCommand(newMigrateCmd())
}
