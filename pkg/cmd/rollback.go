package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/state"
	"github.com/spf13/cobra"
)

func newRollbackCommand() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rollback [component]",
		Short: "Roll back to the previous known-good release",
		Long: `Restore the previously recorded known-good component versions (the
versions installed before the last successful "howl update"). With a
component name, only that component is rolled back. Rollback re-runs
each restored component's health check and only reports success for
components that actually pass -- it never claims success without
verification.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return exitErr(ExitValidationFailure, err)
			}

			if !app.State.HasRollbackTarget() {
				return exitErr(ExitRollbackFailure, fmt.Errorf("no previous known-good release recorded: nothing to roll back to"))
			}

			component := ""
			if len(args) > 0 {
				component = args[0]
			}

			if !yes {
				msg := "Roll back the entire ecosystem to the previous known-good release?"
				if component != "" {
					msg = fmt.Sprintf("Roll back %s to its previous known-good version?", component)
				}
				if !confirm(cmd, msg) {
					return exitErr(ExitCancelled, fmt.Errorf("rollback cancelled"))
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
			result, err := eng.Rollback(cmd.Context(), app.Manifest, app.State, app.Paths.StateFile(), component)
			renderResult(cmd, result)
			if err != nil {
				return exitErr(ExitRollbackFailure, err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\nRollback complete and verified.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Proceed without interactive confirmation")
	return cmd
}
