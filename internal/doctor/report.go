package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// RenderHuman prints a formatted diagnostic report to the writer.
func RenderHuman(w io.Writer, report *DiagnosticReport, verbose bool) {
	fmt.Fprintln(w, "Howl Doctor")
	fmt.Fprintf(w, "Timestamp: %s\n\n", report.Timestamp)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CATEGORY\tCHECK\tSTATUS\tMESSAGE")
	for _, c := range report.Checks {
		mark := c.Status
		if c.Fixed {
			mark = mark + " (fixed)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", c.Category, c.Name, mark, c.Message)
		if verbose && c.Details != "" {
			fmt.Fprintf(tw, "\t\t  \\_ %s\t\n", c.Details)
		}
	}
	tw.Flush()

	fmt.Fprintf(w, "\nEcosystem Status: %s\n", report.Summary)
}

// RenderJSON serializes the diagnostic report as JSON.
func RenderJSON(w io.Writer, report *DiagnosticReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
