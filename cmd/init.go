package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/asm-cli/asm-cli/internal/agent"
	"github.com/asm-cli/asm-cli/internal/initializer"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		agentsFlag []string
		forceFlag  bool
		dryRunFlag bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise ASM home directory and inject agent configs",
		Long: `Creates the ~/.asm/ directory tree, writes config.toml and ASM.md,
and injects @ASM.md references into each detected agent's config file.

Safe to run multiple times — already-injected references are never duplicated.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}

			var agents []agent.Agent
			for _, s := range agentsFlag {
				for _, tok := range strings.Split(s, ",") {
					tok = strings.TrimSpace(tok)
					if tok == "" {
						continue
					}
					a, err := agent.Parse(tok)
					if err != nil {
						return err
					}
					agents = append(agents, a)
				}
			}

			opts := initializer.Options{
				AsmHome: resolveAsmHome(),
				Agents:  agents,
				Force:   forceFlag,
				DryRun:  dryRunFlag,
				HomeDir: homeDir,
			}

			result, err := initializer.New().Run(opts)
			if err != nil {
				return err
			}

			printInitResult(cmd, result)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&agentsFlag, "agents", nil,
		"agents to initialise (default: auto-detect from home directories)")
	cmd.Flags().BoolVar(&forceFlag, "force", false,
		"overwrite existing per-agent ASM.md files")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		"print what would be done without making any changes")

	return cmd
}

func printInitResult(cmd *cobra.Command, r initializer.Result) {
	prefix := ""
	if r.DryRun {
		prefix = "[dry-run] "
	}

	if r.ConfigWrote {
		cmd.Printf("%swrote %s/config.toml\n", prefix, r.AsmHome)
	} else {
		cmd.Printf("%sconfig.toml already exists, skipped\n", prefix)
	}
	cmd.Printf("%swrote %s/ASM.md\n", prefix, r.AsmHome)

	for _, ar := range r.Agents {
		if ar.Skipped {
			cmd.Printf("%sagent %-8s skipped (home directory not found)\n", prefix, ar.Agent)
			continue
		}
		var actions []string
		if ar.ASMmdWrote {
			actions = append(actions, "wrote ASM.md")
		}
		if ar.Injected {
			actions = append(actions, "injected @ASM.md")
		}
		if len(actions) == 0 {
			actions = []string{"already up-to-date"}
		}
		cmd.Printf("%sagent %-8s %s\n", prefix, ar.Agent, strings.Join(actions, ", "))
	}
}

func init() {
	rootCmd.AddCommand(newInitCmd())
}
