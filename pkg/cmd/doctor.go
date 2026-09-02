package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run non-destructive ecosystem health diagnostics",
		Long: `Run comprehensive, non-destructive health diagnostics across the Howl ecosystem.
Inspects platform capabilities, toolchain dependencies, manifest syntax, component discovery,
executable ownership, and cross-component integration contracts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := doctor.Options{
				BaseDir:      ".",
				ManifestPath: globalOpts.ManifestPath,
				ConfigPath:   globalOpts.ConfigPath,
				Strict:       strict,
			}

			report, err := doctor.RunDiagnostics(opts)
			if err != nil {
				return fmt.Errorf("doctor execution failed: %w", err)
			}

			if globalOpts.JSON {
				if err := doctor.RenderJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				doctor.RenderHuman(cmd.OutOrStdout(), report, globalOpts.Verbose)
			}

			if report.Summary == doctor.HealthFailed {
				return fmt.Errorf("ecosystem health check failed")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Fail with non-zero exit code on warnings or unverified optional components")
	return cmd
}
