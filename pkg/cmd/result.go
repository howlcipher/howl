package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/engine"
	"github.com/spf13/cobra"
)

// renderResult prints a one-line-per-component outcome summary. Safe to
// call with a nil result (nothing was attempted).
func renderResult(cmd *cobra.Command, result *engine.Result) {
	if result == nil {
		return
	}
	fmt.Fprintln(cmd.OutOrStdout())
	for _, c := range result.Components {
		status := "ok"
		if c.Err != nil {
			status = "FAILED: " + c.Err.Error()
			if c.RolledBack {
				status += " (rolled back)"
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-16s %s\n", c.Name, status)
	}
}
