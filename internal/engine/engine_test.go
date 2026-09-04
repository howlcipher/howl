package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
)

// fakeInstaller lets tests script per-component/per-version success or
// failure without touching a real filesystem or subprocess.
type fakeInstaller struct {
	failOn        map[string]bool // component name -> fail
	calls         []string
	methodsByName map[string]manifest.InstallMethod
}

func (f *fakeInstaller) Install(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	f.calls = append(f.calls, fmt.Sprintf("%s@%s", c.Name, c.Version))
	if f.methodsByName == nil {
		f.methodsByName = map[string]manifest.InstallMethod{}
	}
	f.methodsByName[c.Name] = c.Install.Method
	if f.failOn[c.Name] {
		return fmt.Errorf("simulated install failure for %s", c.Name)
	}
	return nil
}

type fakeHealth struct {
	failOn map[string]bool
}

func (f *fakeHealth) Check(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	if f.failOn[c.Name] {
		return fmt.Errorf("simulated health check failure for %s", c.Name)
	}
	return nil
}

func twoComponentManifest(t *testing.T, v1, v2 string) *manifest.Manifest {
	t.Helper()
	toml := fmt.Sprintf(`
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[components]]
name = "howlframe"
role = "language"
version = "%s"
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
version = "%s"
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
`, v1, v2)
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatalf("failed to load test manifest: %v", err)
	}
	return m
}

func manifestWithDeveloperInstall(t *testing.T, version string) *manifest.Manifest {
	t.Helper()
	toml := fmt.Sprintf(`
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[components]]
name = "howlchangeops"
role = "gate"
version = "%s"
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
`, version)
	m, err := manifest.LoadBytes([]byte(toml))
	if err != nil {
		t.Fatalf("failed to load developer-install test manifest: %v", err)
	}
	return m
}

