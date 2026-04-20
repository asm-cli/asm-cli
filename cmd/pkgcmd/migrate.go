package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewMigrateCmd returns the migrate subcommand.
func NewMigrateCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Find packages in the agent directory not yet tracked by ASM",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			a := resolveAgent(agentFlag, pctx.Cfg)
			candidates, err := pctx.Linker.Migrate(a)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				cmd.Println("no unmanaged packages found")
				return nil
			}
			cmd.Printf("unmanaged packages in %s:\n", a)
			for _, c := range candidates {
				cmd.Printf("  %s  %s\n", c.ID, c.SourcePath)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "target agent (default: first DefaultAgent or claude)")
	return cmd
}
