package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewEnableCmd returns the enable subcommand.
func NewEnableCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "enable <id>",
		Short: "Create the projection and mark the package as enabled for an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			a := resolveAgent(agentFlag, pctx.Cfg)
			if err := pctx.Linker.Enable(args[0], a); err != nil {
				return err
			}
			cmd.Printf("enabled %s for %s\n", args[0], a)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "target agent (default: first DefaultAgent or claude)")
	return cmd
}
