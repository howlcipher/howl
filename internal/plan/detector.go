package plan

import (
	"os/exec"
	"strings"
)

// ExecDetector detects an external dependency by looking it up on PATH and
// running it with --version. It's the real Detector implementation used
// outside of tests.
type ExecDetector struct{}

// Detect implements Detector.
func (ExecDetector) Detect(name string) (bool, string) {
	path, err := exec.LookPath(name)
	if err != nil {
		return false, ""
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil || len(out) == 0 {
		return true, ""
	}
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return true, line
}
