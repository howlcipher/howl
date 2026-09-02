package cmd

import (
	"github.com/howlcipher/howl/internal/graph"
	"github.com/spf13/cobra"
)

func newGraphCommand() *cobra.Command {
	var mermaid bool

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Render the ecosystem architectural relationship graph",
		Long: `Render the structural relationships, responsibility boundaries, and integration paths
between the Howl front door, HowlChangeOps authority gate, HowlPlane control plane, and other components.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := graph.DefaultGraph()

			if globalOpts.JSON {
				return graph.RenderJSON(cmd.OutOrStdout(), g)
			}
			if mermaid {
				graph.RenderMermaid(cmd.OutOrStdout(), g)
				return nil
			}

			graph.RenderText(cmd.OutOrStdout())
			return nil
		},
	}

	cmd.Flags().BoolVar(&mermaid, "mermaid", false, "Render the architectural graph in Mermaid format")
	return cmd
}
