package pkgcmd

import (
	"github.com/spf13/cobra"
)

// NewDoctorCmd returns the doctor subcommand.
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check all tracked projections and report problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pctx := Must(cmd)
			issues, err := pctx.Linker.Doctor()
			if err != nil {
				return err
			}
			if len(issues) == 0 {
				cmd.Println("all projections healthy")
				return nil
			}
			for _, iss := range issues {
				cmd.Printf("[%s/%s] %s\n", iss.Agent, iss.PackageID, iss.Problem)
			}
			return nil
		},
	}
}
