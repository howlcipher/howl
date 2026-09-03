package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func validComponentTOML(name string, deps string) string {
	return `
[[components]]
name = "` + name + `"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
` + deps + `
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`
}

func baseManifest(components string) string {
	return `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"
` + components
}

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
	if len(m.Components) != 4 {
		t.Errorf("expected 4 components, got %d", len(m.Components))
	}

	order, err := m.TopoOrder()
	if err != nil {
		t.Fatalf("expected embedded manifest to have a valid dependency order: %v", err)
	}
	want := []string{"howlframe", "howlchangeops", "howlplane", "howlwriter"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("expected install order %v, got %v", want, order)
			break
		}
	}
}

func TestLoadValidManifest(t *testing.T) {
	tomlData := baseManifest(validComponentTOML("comp1", ""))
	m, err := LoadBytes([]byte(tomlData))
	if err != nil {
		t.Fatalf("failed to load valid bytes: %v", err)
	}
	c, ok := m.GetComponent("comp1")
	if !ok || c.Name != "comp1" {
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
schema_version = 1
[ecosystem]
name = ""
version = "0.1.0"
` + validComponentTOML("c1", ""),
		},
		{
			name: "no components",
			toml: `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
`,
		},
		{
			name: "empty component name",
			toml: baseManifest(`
[[components]]
name = ""
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "duplicate component name",
			toml: baseManifest(validComponentTOML("c1", "") + validComponentTOML("c1", "")),
		},
		{
			name: "malformed TOML",
			toml: `[ecosystem\nname = invalid`,
		},
		{
			name: "unknown schema version",
			toml: `
schema_version = 99
[ecosystem]
name = "Howl"
version = "0.1.0"
` + validComponentTOML("c1", ""),
		},
		{
			name: "unknown component in depends_on",
			toml: baseManifest(validComponentTOML("c1", `
  [[components.depends_on]]
  component = "does-not-exist"
`)),
		},
		{
			name: "self-referential dependency",
			toml: baseManifest(validComponentTOML("c1", `
  [[components.depends_on]]
  component = "c1"
`)),
		},
		{
			name: "dependency cycle",
			toml: baseManifest(
				validComponentTOML("a", `
  [[components.depends_on]]
  component = "b"
`) + validComponentTOML("b", `
  [[components.depends_on]]
  component = "a"
`)),
		},
		{
			name: "unsupported platform",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["amiga"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "unsupported architecture",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["mips"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "invalid component version",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "not-a-version"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "compatibility failure: min_version exceeds declared dependency version",
			toml: baseManifest(
				validComponentTOML("base", "") +
					`
[[components]]
name = "needs-newer-base"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
  [[components.depends_on]]
  component = "base"
  min_version = "99.0.0"
`),
		},
		{
			name: "github_release missing required fields",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "source_build with unsupported language",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "source_build"
    [components.install.source_build]
    checkout_name = "c1"
    language = "rust"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "python_import health check missing module",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "python_import"
`),
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
	if err := os.WriteFile(manifestFile, []byte(baseManifest(validComponentTOML("c1", ""))), 0644); err != nil {
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

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"v2.0.0", "1.9.9", 1},
		{"1.0.0-rc1", "1.0.0", 0},
	}
	for _, tc := range cases {
		got, err := CompareVersions(tc.a, tc.b)
		if err != nil {
			t.Fatalf("unexpected error comparing %s vs %s: %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Errorf("CompareVersions(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestParseVersionRejectsGarbage(t *testing.T) {
	for _, v := range []string{"", "abc", "1.2", "1.2.3.4"} {
		if _, err := ParseVersion(v); err == nil {
			t.Errorf("expected ParseVersion(%q) to fail", v)
		}
	}
}
