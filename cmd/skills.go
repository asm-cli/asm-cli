package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/6xiaowu9/asm/cmd/pkgcmd"
	"github.com/6xiaowu9/asm/internal/config"
	"github.com/6xiaowu9/asm/internal/installer"
	"github.com/6xiaowu9/asm/internal/linker"
	"github.com/6xiaowu9/asm/internal/store"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skill packages",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return buildPkgContext(cmd, store.PackageKindSkill)
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

// buildPkgContext loads config, resolves asmHome, and injects a pkgcmd.Context
// into cmd's context. Shared by skills and mcp parent commands.
func buildPkgContext(cmd *cobra.Command, kind store.PackageKind) error {
	cfgPath := filepath.Join(resolveAsmHome(), "config.toml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	asmHome := resolveAsmHome()
	s := store.New(asmHome, kind)
	gitCacheDir := filepath.Join(asmHome, "cache", "git")
	inst := installer.New(s, asmHome, gitCacheDir)
	lnk := linker.New(s, cfg.AgentPaths)

	pkgcmd.Set(cmd, pkgcmd.Context{
		Installer: inst,
		Linker:    lnk,
		Cfg:       cfg,
	})
	return nil
}

// resolveAsmHome returns the effective ASM home directory, preferring the
// --asm-home flag if set, then the ASM_HOME env var, then ~/.asm.
func resolveAsmHome() string {
	if asmHomeFlag != "" {
		return asmHomeFlag
	}
	if env := os.Getenv("ASM_HOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".asm"
	}
	return filepath.Join(home, ".asm")
}

func init() {
	rootCmd.AddCommand(newSkillsCmd())
}
