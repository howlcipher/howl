package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/howlcipher/howl/internal/artifact"
	"github.com/howlcipher/howl/internal/component"
	"github.com/howlcipher/howl/internal/doctor"
	"github.com/howlcipher/howl/internal/health"
	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/pyruntime"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var strict bool
	var fix bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run installer-owned diagnostics and repair",
		Long: `Diagnose installer-owned concerns: platform support, required and optional
dependency availability, managed runtime health, installer state
integrity, installation lock staleness, and per-installed-component
health checks. With --fix, apply deterministic repairs (recreate missing
Howl-owned directories, clear a confirmed-stale lock, clear an
interrupted-operation marker, reinstall a component that fails its
health check).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return fmt.Errorf("doctor execution failed: %w", err)
			}

			cwd, _ := os.Getwd()
			installer := &component.Installer{
				Downloader: artifact.NewHTTPDownloader(),
				Runner:     pyruntime.ExecRunner{},
				BaseDir:    cwd,
			}

			opts := doctor.Options{
				Manifest:  app.Manifest,
				State:     app.State,
				StatePath: app.Paths.StateFile(),
				Paths:     app.Paths,
				Strict:    strict,
				Fix:       fix,
				Detector:  plan.ExecDetector{},
				Health:    health.Checker{Exec: health.OSExecer{}},
				Installer: installer,
			}

			report := doctor.Run(context.Background(), opts)

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
	cmd.Flags().BoolVar(&fix, "fix", false, "Apply deterministic, installer-owned repairs")
	return cmd
}
