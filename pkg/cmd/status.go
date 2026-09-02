package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/status"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display lightweight status of ecosystem components",
		Long: `Summarize the discovery state, source, and executables of known ecosystem components
without running deep diagnostic suites.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := status.Collect(".", globalOpts.ManifestPath, globalOpts.ConfigPath)
			if err != nil {
				return fmt.Errorf("failed to collect ecosystem status: %w", err)
			}

			if globalOpts.JSON {
				return status.RenderJSON(cmd.OutOrStdout(), report)
			}

			status.RenderHuman(cmd.OutOrStdout(), report)
			return nil
		},
	}
	return cmd
}
