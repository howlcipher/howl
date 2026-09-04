package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plane"
	"github.com/spf13/cobra"
)

// isHelpArg reports whether an argument is a request for `howl plane`'s own help.
// Only the first argument is ever tested: `howl plane --help` documents this
// command, while `howl plane marathon --help` belongs to HowlPlane and must be
// forwarded like any other argument.
func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func newPlaneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plane",
		Short: "Access HowlPlane AI engineering control plane",
		Long: `Access HowlPlane commands directly from the Howl ecosystem CLI.
Commands are forwarded to the canonical howlplane executable with stream and exit-code fidelity.`,
		// Extended HowlPlane commands own their own flag grammar, which this CLI
		// deliberately does not model. Parsing flags here consumed them: pflag's
		// unknown-flag allowlist strips an unrecognized flag *and its value* from
		// Args(), so `plane marathon --authority-profile howlframe-overnight`
		// reached HowlPlane as bare `marathon`. This is a transparent forwarding
		// boundary, so nothing is parsed and nothing is known -- the remainder of
		// argv is handed over verbatim.
		DisableFlagParsing: true,
	}

	cmd.AddCommand(newProjectCommand())

	// Fallback runner for extended commands not yet part of the static Cobra tree
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}
		// DisableFlagParsing also disables cobra's own help flag handling, so
		// `plane --help` would otherwise be forwarded rather than answered.
		if isHelpArg(args[0]) {
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
