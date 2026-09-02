package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultEmbedded(t *testing.T) {
	tempDir := t.TempDir()
	m, path, err := LoadDefault(tempDir)
	if err != nil {
		t.Fatalf("unexpected error loading embedded manifest: %v", err)
	}
	if path != "<embedded>" {
		t.Errorf("expected path to be <embedded>, got %s", path)
	}
	if m.Ecosystem.Name != "Howl" {
		t.Errorf("expected ecosystem name 'Howl', got '%s'", m.Ecosystem.Name)
	}
	if len(m.Components) != 6 {
		t.Errorf("expected 6 components, got %d", len(m.Components))
	}
}

func TestLoadValidManifest(t *testing.T) {
	tomlData := `
[ecosystem]
name = "CustomHowl"
version = "0.2.0"
description = "Test description"

[[components]]
name = "comp1"
repository = "https://example.com/comp1"
role = "Testing component 1"
binary = "comp1"
`
	m, err := LoadBytes([]byte(tomlData))
	if err != nil {
		t.Fatalf("failed to load valid bytes: %v", err)
	}
	if m.Ecosystem.Name != "CustomHowl" {
		t.Errorf("got %s, expected CustomHowl", m.Ecosystem.Name)
	}
	c, ok := m.GetComponent("comp1")
	if !ok || c.Binary != "comp1" {
		t.Errorf("failed to get component comp1")
	}
}

func TestInvalidManifests(t *testing.T) {
	tests := []struct {
		name string
		toml string
	}{
		{
			name: "empty ecosystem name",
			toml: `
[ecosystem]
name = ""
[[components]]
name = "c1"
repository = "repo"
role = "role"
`,
		},
		{
			name: "no components",
			toml: `
[ecosystem]
name = "Howl"
`,
		},
		{
			name: "empty component name",
			toml: `
[ecosystem]
name = "Howl"
[[components]]
name = ""
repository = "repo"
role = "role"
`,
		},
		{
			name: "duplicate component name",
			toml: `
[ecosystem]
name = "Howl"
[[components]]
name = "c1"
repository = "repo"
role = "role"
[[components]]
name = "c1"
repository = "repo2"
role = "role2"
`,
		},
		{
			name: "malformed TOML",
			toml: `[ecosystem\nname = invalid`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadBytes([]byte(tc.toml))
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestFindManifestPath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "a", "b", "c")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(`
[ecosystem]
name = "Howl"
[[components]]
name = "c1"
repository = "r"
role = "r"
`), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindManifestPath(subDir)
	if err != nil {
		t.Fatalf("unexpected error finding manifest: %v", err)
	}
	if found != manifestFile {
		t.Errorf("expected %s, got %s", manifestFile, found)
	}
}
