package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewUnlinkCmd returns the unlink subcommand (removes projection without changing enabled state).
func NewUnlinkCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "unlink <id>",
		Short: "Remove the projection for a package from an agent directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			a := resolveAgent(agentFlag, pctx.Cfg)
			if err := pctx.Linker.Unlink(args[0], a); err != nil {
				return err
			}
			cmd.Printf("unlinked %s from %s\n", args[0], a)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "target agent (default: first DefaultAgent or claude)")
	return cmd
}
