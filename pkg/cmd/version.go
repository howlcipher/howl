package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
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
				paths, err := platform.DefaultPaths()
				var components map[string]state.ComponentState
				if err == nil {
					if st, existed, loadErr := state.Load(paths.StateFile()); loadErr == nil && existed {
						components = st.Components
					}
				}
				info = version.GetFullInfo(components)
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
				fmt.Fprintln(cmd.OutOrStdout(), "\nInstalled Components:")
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
