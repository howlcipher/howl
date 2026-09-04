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
	if len(m.Components) != 5 {
		t.Errorf("expected 5 components, got %d", len(m.Components))
	}

	order, err := m.TopoOrder()
	if err != nil {
		t.Fatalf("expected embedded manifest to have a valid dependency order: %v", err)
	}
	want := []string{"howlframe", "howlchangeops", "howlplane-engine", "howlplane", "howlwriter"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("expected install order %v, got %v", want, order)
			break
		}
	}

	engine, ok := m.GetComponent("howlplane-engine")
	if !ok {
		t.Fatal("expected howlplane-engine component to exist")
	}
	if !engine.Internal {
		t.Error("expected howlplane-engine to be marked internal (never exposed on PATH)")
	}
	if engine.Install.Method != MethodGithubReleaseWheel {
		t.Errorf("expected howlplane-engine to install via github_release_python_wheel, got %q", engine.Install.Method)
	}

	for _, name := range []string{"howlframe", "howlchangeops", "howlplane", "howlwriter"} {
		c, ok := m.GetComponent(name)
		if !ok {
			t.Fatalf("expected component %q to exist", name)
		}
		if c.Internal {
			t.Errorf("expected %q to be exposed on PATH (not internal)", name)
		}
	}

	// Every non-howlframe component must declare a developer_install,
	// since none of them can be installed the old way anymore by default
	// -- --profile developer is the only way to still build from source.
	for _, name := range []string{"howlchangeops", "howlplane-engine", "howlplane", "howlwriter"} {
		c, ok := m.GetComponent(name)
		if !ok {
			t.Fatalf("expected component %q to exist", name)
		}
		if c.DeveloperInstall == nil {
			t.Errorf("expected %q to declare a developer_install", name)
		} else if c.DeveloperInstall.Method != MethodSourceBuild {
			t.Errorf("expected %q's developer_install to be source_build, got %q", name, c.DeveloperInstall.Method)
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

func TestLoadValidManifestWithGithubReleaseWheel(t *testing.T) {
	tomlData := baseManifest(`
[[components]]
name = "howlwriter"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release_python_wheel"
    [components.install.github_release_python_wheel]
    repository = "x/howlwriter"
    artifact_pattern = "howlwriter-{pep440_version}-py3-none-any.whl"
    checksum_file = "SHA256SUMS"
    console_script = "howlwriter"
    min_python = "3.11.0"
  [components.health_check]
  type = "python_import"
  module = "howlwriter"
`)
	m, err := LoadBytes([]byte(tomlData))
	if err != nil {
		t.Fatalf("failed to load valid github_release_python_wheel manifest: %v", err)
	}
	c, ok := m.GetComponent("howlwriter")
	if !ok {
		t.Fatal("failed to get component howlwriter")
	}
	if c.Install.Method != MethodGithubReleaseWheel {
		t.Errorf("expected method %q, got %q", MethodGithubReleaseWheel, c.Install.Method)
	}
	if c.Install.GithubReleaseWheel == nil || c.Install.GithubReleaseWheel.ConsoleScript != "howlwriter" {
		t.Errorf("expected github_release_python_wheel block to be populated, got %+v", c.Install.GithubReleaseWheel)
	}
}

func TestLoadValidManifestWithDeveloperInstall(t *testing.T) {
	tomlData := baseManifest(validComponentTOML("howlchangeops", "") + `
[components.developer_install]
method = "source_build"
  [components.developer_install.source_build]
  checkout_name = "howlchangeops"
  language = "go"
    [components.developer_install.source_build.go]
    package = "./adapter"
    build_output = "howlchangeops"
`)
	m, err := LoadBytes([]byte(tomlData))
	if err != nil {
		t.Fatalf("failed to load manifest with developer_install: %v", err)
	}
	c, ok := m.GetComponent("howlchangeops")
	if !ok {
		t.Fatal("failed to get component howlchangeops")
	}
	if c.Install.Method != MethodGithubRelease {
		t.Errorf("expected standard install method github_release, got %q", c.Install.Method)
	}
	if c.DeveloperInstall == nil || c.DeveloperInstall.Method != MethodSourceBuild {
		t.Fatalf("expected developer_install to be a valid source_build stanza, got %+v", c.DeveloperInstall)
	}
	if c.DeveloperInstall.SourceBuild == nil || c.DeveloperInstall.SourceBuild.Go.Package != "./adapter" {
		t.Errorf("expected developer_install.source_build.go to be populated, got %+v", c.DeveloperInstall.SourceBuild)
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
		{
			name: "github_release_python_wheel missing block",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release_python_wheel"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "github_release_python_wheel missing console_script and min_python",
			toml: baseManifest(`
[[components]]
name = "c1"
role = "role"
version = "1.0.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release_python_wheel"
    [components.install.github_release_python_wheel]
    repository = "x/y"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`),
		},
		{
			name: "developer_install with an invalid stanza is still caught",
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
  [components.developer_install]
  method = "source_build"
  [components.health_check]
  type = "binary_exists"
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
