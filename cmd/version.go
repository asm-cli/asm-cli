package cmd

import (
	"fmt"

	"github.com/6xiaowu9/asm/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "asm %s (commit: %s, built: %s)\n",
			version.Version, version.Commit, version.BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
