// Package cmd defines the Cobra CLI commands for the Howl ecosystem CLI.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// GlobalOptions stores common ecosystem CLI flags.
type GlobalOptions struct {
	JSON         bool
	Verbose      bool
	ConfigPath   string
	ManifestPath string
}

var globalOpts GlobalOptions

// NewRootCommand creates the canonical 'howl' command tree.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "howl",
		Short: "Howl — installer and lifecycle manager for the Howl ecosystem",
		Long: `Howl installs and manages the lifecycle of the Howl ecosystem.
It resolves compatible component versions, installs and updates them safely,
verifies installation health, repairs installer-owned problems, rolls back
failed upgrades, and uninstalls cleanly.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolVar(&globalOpts.JSON, "json", false, "Output results in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&globalOpts.Verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&globalOpts.ConfigPath, "config", "", "Path to Howl user configuration file")
	rootCmd.PersistentFlags().StringVar(&globalOpts.ManifestPath, "manifest", "", "Path to ecosystem.toml manifest")

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newStatusCommand())

	return rootCmd
}

// Execute runs the root command with os arguments and handles exit status.
func Execute() int {
	rootCmd := NewRootCommand()
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}
