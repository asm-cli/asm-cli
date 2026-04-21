package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

var asmHomeFlag string
var showVersion bool

var rootCmd = &cobra.Command{
	Use:           "asm",
	Short:         "Agent Skills Manager",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			versionCmd.Run(cmd, args)
			return nil
		}
		return cmd.Help()
	},
}

// Execute runs the root command with os.Stdout/Stderr.
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteWithWriter runs the root command writing output to w. Used by tests.
func ExecuteWithWriter(w io.Writer, args ...string) error {
	rootCmd.SetOut(w)
	rootCmd.SetErr(w)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	// Reset args so subsequent calls in the same test binary are independent.
	rootCmd.SetArgs(nil)
	return err
}

func init() {
	rootCmd.PersistentFlags().StringVar(&asmHomeFlag, "asm-home", "", "override ASM home directory")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version information")
}
