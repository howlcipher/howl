package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/state"
	"github.com/spf13/cobra"
)

var validProfiles = map[string]bool{"standard": true, "local-ai": true, "developer": true}

func newInstallCommand() *cobra.Command {
	var profile string
	var yes bool

	cmd := &cobra.Command{
		Use:   "install [component]",
		Short: "Install the Howl ecosystem",
		Long: `Install the complete Howl ecosystem (or, given a component name, just
that component and its dependencies) at the versions declared by the
current release manifest. Shows a plan and asks for confirmation before
touching your machine unless --yes is passed. Safe to run again: anything
already installed at the target version is left alone.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validProfiles[profile] {
				return exitErr(ExitValidationFailure, fmt.Errorf("unknown profile %q (expected standard, local-ai, or developer)", profile))
			}

			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			p, err := plan.Build(app.Manifest, app.State, plan.BuildOptions{
				Profile:    profile,
				Components: args,
				Detector:   plan.ExecDetector{},
			})
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			plan.Render(cmd.OutOrStdout(), p)

			if !p.HasWork() {
				fmt.Fprintln(cmd.OutOrStdout(), "\nAlready installed at the target version.")
				return nil
			}

			if missing := p.MissingRequired(); len(missing) > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
				for _, m := range missing {
					fmt.Fprintf(cmd.OutOrStdout(), "Missing required dependency: %s (needed for %s)\n", m.Name, m.Capability)
				}
				return exitErr(ExitMissingDependency, fmt.Errorf("cannot install: required dependencies are missing"))
			}

			if !yes {
				fmt.Fprintln(cmd.OutOrStdout())
				if !confirm(cmd, "Proceed?") {
					return exitErr(ExitCancelled, fmt.Errorf("installation cancelled"))
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
			result, err := eng.Install(cmd.Context(), p, app.State, app.Paths.StateFile())
			if err != nil {
				renderResult(cmd, result)
				return exitErr(ExitInstallationFailure, err)
			}

			renderResult(cmd, result)
			fmt.Fprintln(cmd.OutOrStdout(), "\nInstall complete.")
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "standard", "Installation profile: standard, local-ai, or developer")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Proceed without interactive confirmation")
	return cmd
}
