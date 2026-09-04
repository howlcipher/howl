package cmd

import (
	"fmt"
	"os"

	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
	"github.com/spf13/cobra"
)

// removeComponentBinLink removes a component's PATH exposure created by
// ActivateRelease, if any. BinDir (~/.local/bin) isn't an exclusively
// Howl-owned path the way DataHome/CacheHome are, so this only removes a
// real symlink that resolves into Howl's data directory -- never a plain
// file, which could be something the user placed there themselves.
func removeComponentBinLink(paths platform.Paths, name string) {
	binLink := paths.ComponentBinLink(name)
	target, err := os.Readlink(binLink)
	if err != nil {
		return
	}
	if !paths.Owns(target) {
		return
	}
	_ = os.Remove(binLink)
}

func newUninstallCommand() *cobra.Command {
	var purge bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "uninstall [component]",
		Short: "Remove installed ecosystem components",
		Long: `Remove artifacts Howl installed and owns: a component's staged releases
and managed runtime (or, with no component name, every installed
component). Every path removed is verified to be under Howl's own data
directory before removal -- nothing outside it is ever touched, and
Howl's own binary and configuration are left in place. With --purge,
also remove cached downloads and installer state for what was removed;
without it, state history is kept.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			fullSweep := len(args) == 0
			var targets []string
			if !fullSweep {
				targets = args
			} else {
				for name := range app.State.Components {
					targets = append(targets, name)
				}
				hasPurgeableHistory := purge && len(app.State.PreviousComponents) > 0
				if len(targets) == 0 && !hasPurgeableHistory {
					fmt.Fprintln(cmd.OutOrStdout(), "Nothing is installed.")
					return nil
				}
			}

			msg := fmt.Sprintf("Remove %d component(s): %v?", len(targets), targets)
			if fullSweep && len(targets) == 0 {
				msg = "Purge cached downloads and leftover rollback history?"
			}
			if purge && len(targets) > 0 {
				msg += " This also purges cached downloads and their state history."
			}
			if !yes && !confirm(cmd, msg) {
				return exitErr(ExitCancelled, fmt.Errorf("uninstall cancelled"))
			}

			lock, err := state.Acquire(app.Paths.LockFile())
			if err != nil {
				if err == state.ErrLocked {
					return exitErr(ExitLocked, err)
				}
				return exitErr(ExitGeneric, err)
			}
			defer lock.Release()

			for _, name := range targets {
				compDir := app.Paths.ComponentDir(name)
				runtimeDir := app.Paths.RuntimeDir(name)
				for _, p := range []string{compDir, runtimeDir} {
					if !app.Paths.Owns(p) {
						return exitErr(ExitGeneric, fmt.Errorf("refusing to remove %s: not a Howl-owned path", p))
					}
					if err := os.RemoveAll(p); err != nil {
						return exitErr(ExitGeneric, fmt.Errorf("failed to remove %s: %w", p, err))
					}
				}
				removeComponentBinLink(app.Paths, name)
				delete(app.State.Components, name)
				if purge {
					delete(app.State.PreviousComponents, name)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-16s removed\n", name)
			}

			if purge {
				if app.Paths.Owns(app.Paths.DownloadCacheDir()) {
					_ = os.RemoveAll(app.Paths.DownloadCacheDir())
				}
				if fullSweep {
					// A full purge clears all rollback history, not just
					// entries for components that happened to still be
					// installed -- there's nothing left to roll back to.
					app.State.PreviousComponents = nil
				}
				if len(app.State.PreviousComponents) == 0 {
					app.State.PreviousEcosystem = ""
				}
				if len(app.State.Components) == 0 {
					app.State.EcosystemVersion = ""
				}
			}

			if err := app.State.Save(app.Paths.StateFile()); err != nil {
				return exitErr(ExitGeneric, fmt.Errorf("failed to persist state after uninstall: %w", err))
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\nUninstall complete.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&purge, "purge", false, "Also remove cached downloads and state history for removed components")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Proceed without interactive confirmation")
	return cmd
}
