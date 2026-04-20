package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewUseCmd returns the use subcommand (enable for one or more agents at once).
func NewUseCmd() *cobra.Command {
	var agentsFlag []string

	cmd := &cobra.Command{
		Use:   "use <id>",
		Short: "Enable a package for one or more agents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			agents := agentsFlag
			if len(agents) == 0 {
				agents = []string{resolveAgent("", pctx.Cfg)}
			}
			if err := pctx.Linker.Use(args[0], agents); err != nil {
				return err
			}
			cmd.Printf("enabled %s for %v\n", args[0], agents)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&agentsFlag, "agents", nil, "agents to enable (default: first DefaultAgent or claude)")
	return cmd
}
