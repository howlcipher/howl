package cmd

import (
	"fmt"
	"os"

	"github.com/howlcipher/howl/internal/platform"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

var validChannels = map[string]bool{"stable": true, "beta": true, "dev": true}

type userConfig struct {
	Channel string `toml:"channel,omitempty"`
}

func newChannelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [stable|beta|dev]",
		Short: "Show or set the release channel",
		Long: `With no argument, show the currently configured release channel. With
stable, beta, or dev, set it. Channels only select which official Howl
ecosystem release manifest is used -- this is not a general package
source configuration mechanism.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := platform.DefaultPaths()
			if err != nil {
				return exitErr(ExitGeneric, err)
			}
			if err := paths.EnsureOwnedDirs(); err != nil {
				return exitErr(ExitGeneric, err)
			}

			cfg, err := loadUserConfig(paths.ConfigFile())
			if err != nil {
				return exitErr(ExitGeneric, err)
			}

			if len(args) == 0 {
				channel := cfg.Channel
				if channel == "" {
					channel = "stable"
				}
				fmt.Fprintln(cmd.OutOrStdout(), channel)
				return nil
			}

			requested := args[0]
			if !validChannels[requested] {
				return exitErr(ExitValidationFailure, fmt.Errorf("unknown channel %q (expected stable, beta, or dev)", requested))
			}

			cfg.Channel = requested
			if err := saveUserConfig(paths.ConfigFile(), cfg); err != nil {
				return exitErr(ExitGeneric, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Channel set to %s.\n", requested)
			return nil
		},
	}
	return cmd
}

func loadUserConfig(path string) (userConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return userConfig{}, nil
	}
	if err != nil {
		return userConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg userConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return userConfig{}, fmt.Errorf("config file at %s is corrupt: %w", path, err)
	}
	return cfg, nil
}

func saveUserConfig(path string, cfg userConfig) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}
	return nil
}
