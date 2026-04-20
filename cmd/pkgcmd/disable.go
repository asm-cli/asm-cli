package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewDisableCmd returns the disable subcommand.
func NewDisableCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "disable <id>",
		Short: "Remove the projection and mark the package as disabled for an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			a := resolveAgent(agentFlag, pctx.Cfg)
			if err := pctx.Linker.Disable(args[0], a); err != nil {
				return err
			}
			cmd.Printf("disabled %s for %s\n", args[0], a)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "target agent (default: first DefaultAgent or claude)")
	return cmd
}
