package cmd

import (
	"path/filepath"
	"testing"
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
