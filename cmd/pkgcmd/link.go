package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewLinkCmd returns the link subcommand (creates projection without marking enabled).
func NewLinkCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "link <id>",
		Short: "Create the projection for a package in an agent directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			a := resolveAgent(agentFlag, pctx.Cfg)
			if err := pctx.Linker.Link(args[0], a); err != nil {
				return err
			}
			cmd.Printf("linked %s for %s\n", args[0], a)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "target agent (default: first DefaultAgent or claude)")
	return cmd
}
