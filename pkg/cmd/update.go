package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/state"
	"github.com/howlcipher/howl/internal/version"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var check bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for or apply ecosystem updates",
		Long: `With --check, report available updates without changing anything. Without
it, resolve the current release manifest, show an update plan, and (after
confirmation, unless --yes) apply it -- staging each new version,
activating it, and verifying it with a health check. If any component
fails, the update is automatically rolled back and re-verified before
Howl reports the outcome.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			if check {
				return runUpdateCheck(cmd, app)
			}

			p, err := plan.Build(app.Manifest, app.State, plan.BuildOptions{
				Profile:  "standard",
				Detector: plan.ExecDetector{},
			})
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			plan.Render(cmd.OutOrStdout(), p)

			if !p.HasWork() {
				fmt.Fprintln(cmd.OutOrStdout(), "\nAlready up to date.")
				return nil
			}
			if missing := p.MissingRequired(); len(missing) > 0 {
				return exitErr(ExitMissingDependency, fmt.Errorf("cannot update: required dependencies are missing"))
			}

			if !yes {
				fmt.Fprintln(cmd.OutOrStdout())
				if !confirm(cmd, "Proceed with update?") {
					return exitErr(ExitCancelled, fmt.Errorf("update cancelled"))
				}
			}

			lock, err := state.Acquire(app.Paths.LockFile())
			if err != nil {
				if err == state.ErrLocked {
					return exitErr(ExitLocked, err)
				}
				return exitErr(ExitGeneric, err)
			}
			defer lock.Release()

			eng := newEngine(app.Paths)
			result, err := eng.Update(cmd.Context(), p, app.Manifest, app.State, app.Paths.StateFile())
			renderResult(cmd, result)
			if err != nil {
				return exitErr(ExitVerificationFailure, err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\nUpdate complete.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Report available updates without changing anything")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Proceed without interactive confirmation")
	return cmd
}

func runUpdateCheck(cmd *cobra.Command, app *appContext) error {
	p, err := plan.Build(app.Manifest, app.State, plan.BuildOptions{Profile: "standard"})
	if err != nil {
		return exitErr(ExitValidationFailure, err)
	}

	type componentUpdate struct {
		Name string `json:"name"`
		From string `json:"from,omitempty"`
		To   string `json:"to"`
	}
	type checkReport struct {
		InstallerVersion       string            `json:"installer_version"`
		Channel                string            `json:"channel"`
		InstalledEcosystem     string            `json:"installed_ecosystem_version,omitempty"`
		TargetEcosystemVersion string            `json:"target_ecosystem_version"`
		Components             []componentUpdate `json:"component_updates"`
	}

	report := checkReport{
		InstallerVersion:       version.GetInfo().Version,
		Channel:                app.Manifest.Ecosystem.Channel,
		InstalledEcosystem:     app.State.EcosystemVersion,
		TargetEcosystemVersion: app.Manifest.Ecosystem.Version,
	}
	for _, c := range p.Components {
		if c.Action == plan.ActionSkip {
			continue
		}
		report.Components = append(report.Components, componentUpdate{Name: c.Name, From: c.FromVersion, To: c.ToVersion})
	}

	if globalOpts.JSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	if len(report.Components) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Everything is up to date.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Updates available")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Howl Installer\n  %s (self-update: run `howl update`)\n\n", report.InstallerVersion)
	if report.InstalledEcosystem != "" && report.InstalledEcosystem != report.TargetEcosystemVersion {
		fmt.Fprintf(cmd.OutOrStdout(), "Ecosystem\n  %s -> %s\n\n", report.InstalledEcosystem, report.TargetEcosystemVersion)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Components")
	for _, c := range report.Components {
		from := c.From
		if from == "" {
			from = "(not installed)"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-16s %s -> %s\n", c.Name, from, c.To)
	}
	return nil
}
