// Package version defines release and build metadata for the Howl ecosystem CLI.
package version

import (
	"fmt"
	"runtime"

	"github.com/howlcipher/howl/internal/state"
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

// GetFullInfo returns build information with installed component versions
// from Howl's own installer state (not live probing -- that's what
// `howl status`/`howl doctor` are for).
func GetFullInfo(components map[string]state.ComponentState) Info {
	info := GetInfo()
	compVersions := make(map[string]string, len(components))
	for name, c := range components {
		compVersions[name] = c.Version
	}
	info.Components = compVersions
	return info
}
