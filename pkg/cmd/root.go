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
		Short: "Howl — Canonical entry point for the Howl ecosystem",
		Long: `Howl is the unified command-line entry point for the Howl ecosystem.
It provides ecosystem discovery, diagnostics, status inspection, architectural visualization,
and integration routing to child components like HowlPlane, HowlFrame, and HowlChangeOps.`,
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
	rootCmd.AddCommand(newGraphCommand())
	rootCmd.AddCommand(newPlaneCommand())
	rootCmd.AddCommand(newProjectCommand())

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
