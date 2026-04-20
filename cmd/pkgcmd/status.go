package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewStatusCmd returns the status subcommand.
func NewStatusCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show projection status for an agent",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			a := resolveAgent(agentFlag, pctx.Cfg)
			report, err := pctx.Linker.Status(a)
			if err != nil {
				return err
			}
			cmd.Printf("agent: %s  kind: %s\n", report.Agent, report.Kind)
			cmd.Printf("enabled (%d):\n", len(report.EnabledPackages))
			for _, p := range report.EnabledPackages {
				cmd.Printf("  %s\n", p.ID)
			}
			cmd.Printf("disabled (%d):\n", len(report.DisabledPackages))
			for _, p := range report.DisabledPackages {
				cmd.Printf("  %s\n", p.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "target agent (default: first DefaultAgent or claude)")
	return cmd
}
