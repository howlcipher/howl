package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	var showComponents bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the Howl CLI version and build metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			var info version.Info
			if showComponents {
				m, _, err := manifest.LoadDefault("")
				if err != nil {
					m = &manifest.Manifest{}
				}
				disc := discovery.NewEngine(discovery.DiscoveryOptions{}).DiscoverAll(m)
				info = version.GetFullInfo(disc)
			} else {
				info = version.GetInfo()
			}

			if globalOpts.JSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "howl version %s (%s, %s, %s)\n", info.Version, info.GitCommit, info.BuildDate, info.Platform)
			if showComponents && len(info.Components) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nDiscovered Components:")
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				for comp, ver := range info.Components {
					fmt.Fprintf(tw, "  %s:\t%s\n", comp, ver)
				}
				tw.Flush()
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&showComponents, "components", "c", false, "Inspect and show versions of discovered components")
	return cmd
}
