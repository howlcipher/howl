package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/status"
	"github.com/howlcipher/howl/internal/version"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display installer-level status of ecosystem components",
		Long: `Report what's installed, at what version, against the current release
manifest's targets. This is a fast, state-only report; it does not run
health checks -- use "howl doctor" for that.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadAppContext(globalOpts.ManifestPath)
			if err != nil {
				return fmt.Errorf("failed to collect ecosystem status: %w", err)
			}

			report := status.Collect(app.Manifest, app.State, version.GetInfo().Version)

			if globalOpts.JSON {
				return status.RenderJSON(cmd.OutOrStdout(), report)
			}
			status.RenderHuman(cmd.OutOrStdout(), report)
			return nil
		},
	}
	return cmd
}
