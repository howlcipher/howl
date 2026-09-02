package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plane"
	planeCli "github.com/howlcipher/howlplane/pkg/cli"
	"github.com/spf13/cobra"
)

func newPlaneCommand() *cobra.Command {
	cmd := planeCli.NewPlaneCommand()
	cmd.Short = "Access HowlPlane AI engineering control plane"
	cmd.Long = `Access HowlPlane commands directly from the Howl ecosystem CLI.
Subcommands implemented in Go are executed in-process; other HowlPlane commands are forwarded
to the canonical howlplane executable.`
	cmd.FParseErrWhitelist.UnknownFlags = true
	cmd.DisableFlagParsing = false

	// Fallback runner for extended commands not yet part of the static Cobra tree
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}

		disc := discovery.NewEngine(discovery.DiscoveryOptions{}).DiscoverComponent(manifest.Component{
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
