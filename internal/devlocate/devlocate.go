// Package devlocate locates a local source checkout for a component that
// Howl installs via the source_build method. This is only meaningful on a
// developer machine that already has the ecosystem's sibling repositories
// checked out -- it is not how a normal user's install works, and it is
// used nowhere except the source_build install path (see
// docs/ARCHITECTURE.md's "Known v1 Limitations").
package devlocate

import (
	"os"
	"path/filepath"
	"strings"
)

// Options configures where Locate looks for a checkout.
type Options struct {
	// ExplicitDirs maps a checkout name directly to a directory,
	// overriding every other signal (e.g. a --<component>-dir flag).
	ExplicitDirs map[string]string
	// BaseDir is used for the sibling-directory fallback: a checkout
	// named "howlframe" is looked for at filepath.Join(filepath.Dir(BaseDir),
	// "howlframe"). Typically the current working directory.
	BaseDir string
}

// Locate resolves a checkout directory for checkoutName using, in order:
// an explicit override, the HOWL_<NAME>_DIR environment variable, the
// HOWL_DEV_WORKSPACE environment variable joined with checkoutName, and
// finally a directory named checkoutName sibling to opts.BaseDir.
func Locate(checkoutName string, opts Options, getenv func(string) string) (string, bool) {
	if opts.ExplicitDirs != nil {
		if dir, ok := opts.ExplicitDirs[checkoutName]; ok && isDir(dir) {
			return cleanAbs(dir), true
		}
	}

	envKey := "HOWL_" + strings.ToUpper(strings.ReplaceAll(checkoutName, "-", "_")) + "_DIR"
	if dir := getenv(envKey); dir != "" && isDir(dir) {
		return cleanAbs(dir), true
	}

	if workspace := getenv("HOWL_DEV_WORKSPACE"); workspace != "" {
		candidate := filepath.Join(workspace, checkoutName)
		if isDir(candidate) {
			return cleanAbs(candidate), true
		}
	}

	if opts.BaseDir != "" {
		candidate := filepath.Join(filepath.Dir(opts.BaseDir), checkoutName)
		if isDir(candidate) {
			return cleanAbs(candidate), true
		}
	}

	return "", false
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func cleanAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
