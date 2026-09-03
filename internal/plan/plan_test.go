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
