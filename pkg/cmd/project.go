package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plane"
	"github.com/spf13/cobra"
)

func newProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage and inspect project integrations",
	}

	validateCmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate project integration manifest (.ai-project.toml)",
		RunE: func(c *cobra.Command, args []string) error {
			disc := discovery.NewEngine(discovery.DiscoveryOptions{
				ConfigPath: globalOpts.ConfigPath,
			}).DiscoverComponent(manifest.Component{
				Name:   "howlplane",
				Binary: "howlplane",
			})

			if disc.ExecutablePath == "" {
				return fmt.Errorf("howlplane executable not found on system. Install or configure howlplane to use project validation")
			}

			forwardArgs := append([]string{"project", "validate"}, args...)
			code, err := plane.DefaultRunner.Run(c.Context(), disc.ExecutablePath, forwardArgs, c.InOrStdin(), c.OutOrStdout(), c.ErrOrStderr())
			if err != nil {
				return err
			}
			if code != 0 {
				return fmt.Errorf("howlplane exited with code %d", code)
			}
			return nil
		},
	}

	cmd.AddCommand(validateCmd)
	return cmd
}
