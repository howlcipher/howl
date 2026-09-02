// Package version defines release and build metadata for the Howl ecosystem CLI.
package version

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/howlcipher/howl/internal/discovery"
)

var (
	// Version is the semver release string.
	Version = "0.1.0"
	// GitCommit is the commit hash injected at build time.
	GitCommit = "dev"
	// BuildDate is the ISO timestamp injected at build time.
	BuildDate = "unknown"
)

// Info contains build and platform information for the Howl CLI.
type Info struct {
	Version    string            `json:"version"`
	GitCommit  string            `json:"git_commit"`
	BuildDate  string            `json:"build_date"`
	GoVersion  string            `json:"go_version"`
	Platform   string            `json:"platform"`
	Components map[string]string `json:"components,omitempty"`
}

// GetInfo returns the current build information.
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetFullInfo returns build information with component versions when available.
func GetFullInfo(discovered []discovery.DiscoveredComponent) Info {
	info := GetInfo()
	compVersions := make(map[string]string)

	for _, c := range discovered {
		if c.ExecutablePath != "" {
			if ver := probeComponentVersion(c.ExecutablePath); ver != "" {
				compVersions[c.Name] = ver
			} else {
				compVersions[c.Name] = "available (unversioned)"
			}
		} else if c.Found {
			compVersions[c.Name] = "source-only"
		} else {
			compVersions[c.Name] = "not-found"
		}
	}

	info.Components = compVersions
	return info
}

func probeComponentVersion(executablePath string) string {
	cmd := exec.Command(executablePath, "--version")
	out, err := cmd.Output()
	if err == nil {
		line := strings.TrimSpace(string(out))
		if line != "" {
			return strings.Split(line, "\n")[0]
		}
	}
	return ""
}
