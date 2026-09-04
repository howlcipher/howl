package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/howlcipher/howl/internal/artifact"
	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/selfupdate"
	"github.com/howlcipher/howl/internal/state"
	"github.com/howlcipher/howl/internal/version"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var check bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for or apply ecosystem and installer updates",
		Long: `With --check, report available component and installer updates without
changing anything. Without it, resolve the current release manifest,
show an update plan, and (after confirmation, unless --yes) apply it --
staging each new component version, activating it, and verifying it with
a health check. If any component fails, the update is automatically
rolled back and re-verified before Howl reports the outcome. Howl also
checks for and, on confirmation, applies an update to itself, keeping
the previous binary as howl.prev.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			selfRelease := checkSelfUpdate(cmd.Context())

			if check {
				return runUpdateCheck(cmd, app, selfRelease)
			}

			p, err := plan.Build(app.Manifest, app.State, plan.BuildOptions{
				Profile:  installedProfile(app.State),
				Detector: plan.ExecDetector{},
			})
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			if p.HasWork() {
				plan.Render(cmd.OutOrStdout(), p)
			}
			if selfRelease != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\nHowl Installer\n  %s -> %s\n", version.GetInfo().Version, selfRelease.Version)
			}

			if !p.HasWork() && selfRelease == nil {
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

			if p.HasWork() {
				eng := newEngine(app.Paths)
				result, err := eng.Update(cmd.Context(), p, app.Manifest, app.State, app.Paths.StateFile())
				renderResult(cmd, result)
				if err != nil {
					return exitErr(ExitVerificationFailure, err)
				}
			}

			if selfRelease != nil {
				if err := applySelfUpdate(cmd.Context(), app, selfRelease); err != nil {
					// The ecosystem update (if any) already succeeded; a
					// failed self-update doesn't invalidate that, so this
					// is reported, not fatal.
					fmt.Fprintf(cmd.OutOrStdout(), "\nHowl installer self-update failed: %v\n", err)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "\nHowl installer updated to %s (previous binary kept as howl.prev).\n", selfRelease.Version)
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\nUpdate complete.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Report available updates without changing anything")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Proceed without interactive confirmation")
	return cmd
}

// installedProfile reports the profile the ecosystem was last installed
// with, defaulting to "standard" for state files written before Howl
// started persisting it. Update must reuse this rather than assuming
// "standard" outright, or a component installed with --profile developer
// would silently flip back to a downloaded release artifact on the next
// plain `howl update`.
func installedProfile(st *state.State) string {
	if st.Profile == "" {
		return "standard"
	}
	return st.Profile
}

// checkSelfUpdate looks for a newer howl release. Failure (no network, no
// releases published yet, GitHub unreachable) is treated as "no update
// available" rather than an error -- self-update is a nice-to-have on top
// of a working installer, never a hard requirement for other commands.
func checkSelfUpdate(ctx context.Context) *selfupdate.ReleaseInfo {
	api := selfupdate.HTTPGithubAPI{
		Downloader: artifact.NewHTTPDownloader(),
		BaseURL:    os.Getenv("HOWL_SELFUPDATE_API_BASE_URL"),
	}
	rel, err := selfupdate.Check(ctx, api, selfupdate.Repository)
	if err != nil {
		return nil
	}
	current := version.GetInfo().Version
	if rel.Version == current {
		return nil
	}
	return rel
}

func applySelfUpdate(ctx context.Context, app *appContext, rel *selfupdate.ReleaseInfo) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate current howl executable: %w", err)
	}
	return selfupdate.Apply(ctx, artifact.NewHTTPDownloader(), rel, exePath, app.Paths.DownloadCacheDir())
}

func runUpdateCheck(cmd *cobra.Command, app *appContext, selfRelease *selfupdate.ReleaseInfo) error {
	p, err := plan.Build(app.Manifest, app.State, plan.BuildOptions{Profile: installedProfile(app.State)})
	if err != nil {
		return exitErr(ExitValidationFailure, err)
	}

	type componentUpdate struct {
		Name string `json:"name"`
		From string `json:"from,omitempty"`
		To   string `json:"to"`
	}
	type checkReport struct {
		InstallerVersion          string            `json:"installer_version"`
		InstallerAvailableVersion string            `json:"installer_available_version,omitempty"`
		Channel                   string            `json:"channel"`
		InstalledEcosystem        string            `json:"installed_ecosystem_version,omitempty"`
		TargetEcosystemVersion    string            `json:"target_ecosystem_version"`
		Components                []componentUpdate `json:"component_updates"`
	}

	report := checkReport{
		InstallerVersion:       version.GetInfo().Version,
		Channel:                app.Manifest.Ecosystem.Channel,
		InstalledEcosystem:     app.State.EcosystemVersion,
		TargetEcosystemVersion: app.Manifest.Ecosystem.Version,
	}
	if selfRelease != nil {
		report.InstallerAvailableVersion = selfRelease.Version
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

	if len(report.Components) == 0 && selfRelease == nil {
		fmt.Fprintln(cmd.OutOrStdout(), "Everything is up to date.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Updates available")
	fmt.Fprintln(cmd.OutOrStdout())
	if selfRelease != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Howl Installer\n  %s -> %s\n\n", report.InstallerVersion, selfRelease.Version)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Howl Installer\n  %s (current)\n\n", report.InstallerVersion)
	}
	if report.InstalledEcosystem != "" && report.InstalledEcosystem != report.TargetEcosystemVersion {
		fmt.Fprintf(cmd.OutOrStdout(), "Ecosystem\n  %s -> %s\n\n", report.InstalledEcosystem, report.TargetEcosystemVersion)
	}
	if len(report.Components) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Components")
		for _, c := range report.Components {
			from := c.From
			if from == "" {
				from = "(not installed)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %-16s %s -> %s\n", c.Name, from, c.To)
		}
	}
	return nil
}
