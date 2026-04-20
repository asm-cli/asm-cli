package cmd

import (
	"github.com/spf13/cobra"
)

var asmHomeFlag string

var rootCmd = &cobra.Command{
	Use:           "asm",
	Short:         "Agent Skills Manager",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&asmHomeFlag, "asm-home", "", "override ASM home directory")
}
