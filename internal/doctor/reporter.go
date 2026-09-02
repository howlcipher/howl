package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// RenderHuman prints a formatted diagnostic report to the writer.
func RenderHuman(w io.Writer, report *DiagnosticReport, verbose bool) {
	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintln(w, "                          HOWL ECOSYSTEM DOCTOR")
	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintf(w, "Timestamp: %s\n\n", report.Timestamp)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CATEGORY\tCHECK\tSTATUS\tMESSAGE")
	fmt.Fprintln(tw, "--------\t-----\t------\t-------")

	for _, c := range report.Checks {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", c.Category, c.Name, c.Status, c.Message)
		if verbose && c.Details != "" {
			fmt.Fprintf(tw, "\t\t  └─ %s\t\n", c.Details)
		}
	}
	tw.Flush()

	fmt.Fprintln(w, "\n--------------------------------------------------------------------------------")
	fmt.Fprintf(w, "ECOSYSTEM STATUS: %s\n", report.Summary)
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
}

// RenderJSON serializes the diagnostic report as JSON.
func RenderJSON(w io.Writer, report *DiagnosticReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
