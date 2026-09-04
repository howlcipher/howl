package plan

import (
	"bytes"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/state"
)

func testManifest(t *testing.T) *manifest.Manifest {
	t.Helper()
	toml := `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[components]]
name = "howlframe"
role = "language"
version = "0.1.1"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/howlframe"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "exec_version"
  args = ["--version"]

[[components]]
name = "howlchangeops"
role = "gate"
version = "0.1.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "source_build"
    [components.install.source_build]
    checkout_name = "howlchangeops"
    language = "go"
      [components.install.source_build.go]
      package = "./adapter"
      build_output = "howlchangeops"
  [components.health_check]
  type = "binary_exists"
  [[components.depends_on]]
  component = "howlframe"
  [[components.external_dependencies]]
  name = "go"
  required = true
  capability = "source-build"
`
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatalf("failed to load test manifest: %v", err)
	}
	return m
}

type fakeDetector struct {
	found map[string]string
}

func (f fakeDetector) Detect(name string) (bool, string) {
	v, ok := f.found[name]
	return ok, v
}

func TestBuildCleanInstall(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")

	p, err := Build(m, st, BuildOptions{Profile: "standard", Detector: fakeDetector{found: map[string]string{"go": "go1.22"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.Components) != 2 {
		t.Fatalf("expected 2 components in plan, got %d", len(p.Components))
	}
	if p.Components[0].Name != "howlframe" || p.Components[1].Name != "howlchangeops" {
		t.Fatalf("expected dependency order howlframe, howlchangeops; got %+v", p.Components)
	}
	for _, c := range p.Components {
		if c.Action != ActionInstall {
			t.Errorf("expected install action for %s, got %s", c.Name, c.Action)
		}
	}
	if !p.HasWork() {
		t.Error("expected plan to have work on a clean install")
	}
	if len(p.MissingRequired()) != 0 {
		t.Errorf("expected no missing required deps, got %+v", p.MissingRequired())
	}
}

func TestBuildIdempotentWhenAlreadyAtTargetVersion(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.1"}
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.1.0"}

	p, err := Build(m, st, BuildOptions{Profile: "standard", Detector: fakeDetector{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.HasWork() {
		t.Errorf("expected no-op plan when everything is already at target version, got %+v", p.Components)
	}
	for _, c := range p.Components {
		if c.Action != ActionSkip {
			t.Errorf("expected skip action for %s, got %s", c.Name, c.Action)
		}
	}
}

func TestBuildPartialExistingInstallProducesUpdate(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.0"} // older than manifest's 0.1.1

	p, err := Build(m, st, BuildOptions{Profile: "standard", Detector: fakeDetector{found: map[string]string{"go": "go1.22"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fp := p.Components[0]
	if fp.Name != "howlframe" || fp.Action != ActionUpdate || fp.FromVersion != "0.1.0" || fp.ToVersion != "0.1.1" {
		t.Errorf("expected howlframe update 0.1.0 -> 0.1.1, got %+v", fp)
	}
	cp := p.Components[1]
	if cp.Action != ActionInstall {
		t.Errorf("expected howlchangeops fresh install since it was never installed, got %s", cp.Action)
	}
}

func TestBuildMissingRequiredDependency(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")

	p, err := Build(m, st, BuildOptions{Profile: "standard", Detector: fakeDetector{}}) // "go" not found
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	missing := p.MissingRequired()
	if len(missing) != 1 || missing[0].Name != "go" {
		t.Fatalf("expected missing required dependency 'go', got %+v", missing)
	}
	if len(p.Warnings) == 0 {
		t.Errorf("expected a warning about the missing required dependency")
	}
}

func developerOverrideManifest(t *testing.T) *manifest.Manifest {
	t.Helper()
	toml := `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[components]]
name = "howlchangeops"
role = "gate"
version = "0.1.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/howlchangeops"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.developer_install]
  method = "source_build"
    [components.developer_install.source_build]
    checkout_name = "howlchangeops"
    language = "go"
      [components.developer_install.source_build.go]
      package = "./adapter"
      build_output = "howlchangeops"
  [components.health_check]
  type = "binary_exists"

[[components]]
name = "howlframe"
role = "language"
version = "0.1.1"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/howlframe"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "exec_version"
  args = ["--version"]
`
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatalf("failed to load developer-override test manifest: %v", err)
	}
	return m
}

func TestBuildDeveloperProfileUsesDeveloperInstall(t *testing.T) {
	m := developerOverrideManifest(t)
	st := state.New("stable")

	dev, err := Build(m, st, BuildOptions{Profile: "developer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdPlan, err := Build(m, st, BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var devMethod, stdMethod manifest.InstallMethod
	for _, cp := range dev.Components {
		if cp.Name == "howlchangeops" {
			devMethod = cp.Component.Install.Method
		}
	}
	for _, cp := range stdPlan.Components {
		if cp.Name == "howlchangeops" {
			stdMethod = cp.Component.Install.Method
		}
	}

	if devMethod != manifest.MethodSourceBuild {
		t.Errorf("expected developer profile to resolve howlchangeops to source_build, got %q", devMethod)
	}
	if stdMethod != manifest.MethodGithubRelease {
		t.Errorf("expected standard profile to resolve howlchangeops to github_release, got %q", stdMethod)
	}
}

func TestBuildStandardProfileNeverFallsBackWhenDeveloperInstallAbsent(t *testing.T) {
	// howlframe in developerOverrideManifest has no developer_install
	// stanza at all -- both profiles must resolve it identically, proving
	// its absence never synthesizes an implicit fallback.
	m := developerOverrideManifest(t)
	st := state.New("stable")

	dev, err := Build(m, st, BuildOptions{Profile: "developer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdPlan, err := Build(m, st, BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var devMethod, stdMethod manifest.InstallMethod
	for _, cp := range dev.Components {
		if cp.Name == "howlframe" {
			devMethod = cp.Component.Install.Method
		}
	}
	for _, cp := range stdPlan.Components {
		if cp.Name == "howlframe" {
			stdMethod = cp.Component.Install.Method
		}
	}
	if devMethod != manifest.MethodGithubRelease || stdMethod != manifest.MethodGithubRelease {
		t.Errorf("expected both profiles to resolve howlframe to github_release with no developer_install set, got developer=%q standard=%q", devMethod, stdMethod)
	}
}

func TestBuildOptionalCapabilityOnlyCheckedUnderLocalAIProfile(t *testing.T) {
	toml := `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[optional_capabilities]]
name = "ollama"
required = false
capability = "local-inference"

[[components]]
name = "howlframe"
role = "language"
version = "0.1.1"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/howlframe"
    artifact_pattern = "p"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "exec_version"
`
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatal(err)
	}
	st := state.New("stable")

	standard, err := Build(m, st, BuildOptions{Profile: "standard", Detector: fakeDetector{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range standard.Dependencies {
		if d.Name == "ollama" {
			t.Errorf("did not expect ollama to be checked under the standard profile")
		}
	}

	localAI, err := Build(m, st, BuildOptions{Profile: "local-ai", Detector: fakeDetector{}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range localAI.Dependencies {
		if d.Name == "ollama" {
			found = true
			if d.Found {
				t.Errorf("expected ollama to be reported not found")
			}
		}
	}
	if !found {
		t.Errorf("expected ollama to be checked under the local-ai profile")
	}
	if !localAI.HasWork() {
		t.Errorf("a missing optional capability must not block the rest of the install")
	}
}

func TestBuildUnknownComponentScopeErrors(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	_, err := Build(m, st, BuildOptions{Components: []string{"does-not-exist"}})
	if err == nil {
		t.Fatal("expected error for unknown component in scope")
	}
}

// TestBuildAgainstRealEmbeddedManifest proves the actual shipped
// ecosystem.toml -- not a synthetic fixture -- resolves to a complete,
// installable plan under both profiles, before any real release has been
// tagged for the three components this milestone moved off source_build.
func TestBuildAgainstRealEmbeddedManifest(t *testing.T) {
	m, path, err := manifest.LoadDefault(t.TempDir())
	if err != nil {
		t.Fatalf("failed to load the embedded default manifest: %v", err)
	}
	if path != "<embedded>" {
		t.Fatalf("expected the embedded manifest (no local override), got %s", path)
	}
	st := state.New("stable")

	wantStandardMethod := map[string]manifest.InstallMethod{
		"howlframe":        manifest.MethodGithubRelease,
		"howlchangeops":    manifest.MethodGithubRelease,
		"howlplane-engine": manifest.MethodGithubReleaseWheel,
		"howlplane":        manifest.MethodGithubRelease,
		"howlwriter":       manifest.MethodGithubReleaseWheel,
	}
	wantExposedOnBin := map[string]bool{
		"howlframe":        true,
		"howlchangeops":    true,
		"howlplane-engine": false,
		"howlplane":        true,
		"howlwriter":       true,
	}

	standard, err := Build(m, st, BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatalf("unexpected error building the standard-profile plan: %v", err)
	}
	if len(standard.Components) != len(wantStandardMethod) {
		t.Fatalf("expected %d components in the standard plan, got %d", len(wantStandardMethod), len(standard.Components))
	}
	for _, cp := range standard.Components {
		wantMethod, ok := wantStandardMethod[cp.Name]
		if !ok {
			t.Fatalf("unexpected component %q in the standard plan", cp.Name)
		}
		if cp.Component.Install.Method != wantMethod {
			t.Errorf("standard profile: expected %q to install via %q, got %q", cp.Name, wantMethod, cp.Component.Install.Method)
		}
		if cp.Component.Internal == wantExposedOnBin[cp.Name] {
			t.Errorf("standard profile: expected %q internal=%v (exposed on PATH=%v)", cp.Name, cp.Component.Internal, wantExposedOnBin[cp.Name])
		}
	}

	developer, err := Build(m, st, BuildOptions{Profile: "developer"})
	if err != nil {
		t.Fatalf("unexpected error building the developer-profile plan: %v", err)
	}
	for _, cp := range developer.Components {
		if cp.Name == "howlframe" {
			// howlframe has no developer_install: both profiles must
			// resolve it identically, proving absence never synthesizes
			// an implicit source-build fallback.
			if cp.Component.Install.Method != manifest.MethodGithubRelease {
				t.Errorf("developer profile: expected howlframe to still install via github_release (no developer_install declared), got %q", cp.Component.Install.Method)
			}
			continue
		}
		if cp.Component.Install.Method != manifest.MethodSourceBuild {
			t.Errorf("developer profile: expected %q to install via source_build, got %q", cp.Name, cp.Component.Install.Method)
		}
	}
}

func TestRenderProducesReadablePlan(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	p, err := Build(m, st, BuildOptions{Profile: "standard", Detector: fakeDetector{found: map[string]string{"go": "go1.22"}}})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	Render(&buf, p)
	out := buf.String()

	for _, want := range []string{"Howl Ecosystem 0.1.0", "howlframe", "howlchangeops", "install 0.1.1", "go"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected rendered plan to contain %q, got:\n%s", want, out)
		}
	}
}
