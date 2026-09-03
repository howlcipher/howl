package status

import (
	"bytes"
	"encoding/json"
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
version = "0.2.0"
channel = "stable"

[[components]]
name = "howlframe"
display_name = "HowlFrame"
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

[[components]]
name = "howlwriter"
role = "writing"
version = "0.1.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "source_build"
    [components.install.source_build]
    checkout_name = "howlwriter"
    language = "python"
      [components.install.source_build.python]
      console_script = "howlwriter"
  [components.health_check]
  type = "binary_exists"
`
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatalf("failed to load test manifest: %v", err)
	}
	return m
}

func TestCollectNothingInstalled(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")

	report := Collect(m, st, "0.1.0")

	if report.EcosystemName != "Howl" {
		t.Errorf("expected Howl, got %s", report.EcosystemName)
	}
	if len(report.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(report.Components))
	}
	for _, c := range report.Components {
		if c.Installed {
			t.Errorf("expected %s to be reported not installed", c.Name)
		}
	}
	if !report.UpdatesAvailable {
		t.Errorf("expected updates available when nothing is installed")
	}
}

func TestCollectUpToDate(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.1"}
	st.Components["howlwriter"] = state.ComponentState{Version: "0.1.0"}

	report := Collect(m, st, "0.1.0")

	if report.UpdatesAvailable {
		t.Errorf("expected no updates available when everything matches target")
	}
	for _, c := range report.Components {
		if !c.Installed || !c.UpToDate {
			t.Errorf("expected %s to be installed and up to date, got %+v", c.Name, c)
		}
	}
}

func TestCollectPartiallyOutOfDate(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.0"} // older than target 0.1.1

	report := Collect(m, st, "0.1.0")

	if !report.UpdatesAvailable {
		t.Fatal("expected updates available")
	}

	frame := findComponent(t, report, "howlframe")
	if !frame.Installed || frame.UpToDate {
		t.Errorf("expected howlframe installed but out of date, got %+v", frame)
	}
	writer := findComponent(t, report, "howlwriter")
	if writer.Installed {
		t.Errorf("expected howlwriter reported not installed, got %+v", writer)
	}
}

func findComponent(t *testing.T, report *Report, name string) ComponentStatus {
	t.Helper()
	for _, c := range report.Components {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("component %s not found in report", name)
	return ComponentStatus{}
}

func TestRenderHuman(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	report := Collect(m, st, "0.1.0")

	var buf bytes.Buffer
	RenderHuman(&buf, report)
	if !bytes.Contains(buf.Bytes(), []byte("Howl Ecosystem")) {
		t.Errorf("expected header in human output, got:\n%s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("HowlFrame")) {
		t.Errorf("expected component display name in output, got:\n%s", buf.String())
	}
}

func TestRenderJSON(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	report := Collect(m, st, "0.1.0")

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed Report
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.EcosystemName != "Howl" {
		t.Errorf("expected round-tripped ecosystem name")
	}
}
