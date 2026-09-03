package plan

import (
	"os/exec"
	"strings"
)

// ExecDetector detects an external dependency by looking it up on PATH and
// probing its version. It's the real Detector implementation used outside
// of tests.
type ExecDetector struct{}

// versionProbes are argument lists tried in order until one produces
// output. Most CLIs support "--version"; a few (notably `go`) only
// recognize a bare "version" subcommand.
var versionProbes = [][]string{{"--version"}, {"version"}, {"-version"}}

// Detect implements Detector.
func (ExecDetector) Detect(name string) (bool, string) {
	path, err := exec.LookPath(name)
	if err != nil {
		return false, ""
	}
	for _, args := range versionProbes {
		out, err := exec.Command(path, args...).Output()
		if err != nil || len(out) == 0 {
			continue
		}
		line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		if line != "" {
			return true, line
		}
	}
	return true, ""
}
