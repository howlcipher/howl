// Package component implements engine.Installer for each of Howl's
// install methods (GitHub release download, Go source build, Python
// source build) behind one narrow interface, and the shared "current
// release" activation logic all three use.
package component

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/howlcipher/howl/internal/platform"
)

// keepReleases is how many staged releases (current + previous) Howl
// retains per component for rollback, per v1's scope.
const keepReleases = 2

// ActivateRelease points a component's "current" pointer at the given
// version's staged release directory, prunes old releases beyond the
// current + previous retained for rollback, and -- unless exposeOnBin is
// false, for components users never invoke directly -- exposes the
// component's executable on the user's PATH via paths.ComponentBinLink.
func ActivateRelease(paths platform.Paths, name, version string, exposeOnBin bool) error {
	link := paths.ComponentCurrentLink(name)
	target := paths.ComponentReleaseDir(name, version)

	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("cannot activate %s@%s: staged release directory missing: %w", name, version, err)
	}

	_ = os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		// Fall back to a plain pointer file for platforms without symlink
		// support (e.g. Windows without the privilege to create one).
		if writeErr := os.WriteFile(link, []byte(version), 0o644); writeErr != nil {
			return fmt.Errorf("failed to activate %s@%s: %w", name, version, writeErr)
		}
	}

	if exposeOnBin {
		binLink := paths.ComponentBinLink(name)
		exePath := filepath.Join(target, platform.ExeName(name))
		_ = os.Remove(binLink)
		if err := os.Symlink(exePath, binLink); err != nil {
			// Fall back to copying the executable for platforms without
			// symlink support -- a plain pointer file wouldn't itself be
			// runnable from PATH the way a symlink or a real copy is.
			if copyErr := copyFile(exePath, binLink); copyErr != nil {
				return fmt.Errorf("failed to expose %s@%s on PATH: %w", name, version, copyErr)
			}
		}
	}

	pruneOldReleases(paths, name, keepReleases)
	return nil
}

// CurrentVersion reports the version a component's "current" pointer
// resolves to, if any.
func CurrentVersion(paths platform.Paths, name string) (string, bool) {
	link := paths.ComponentCurrentLink(name)
	if target, err := os.Readlink(link); err == nil {
		return filepath.Base(target), true
	}
	if data, err := os.ReadFile(link); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v, true
		}
	}
	return "", false
}

// CurrentDir returns the staged release directory a component's "current"
// pointer resolves to.
func CurrentDir(paths platform.Paths, name string) (string, bool) {
	version, ok := CurrentVersion(paths, name)
	if !ok {
		return "", false
	}
	return paths.ComponentReleaseDir(name, version), true
}

// CurrentBinaryPath returns the path to a component's activated
// executable (by the uniform convention that every install method
// produces a <release-dir>/<component-name> executable).
func CurrentBinaryPath(paths platform.Paths, name string) (string, bool) {
	dir, ok := CurrentDir(paths, name)
	if !ok {
		return "", false
	}
	return filepath.Join(dir, platform.ExeName(name)), true
}

func pruneOldReleases(paths platform.Paths, name string, keep int) {
	releasesRoot := filepath.Join(paths.ComponentDir(name), "releases")
	entries, err := os.ReadDir(releasesRoot)
	if err != nil || len(entries) <= keep {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		ii, _ := entries[i].Info()
		jj, _ := entries[j].Info()
		if ii == nil || jj == nil {
			return false
		}
		return ii.ModTime().After(jj.ModTime())
	})

	for _, e := range entries[keep:] {
		_ = os.RemoveAll(filepath.Join(releasesRoot, e.Name()))
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	info, err := os.Stat(src)
	mode := os.FileMode(0o755)
	if err == nil {
		mode = info.Mode()
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}
	return nil
}
