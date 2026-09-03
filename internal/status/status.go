// Package status reports installer-level status: what's installed, at
// what version, against what the current release manifest targets. It
// intentionally does not run health checks (that's `howl doctor`'s job)
// so `howl status` stays fast.
package status

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/state"
)

// ComponentStatus is the installed-vs-target condition of one component.
type ComponentStatus struct {
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installed_version,omitempty"`
	TargetVersion    string `json:"target_version"`
	UpToDate         bool   `json:"up_to_date"`
}

// Report is the full installer-level ecosystem status.
type Report struct {
	EcosystemName          string            `json:"ecosystem_name"`
	InstallerVersion       string            `json:"installer_version"`
	Channel                string            `json:"channel"`
	InstalledEcosystem     string            `json:"installed_ecosystem_version,omitempty"`
	TargetEcosystemVersion string            `json:"target_ecosystem_version"`
	Components             []ComponentStatus `json:"components"`
	UpdatesAvailable       bool              `json:"updates_available"`
}

// Collect builds a status report from a release manifest and the current
// installer state. It never touches the filesystem beyond what the caller
// already loaded.
func Collect(m *manifest.Manifest, st *state.State, installerVersion string) *Report {
	report := &Report{
		EcosystemName:          m.Ecosystem.Name,
		InstallerVersion:       installerVersion,
		Channel:                m.Ecosystem.Channel,
		InstalledEcosystem:     st.EcosystemVersion,
		TargetEcosystemVersion: m.Ecosystem.Version,
	}

	for _, c := range m.Components {
		cs := ComponentStatus{
			Name:          c.Name,
			DisplayName:   displayName(c),
			TargetVersion: c.Version,
		}
		if installed, ok := st.Components[c.Name]; ok {
			cs.Installed = true
			cs.InstalledVersion = installed.Version
			cs.UpToDate = installed.Version == c.Version
		}
		if !cs.UpToDate {
			report.UpdatesAvailable = true
		}
		report.Components = append(report.Components, cs)
	}

	return report
}

func displayName(c manifest.Component) string {
	if c.DisplayName != "" {
		return c.DisplayName
	}
	return c.Name
}

// RenderHuman writes a formatted table of component statuses.
func RenderHuman(w io.Writer, report *Report) {
	fmt.Fprintln(w, "Howl Ecosystem")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Installer\n  Howl            %s\n\n", report.InstallerVersion)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMPONENT\tINSTALLED\tTARGET\tSTATUS")
	for _, c := range report.Components {
		installed := "-"
		if c.Installed {
			installed = c.InstalledVersion
		}
		state := "not installed"
		if c.Installed {
			if c.UpToDate {
				state = "up to date"
			} else {
				state = "update available"
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", c.DisplayName, installed, c.TargetVersion, state)
	}
	tw.Flush()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Ecosystem\n  Version         %s\n", report.TargetEcosystemVersion)
	updates := "current"
	if report.UpdatesAvailable {
		updates = "available"
	}
	fmt.Fprintf(w, "  Updates         %s\n", updates)
}

// RenderJSON serializes the status report as JSON.
func RenderJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
