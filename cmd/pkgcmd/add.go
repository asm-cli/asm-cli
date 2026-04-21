package pkgcmd

import (
	"fmt"
	"strings"

	"github.com/asm-cli/asm-cli/internal/agent"
	"github.com/spf13/cobra"
)

// NewAddCmd returns the add subcommand for registering MCP servers inline.
// Unlike install, add does not require a source directory — it stores the
// config directly (suitable for npx, http/sse, and other non-local servers).
func NewAddCmd() *cobra.Command {
	var (
		typeFlag    string
		commandFlag string
		argsFlag    []string
		urlFlag     string
		envFlag     []string
		agentsFlag  []string
	)

	cmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Register an MCP server from inline config (npx / http / sse)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			id := args[0]

			cfg := agent.MCPServerConfig{
				Type:    typeFlag,
				Command: commandFlag,
				Args:    argsFlag,
				URL:     urlFlag,
			}
			if len(envFlag) > 0 {
				cfg.Env = make(map[string]string, len(envFlag))
				for _, kv := range envFlag {
					k, v, ok := strings.Cut(kv, "=")
					if !ok {
						return fmt.Errorf("invalid --env value %q: expected KEY=VALUE", kv)
					}
					cfg.Env[k] = v
				}
			}

			rec, err := pctx.Installer.InstallMCPServer(id, cfg)
			if err != nil {
				return err
			}
			cmd.Printf("added %s\n", rec.ID)

			for _, a := range agentsFlag {
				if err := pctx.Linker.Enable(rec.ID, a); err != nil {
					return fmt.Errorf("enable for %s: %w", a, err)
				}
				cmd.Printf("enabled for %s\n", a)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&typeFlag, "type", "stdio", "transport type: stdio | sse | http")
	cmd.Flags().StringVar(&commandFlag, "command", "", "command to run (stdio)")
	cmd.Flags().StringArrayVar(&argsFlag, "args", nil, "arguments (repeatable: --args -y --args pkg)")
	cmd.Flags().StringVar(&urlFlag, "url", "", "server URL (sse/http)")
	cmd.Flags().StringArrayVar(&envFlag, "env", nil, "env vars KEY=VALUE (repeatable)")
	cmd.Flags().StringSliceVar(&agentsFlag, "agents", nil, "agents to enable after add")

	return cmd
}
