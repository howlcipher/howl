package cmd

import (
	"path/filepath"
	"testing"

	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
)

// sandboxHowlPaths points every XDG directory Howl reads/writes at a
// throwaway temp directory, so command-level tests never touch the real
// user's home directory.
func sandboxHowlPaths(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "share"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("HOME", root)
}

// mustResolveTestPaths resolves Howl's paths from the sandboxed
// environment set up by sandboxHowlPaths, for tests that need to seed
// state or fixtures directly on disk.
func mustResolveTestPaths(t *testing.T) platform.Paths {
	t.Helper()
	paths, err := platform.DefaultPaths()
	if err != nil {
		t.Fatalf("failed to resolve sandboxed paths: %v", err)
	}
	if err := paths.EnsureOwnedDirs(); err != nil {
		t.Fatal(err)
	}
	return paths
}

// seedInstalledState writes an installer state file recording name as
// already installed at version, for tests exercising idempotency, update,
// rollback, or uninstall against a pre-existing installation.
func seedInstalledState(t *testing.T, paths platform.Paths, name, version string) {
	t.Helper()
	st, _, err := state.Load(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	st.Components[name] = state.ComponentState{Version: version, InstalledAt: "2026-01-01T00:00:00Z"}
	if err := st.Save(paths.StateFile()); err != nil {
		t.Fatal(err)
	}
}

// testManifestTOML is a minimal valid ecosystem release manifest shared by
// command-level tests that just need something loadable, not a specific
// ecosystem shape.
const testManifestTOML = `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"
description = "Test"

[[components]]
name = "howlplane"
repository = "https://github.com/howlcipher/howlplane"
role = "Control plane"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "howlcipher/howlplane"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`