func TestEngineRollbackRespectsPersistedDeveloperProfile(t *testing.T) {
	m := manifestWithDeveloperInstall(t, "0.2.0")
	st := state.New("stable")
	st.Profile = "developer"
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.2.0"}
	st.PreviousComponents = map[string]state.ComponentState{"howlchangeops": {Version: "0.1.0"}}
	paths, statePath := testPaths(t)

	inst := &fakeInstaller{}
	eng := New(inst, &fakeHealth{}, paths)

	if _, err := eng.Rollback(context.Background(), m, st, statePath, "howlchangeops"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := inst.methodsByName["howlchangeops"]; got != manifest.MethodSourceBuild {
		t.Errorf("expected rollback under a developer-profile installation to reinstall via source_build, got %q", got)
	}
}

func TestEngineRollbackUsesStandardInstallWhenProfileUnset(t *testing.T) {
	m := manifestWithDeveloperInstall(t, "0.2.0")
	st := state.New("stable")
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.2.0"}
	st.PreviousComponents = map[string]state.ComponentState{"howlchangeops": {Version: "0.1.0"}}
	paths, statePath := testPaths(t)

	inst := &fakeInstaller{}
	eng := New(inst, &fakeHealth{}, paths)

	if _, err := eng.Rollback(context.Background(), m, st, statePath, "howlchangeops"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := inst.methodsByName["howlchangeops"]; got != manifest.MethodGithubRelease {
		t.Errorf("expected rollback with no persisted developer profile to use github_release, got %q", got)
	}
}

func testPaths(t *testing.T) (platform.Paths, string) {
	t.Helper()
	home := t.TempDir()
	paths := platform.ResolvePaths(func(string) string { return "" }, home)
	statePath := filepath.Join(paths.StateDir(), "state.json")
	return paths, statePath
}

func TestEngineCleanInstall(t *testing.T) {
	m := twoComponentManifest(t, "0.1.1", "0.1.0")
	st := state.New("stable")
	paths, statePath := testPaths(t)

	p, err := plan.Build(m, st, plan.BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatal(err)
	}

	installer := &fakeInstaller{failOn: map[string]bool{}}
	health := &fakeHealth{failOn: map[string]bool{}}
	eng := New(installer, health, paths)

	result, err := eng.Install(context.Background(), p, st, statePath)
	if err != nil {
		t.Fatalf("unexpected install error: %v", err)
	}
	if !result.Succeeded() {
		t.Fatalf("expected success, got %+v", result)
	}
	if len(installer.calls) != 2 {
		t.Fatalf("expected 2 install calls, got %v", installer.calls)
	}
	if st.Components["howlframe"].Version != "0.1.1" {
		t.Errorf("expected howlframe recorded at 0.1.1, got %+v", st.Components["howlframe"])
	}
	if st.InProgress != nil {
		t.Errorf("expected in-progress marker cleared after successful install")
	}

	reloaded, existed, err := state.Load(statePath)
	if err != nil || !existed {
		t.Fatalf("expected persisted state to load back, err=%v existed=%v", err, existed)
	}
	if reloaded.Components["howlchangeops"].Version != "0.1.0" {
		t.Errorf("expected persisted state to include howlchangeops")
	}
}

func TestEngineInstallFailureStopsAndLeavesPriorComponentsInstalled(t *testing.T) {
	m := twoComponentManifest(t, "0.1.1", "0.1.0")
	st := state.New("stable")
	paths, statePath := testPaths(t)

	p, err := plan.Build(m, st, plan.BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatal(err)
	}

	installer := &fakeInstaller{failOn: map[string]bool{"howlchangeops": true}}
	health := &fakeHealth{}
	eng := New(installer, health, paths)

	result, err := eng.Install(context.Background(), p, st, statePath)
	if err == nil {
		t.Fatal("expected install error")
	}
	if result.Succeeded() {
		t.Fatal("expected failure result")
	}
	if _, ok := st.Components["howlframe"]; !ok {
		t.Errorf("expected howlframe (installed before the failure) to remain recorded")
	}
	if _, ok := st.Components["howlchangeops"]; ok {
		t.Errorf("expected howlchangeops (the failing component) to not be recorded")
	}
}

func TestEngineInstallHealthCheckFailureStops(t *testing.T) {
	m := twoComponentManifest(t, "0.1.1", "0.1.0")
	st := state.New("stable")
	paths, statePath := testPaths(t)

	p, err := plan.Build(m, st, plan.BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatal(err)
	}

	installer := &fakeInstaller{}
	health := &fakeHealth{failOn: map[string]bool{"howlframe": true}}
	eng := New(installer, health, paths)

	result, err := eng.Install(context.Background(), p, st, statePath)
	if err == nil {
		t.Fatal("expected health check failure to surface as an error")
	}
	if len(result.Components) != 1 || result.Components[0].Err == nil {
		t.Fatalf("expected a single failed component result, got %+v", result.Components)
	}
}

func TestEngineUpdateSuccess(t *testing.T) {
	m := twoComponentManifest(t, "0.2.0", "0.1.0")
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.0"}
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.1.0"}
	paths, statePath := testPaths(t)

	p, err := plan.Build(m, st, plan.BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatal(err)
	}

	installer := &fakeInstaller{}
	health := &fakeHealth{}
	eng := New(installer, health, paths)

	result, err := eng.Update(context.Background(), p, m, st, statePath)
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if !result.Succeeded() {
		t.Fatalf("expected update success, got %+v", result)
	}
	if st.Components["howlframe"].Version != "0.2.0" {
		t.Errorf("expected howlframe updated to 0.2.0, got %+v", st.Components["howlframe"])
	}
	if !st.HasRollbackTarget() || st.PreviousComponents["howlframe"].Version != "0.1.0" {
		t.Errorf("expected previous version 0.1.0 recorded as rollback target")
	}
}

func TestEngineUpdateFailureTriggersAutomaticRollback(t *testing.T) {
	m := twoComponentManifest(t, "0.2.0", "0.2.0")
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.0"}
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.1.0"}
	paths, statePath := testPaths(t)

	p, err := plan.Build(m, st, plan.BuildOptions{Profile: "standard"})
	if err != nil {
		t.Fatal(err)
	}

	// howlchangeops fails to install at its target version during the
	// update; automatic rollback should restore both components
	// (howlframe already updated, howlchangeops never got past 0.1.0)
	// back to their pre-update versions.
	health := &fakeHealth{}
	failingInstaller := &versionFailInstaller{failVersion: map[string]string{"howlchangeops": "0.2.0"}}
	eng := New(failingInstaller, health, paths)

	result, err := eng.Update(context.Background(), p, m, st, statePath)
	if err == nil {
		t.Fatal("expected update to fail")
	}
	if result == nil {
		t.Fatal("expected a non-nil result even on failure")
	}

	if st.Components["howlframe"].Version != "0.1.0" {
		t.Errorf("expected howlframe rolled back to 0.1.0, got %+v", st.Components["howlframe"])
	}
	if st.Components["howlchangeops"].Version != "0.1.0" {
		t.Errorf("expected howlchangeops still at 0.1.0, got %+v", st.Components["howlchangeops"])
	}
}

type versionFailInstaller struct {
	failVersion map[string]string
}

func (f *versionFailInstaller) Install(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	if f.failVersion[c.Name] == c.Version {
		return fmt.Errorf("simulated failure installing %s@%s", c.Name, c.Version)
	}
	return nil
}

func TestEngineRollbackWithNoTargetFails(t *testing.T) {
	st := state.New("stable")
	paths, statePath := testPaths(t)
	m := twoComponentManifest(t, "0.1.1", "0.1.0")

	eng := New(&fakeInstaller{}, &fakeHealth{}, paths)
	_, err := eng.Rollback(context.Background(), m, st, statePath, "")
	if err == nil {
		t.Fatal("expected error rolling back with no recorded target")
	}
}

func TestEngineRollbackVerificationFailure(t *testing.T) {
	m := twoComponentManifest(t, "0.1.1", "0.1.0")
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.2.0"}
	st.PreviousComponents = map[string]state.ComponentState{"howlframe": {Version: "0.1.0"}}
	paths, statePath := testPaths(t)

	health := &fakeHealth{failOn: map[string]bool{"howlframe": true}}
	eng := New(&fakeInstaller{}, health, paths)

	result, err := eng.Rollback(context.Background(), m, st, statePath, "")
	if err == nil {
		t.Fatal("expected rollback verification failure to surface as an error")
	}
	if len(result.Components) != 1 || result.Components[0].Err == nil {
		t.Fatalf("expected failed rollback component result, got %+v", result.Components)
	}
	// State must not claim the rollback succeeded.
	if st.Components["howlframe"].Version != "0.2.0" {
		t.Errorf("expected component version left unchanged after failed rollback verification, got %+v", st.Components["howlframe"])
	}
}

func TestEngineManualRollbackSingleComponent(t *testing.T) {
	m := twoComponentManifest(t, "0.1.1", "0.1.0")
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.2.0"}
	st.Components["howlchangeops"] = state.ComponentState{Version: "0.5.0"}
	st.PreviousComponents = map[string]state.ComponentState{
		"howlframe":     {Version: "0.1.0"},
		"howlchangeops": {Version: "0.4.0"},
	}
	paths, statePath := testPaths(t)

	eng := New(&fakeInstaller{}, &fakeHealth{}, paths)
	_, err := eng.Rollback(context.Background(), m, st, statePath, "howlframe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Components["howlframe"].Version != "0.1.0" {
		t.Errorf("expected howlframe rolled back, got %+v", st.Components["howlframe"])
	}
	if st.Components["howlchangeops"].Version != "0.5.0" {
		t.Errorf("expected howlchangeops untouched by a single-component rollback, got %+v", st.Components["howlchangeops"])
	}
}
