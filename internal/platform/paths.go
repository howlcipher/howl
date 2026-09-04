package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Paths holds every filesystem location Howl owns. Every path Howl writes
// to must come from this struct, so "does howl uninstall --purge only
// touch Howl-owned paths" is provable by construction.
type Paths struct {
	// ConfigHome is where howl's own config.toml lives.
	ConfigHome string
	// DataHome is the root of installed components, runtimes, and state.
	DataHome string
	// CacheHome is where downloads are staged before verification.
	CacheHome string
	// BinDir is where the howl binary and generated wrapper scripts live.
	BinDir string
}

// StateDir returns the directory holding state.json and the install lock.
func (p Paths) StateDir() string { return filepath.Join(p.DataHome, "state") }

// StateFile returns the path to the installer state file.
func (p Paths) StateFile() string { return filepath.Join(p.StateDir(), "state.json") }

// LockFile returns the path to the installation lock file.
func (p Paths) LockFile() string { return filepath.Join(p.StateDir(), "lock") }

// ComponentDir returns the root directory for a single managed component.
func (p Paths) ComponentDir(name string) string {
	return filepath.Join(p.DataHome, "components", name)
}

// ComponentReleaseDir returns the staged, immutable directory for one
// version of a component.
func (p Paths) ComponentReleaseDir(name, version string) string {
	return filepath.Join(p.ComponentDir(name), "releases", version)
}

// ComponentCurrentLink returns the path to the "current" activation pointer
// for a component.
func (p Paths) ComponentCurrentLink(name string) string {
	return filepath.Join(p.ComponentDir(name), "current")
}

// RuntimeDir returns the root directory for a component's managed runtime
// (e.g. an isolated Python virtualenv).
func (p Paths) RuntimeDir(name string) string {
	return filepath.Join(p.DataHome, "runtimes", name)
}

// ComponentBinLink returns the path where a component's executable is
// exposed on the user's PATH (a symlink, or a copy on platforms without
// symlink privilege, into the currently activated release).
func (p Paths) ComponentBinLink(name string) string {
	return filepath.Join(p.BinDir, ExeName(name))
}

// DownloadCacheDir returns the directory downloads are staged into before
// integrity verification.
func (p Paths) DownloadCacheDir() string {
	return filepath.Join(p.CacheHome, "downloads")
}

// ConfigFile returns the path to howl's user configuration file.
func (p Paths) ConfigFile() string {
	return filepath.Join(p.ConfigHome, "config.toml")
}

// ResolvePaths computes Howl's XDG-compliant paths from an environment
// lookup function and a home directory, so tests can supply fixtures
// instead of the real environment.
func ResolvePaths(getenv func(string) string, homeDir string) Paths {
	dataHome := xdgOrDefault(getenv, "XDG_DATA_HOME", filepath.Join(homeDir, ".local", "share"))
	configHome := xdgOrDefault(getenv, "XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	cacheHome := xdgOrDefault(getenv, "XDG_CACHE_HOME", filepath.Join(homeDir, ".cache"))

	return Paths{
		ConfigHome: filepath.Join(configHome, "howl"),
		DataHome:   filepath.Join(dataHome, "howl"),
		CacheHome:  filepath.Join(cacheHome, "howl"),
		BinDir:     filepath.Join(homeDir, ".local", "bin"),
	}
}

// DefaultPaths resolves Howl's paths from the real process environment.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return ResolvePaths(os.Getenv, home), nil
}

// EnsureOwnedDirs creates every directory Howl needs to have present
// (idempotent; safe to call before any lifecycle operation).
func (p Paths) EnsureOwnedDirs() error {
	dirs := []string{
		p.StateDir(),
		filepath.Join(p.DataHome, "components"),
		filepath.Join(p.DataHome, "runtimes"),
		p.DownloadCacheDir(),
		p.ConfigHome,
		p.BinDir,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", d, err)
		}
	}
	return nil
}

// Owns reports whether path lies within a Howl-owned root (DataHome or
// CacheHome). Used to prove uninstall/purge never touches anything outside
// Howl's own filesystem footprint.
func (p Paths) Owns(path string) bool {
	for _, root := range []string{p.DataHome, p.CacheHome} {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return true
		}
	}
	return false
}

func xdgOrDefault(getenv func(string) string, key, def string) string {
	if v := getenv(key); v != "" && filepath.IsAbs(v) {
		return v
	}
	return def
}
