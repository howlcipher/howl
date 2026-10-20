package interop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/howlcipher/howl/internal/component"
	"github.com/howlcipher/howl/internal/platform"
)

// Locator finds a component's executable in the documented discovery
// order: HOWL<NAME>_BIN, the Howl-managed release, PATH, ~/.local/bin.
type Locator struct {
	Paths    platform.Paths
	Getenv   func(string) string
	LookPath func(string) (string, error)
	Home     string
}

// DefaultLocator uses the process environment and the default Howl paths.
func DefaultLocator() (Locator, error) {
	paths, err := platform.DefaultPaths()
	if err != nil {
		return Locator{}, err
	}
	home, _ := os.UserHomeDir()
	return Locator{Paths: paths, Getenv: os.Getenv, LookPath: exec.LookPath, Home: home}, nil
}

// EnvVar is the override variable for a component, e.g. HOWLCHANGEOPS_BIN.
func EnvVar(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_BIN"
}

// Locate returns the first executable found for name.
func (l Locator) Locate(name string) (string, bool) {
	if l.Getenv != nil {
		if p := strings.TrimSpace(l.Getenv(EnvVar(name))); p != "" {
			return p, true
		}
	}
	if p, ok := component.CurrentBinaryPath(l.Paths, name); ok && isExecutable(p) {
		return p, true
	}
	if l.LookPath != nil {
		if p, err := l.LookPath(platform.ExeName(name)); err == nil {
			return p, true
		}
	}
	if l.Home != "" {
		p := filepath.Join(l.Home, ".local", "bin", platform.ExeName(name))
		if isExecutable(p) {
			return p, true
		}
	}
	return "", false
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
