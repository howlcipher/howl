package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plane"
	"github.com/spf13/cobra"
)

func newPlaneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plane",
		Short: "Access HowlPlane AI engineering control plane",
		Long: `Access HowlPlane commands directly from the Howl ecosystem CLI.
Commands are forwarded to the canonical howlplane executable with stream and exit-code fidelity.`,
	}
	cmd.FParseErrWhitelist.UnknownFlags = true
	cmd.DisableFlagParsing = false

	cmd.AddCommand(newProjectCommand())

	// Fallback runner for extended commands not yet part of the static Cobra tree
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}

		disc := discovery.NewEngine(discovery.DiscoveryOptions{
			ConfigPath: globalOpts.ConfigPath,
		}).DiscoverComponent(manifest.Component{
			Name:   "howlplane",
			Binary: "howlplane",
		})

		if disc.ExecutablePath == "" {
			return fmt.Errorf("howlplane executable not found on system. Install or configure howlplane to use extended CLI commands")
		}

		code, err := plane.DefaultRunner.Run(c.Context(), disc.ExecutablePath, args, c.InOrStdin(), c.OutOrStdout(), c.ErrOrStderr())
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("howlplane exited with code %d", code)
		}
		return nil
	}

	return cmd
}
