package pkgcmd

import (
	"fmt"
	"strings"

	"github.com/asm-cli/asm-cli/internal/agent"
	"github.com/asm-cli/asm-cli/internal/migrator"
	"github.com/spf13/cobra"
)

// NewMigrateCmd returns the migrate subcommand.
// Without --agent it scans all detected agents; with --agent it scans only that one.
func NewMigrateCmd() *cobra.Command {
	var (
		agentFlag  string
		dryRunFlag bool
		allFlag    bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import unmanaged packages from agent directories into ASM",
		Long: `Scans agent skill/mcp directories for packages not tracked by ASM,
deduplicates across agents, then imports each unique package into the store
and links it to every agent that had it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)

			// Determine which agents to scan.
			var agentNames []string
			if allFlag || agentFlag == "" {
				for _, a := range agent.Supported() {
					if _, ok := pctx.Cfg.AgentPaths[string(a)]; ok {
						agentNames = append(agentNames, string(a))
					}
				}
			} else {
				agentNames = []string{resolveAgent(agentFlag, pctx.Cfg)}
			}

			m := migrator.New(pctx.Cfg.ASMHome)

			candidates, err := m.Scan(pctx.Installer.Store(), pctx.Linker, agentNames)
			if err != nil {
				return err
			}

			if len(candidates) == 0 {
				cmd.Println("no unmanaged packages found")
				return nil
			}

			printCandidates(cmd, candidates, dryRunFlag)

			if dryRunFlag {
				return nil
			}

			result, err := m.Apply(candidates, pctx.Installer, pctx.Linker, false)
			if err != nil {
				return err
			}

			cmd.Println()
			printMigrateResult(cmd, result)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "scan only this agent")
	cmd.Flags().BoolVar(&allFlag, "all", false, "scan all configured agents (default when --agent is omitted)")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "show what would be imported without making changes")
	return cmd
}

func printCandidates(cmd *cobra.Command, candidates []migrator.Candidate, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	cmd.Printf("%sfound %d unmanaged package(s):\n", prefix, len(candidates))
	for _, c := range candidates {
		cmd.Printf("  %-30s  agents: %s\n", c.ID, strings.Join(c.FoundIn, ", "))
	}
}

func printMigrateResult(cmd *cobra.Command, result migrator.Result) {
	imported := 0
	skipped := 0
	for _, p := range result.Imported {
		if p.Skipped {
			skipped++
			cmd.Printf("  skipped  %-28s (already in store, linked to: %s)\n",
				p.ID, strings.Join(p.Agents, ", "))
		} else {
			imported++
			cmd.Printf("  imported %-28s → %s\n",
				p.ID, strings.Join(p.Agents, ", "))
		}
	}
	cmd.Printf("\nmigrated %d package(s), %d skipped\n", imported, skipped)
}

// MigrateKind runs migration for a specific kind using pre-built components.
// Used by the root-level `asm migrate` command to avoid duplicating output logic.
func MigrateKind(
	cmd *cobra.Command,
	pctx Context,
	agentNames []string,
	dryRun bool,
) error {
	m := migrator.New(pctx.Cfg.ASMHome)

	candidates, err := m.Scan(pctx.Installer.Store(), pctx.Linker, agentNames)
	if err != nil {
		return err
	}

	kind := pctx.Installer.Store().Kind()
	if len(candidates) == 0 {
		cmd.Printf("[%s] no unmanaged packages found\n", kind)
		return nil
	}

	prefix := fmt.Sprintf("[%s] ", kind)
	if dryRun {
		prefix = "[dry-run][" + string(kind) + "] "
	}
	cmd.Printf("%sfound %d unmanaged package(s):\n", prefix, len(candidates))
	for _, c := range candidates {
		cmd.Printf("  %-30s  agents: %s\n", c.ID, strings.Join(c.FoundIn, ", "))
	}

	if dryRun {
		return nil
	}

	result, err := m.Apply(candidates, pctx.Installer, pctx.Linker, false)
	if err != nil {
		return err
	}

	for _, p := range result.Imported {
		if p.Skipped {
			cmd.Printf("  %sskipped  %-26s (already in store)\n", prefix, p.ID)
		} else {
			cmd.Printf("  %simported %-26s → %s\n", prefix, p.ID, strings.Join(p.Agents, ", "))
		}
	}
	return nil
}
