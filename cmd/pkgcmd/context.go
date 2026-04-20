package pkgcmd

import (
	"context"

	"github.com/6xiaowu9/asm/internal/config"
	"github.com/6xiaowu9/asm/internal/installer"
	"github.com/6xiaowu9/asm/internal/linker"
	"github.com/spf13/cobra"
)

type contextKey struct{}

// Context carries shared dependencies for pkgcmd subcommands.
type Context struct {
	Installer *installer.Installer
	Linker    *linker.Linker
	Cfg       config.Config
}

// Set attaches ctx to cmd's context so subcommands can retrieve it via Must.
func Set(cmd *cobra.Command, ctx Context) {
	cmd.SetContext(context.WithValue(cmd.Context(), contextKey{}, ctx))
}

// Must retrieves Context from cmd. Panics if not set (indicates missing PersistentPreRunE).
func Must(cmd *cobra.Command) Context {
	ctx, ok := cmd.Context().Value(contextKey{}).(Context)
	if !ok {
		panic("pkgcmd: context not set — missing PersistentPreRunE on parent command")
	}
	return ctx
}

// resolveAgent returns agentFlag if non-empty, otherwise the first DefaultAgent,
// otherwise "claude".
func resolveAgent(agentFlag string, cfg config.Config) string {
	if agentFlag != "" {
		return agentFlag
	}
	if len(cfg.DefaultAgents) > 0 {
		return cfg.DefaultAgents[0]
	}
	return "claude"
}
