// Package status provides fast, lightweight inspection of ecosystem components.
package status

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
)

// ComponentStatus represents high-level condition of a single ecosystem component.
type ComponentStatus struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	Available       bool   `json:"available"`
	Runnable        bool   `json:"runnable"`
	RepoPath        string `json:"repo_path,omitempty"`
	ExecutablePath  string `json:"executable_path,omitempty"`
	DiscoverySource string `json:"discovery_source"`
	GitBranch       string `json:"git_branch,omitempty"`
	GitCommit       string `json:"git_commit,omitempty"`
}

// Report holds the ecosystem status summary.
type Report struct {
	EcosystemName string            `json:"ecosystem_name"`
	Components    []ComponentStatus `json:"components"`
}

// Collect inspects the ecosystem quickly without heavy diagnostic execution.
func Collect(baseDir, manifestPath, configPath string) (*Report, error) {
	var m *manifest.Manifest
	var err error
	if manifestPath != "" {
		m, err = manifest.Load(manifestPath)
	} else {
		m, _, err = manifest.LoadDefault(baseDir)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load ecosystem manifest: %w", err)
	}

	discEngine := discovery.NewEngine(discovery.DiscoveryOptions{
		BaseDir:    baseDir,
		ConfigPath: configPath,
	})
	discovered := discEngine.DiscoverAll(m)

	compStatuses := make([]ComponentStatus, 0, len(discovered))
	for _, d := range discovered {
		compStatuses = append(compStatuses, ComponentStatus{
			Name:            d.Name,
			Role:            d.Role,
			Available:       d.Found,
			Runnable:        d.ExecutablePath != "",
			RepoPath:        d.RepoPath,
			ExecutablePath:  d.ExecutablePath,
			DiscoverySource: string(d.DiscoverySource),
			GitBranch:       d.GitBranch,
			GitCommit:       d.GitCommit,
		})
	}

	return &Report{
		EcosystemName: m.Ecosystem.Name,
		Components:    compStatuses,
	}, nil
}

// RenderHuman writes a formatted table of component statuses.
func RenderHuman(w io.Writer, report *Report) {
	fmt.Fprintf(w, "HOWL ECOSYSTEM STATUS (%s)\n\n", report.EcosystemName)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMPONENT\tCONDITION\tSOURCE\tPATH / BINARY")
	fmt.Fprintln(tw, "---------\t---------\t------\t-------------")

	for _, c := range report.Components {
		condition := "MISSING"
		if c.Runnable {
			condition = "RUNNABLE"
		} else if c.Available {
			condition = "SOURCE-ONLY"
		}

		loc := "-"
		if c.ExecutablePath != "" {
			loc = c.ExecutablePath
		} else if c.RepoPath != "" {
			loc = c.RepoPath
			if c.GitBranch != "" {
				loc += fmt.Sprintf(" (%s@%s)", c.GitBranch, c.GitCommit)
			}
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", c.Name, condition, c.DiscoverySource, loc)
	}
	tw.Flush()
}

// RenderJSON serializes the status report as JSON.
func RenderJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
