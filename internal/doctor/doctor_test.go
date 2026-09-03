package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
)

type fakeDetector struct{ found map[string]string }

func (f fakeDetector) Detect(name string) (bool, string) {
	v, ok := f.found[name]
	return ok, v
}

type fakeHealth struct{ failOn map[string]bool }

func (f fakeHealth) Check(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	if f.failOn[c.Name] {
		return fmt.Errorf("simulated unhealthy %s", c.Name)
	}
	return nil
}

type fakeInstaller struct{ calls []string }

func (f *fakeInstaller) Install(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	f.calls = append(f.calls, c.Name)
	return nil
}

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
  min_version = "0.1.0"
  [[components.external_dependencies]]
  name = "go"
  required = true
  capability = "source-build"
`
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testPaths(t *testing.T) platform.Paths {
	t.Helper()
	home := t.TempDir()
	paths := platform.ResolvePaths(func(string) string { return "" }, home)
	if err := paths.EnsureOwnedDirs(); err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestDoctorHealthySystem(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.1"}
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.1.0"}
	paths := testPaths(t)

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{found: map[string]string{"go": "go1.22"}},
		Health:    fakeHealth{},
	})

	if report.Summary != HealthHealthy {
		t.Fatalf("expected HEALTHY, got %s: %+v", report.Summary, report.Checks)
	}
}

func TestDoctorMissingRequiredDependency(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	paths := testPaths(t)

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{}, // "go" not found
		Health:    fakeHealth{},
	})

	if report.Summary != HealthFailed {
		t.Fatalf("expected FAILED for missing required dependency, got %s", report.Summary)
	}
}

func TestDoctorOptionalCapabilityWarns(t *testing.T) {
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
	paths := testPaths(t)

	// Only local-ai profile checks optional_capabilities; doctor always
	// reports on every declared optional capability regardless of the
	// active profile, since it's diagnosing the whole manifest.
	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{},
		Health:    fakeHealth{},
	})

	if report.Summary == HealthFailed {
		t.Fatalf("a missing optional capability must not fail doctor, got %+v", report.Checks)
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "ollama" && c.Status == StatusWarn {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a WARN check for the missing optional ollama capability, got %+v", report.Checks)
	}
}

func TestDoctorInterruptedOperationDetectedAndFixed(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.BeginOperation(state.OpInstall, "howlframe", time.Now())
	paths := testPaths(t)
	statePath := filepath.Join(paths.StateDir(), "state.json")
	if err := st.Save(statePath); err != nil {
		t.Fatal(err)
	}

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: statePath,
		Paths:     paths,
		Detector:  fakeDetector{found: map[string]string{"go": "go1.22"}},
		Health:    fakeHealth{},
		Fix:       false,
	})
	if !containsCheck(report, "State", StatusWarn) {
		t.Fatalf("expected a WARN check for the interrupted operation, got %+v", report.Checks)
	}
	if st.InProgress == nil {
		t.Fatal("expected in-progress marker to remain without --fix")
	}

	report2 := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: statePath,
		Paths:     paths,
		Detector:  fakeDetector{found: map[string]string{"go": "go1.22"}},
		Health:    fakeHealth{},
		Fix:       true,
	})
	_ = report2
	if st.InProgress != nil {
		t.Errorf("expected --fix to clear the interrupted-operation marker")
	}
}

func TestDoctorStaleLockDetectedAndFixed(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	paths := testPaths(t)

	lock, err := state.Acquire(paths.LockFile())
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the lock-holder process being gone by overwriting the lock
	// file with a PID that cannot possibly be alive.
	lock.Release()
	writeStaleLock(t, paths.LockFile())

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{found: map[string]string{"go": "go1.22"}},
		Health:    fakeHealth{},
		Fix:       true,
	})

	found := false
	for _, c := range report.Checks {
		if c.Name == "installation lock" && c.Fixed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the stale lock to be fixed, got %+v", report.Checks)
	}
	if _, exists, _ := state.Inspect(paths.LockFile()); exists {
		t.Errorf("expected stale lock file to be removed")
	}
}

func TestDoctorComponentHealthFailureAndRepair(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.1"}
	paths := testPaths(t)
	installer := &fakeInstaller{}

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{found: map[string]string{"go": "go1.22"}},
		Health:    fakeHealth{failOn: map[string]bool{"howlframe": true}},
		Installer: installer,
		Fix:       true,
	})

	if len(installer.calls) != 1 || installer.calls[0] != "howlframe" {
		t.Fatalf("expected doctor --fix to reinstall the unhealthy component, calls=%v", installer.calls)
	}
	// The fake installer "fixes" it (fakeHealth still reports it failing on
	// recheck since failOn is unconditional), so this should report the
	// repair attempt in the message without crashing.
	found := false
	for _, c := range report.Checks {
		if c.Category == "Component Health" && c.Name == "howlframe" {
			found = true
			if c.Status != StatusFail {
				t.Errorf("expected still-FAIL status since the fake health check unconditionally fails, got %s", c.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected a Component Health check for howlframe")
	}
}

func TestDoctorIncompatibleComponentVersions(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.0.1"} // below howlchangeops' min_version 0.1.0
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.1.0"}
	paths := testPaths(t)

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{found: map[string]string{"go": "go1.22"}},
		Health:    fakeHealth{},
	})

	if report.Summary != HealthFailed {
		t.Fatalf("expected FAILED for incompatible installed versions, got %s: %+v", report.Summary, report.Checks)
	}
}

func TestRenderHumanAndJSON(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	paths := testPaths(t)

	report := Run(context.Background(), Options{
		Manifest:  m,
		State:     st,
		StatePath: filepath.Join(paths.StateDir(), "state.json"),
		Paths:     paths,
		Detector:  fakeDetector{},
		Health:    fakeHealth{},
	})

	var buf bytes.Buffer
	RenderHuman(&buf, report, true)
	if !bytes.Contains(buf.Bytes(), []byte("Howl Doctor")) {
		t.Errorf("expected header in human output")
	}

	var jsonBuf bytes.Buffer
	if err := RenderJSON(&jsonBuf, report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed DiagnosticReport
	if err := json.Unmarshal(jsonBuf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func containsCheck(report *DiagnosticReport, category string, status CheckStatus) bool {
	for _, c := range report.Checks {
		if c.Category == category && c.Status == status {
			return true
		}
	}
	return false
}

func writeStaleLock(t *testing.T, path string) {
	t.Helper()
	// PID astronomically unlikely to be alive.
	content := "999999999\n2020-01-01T00:00:00Z\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
