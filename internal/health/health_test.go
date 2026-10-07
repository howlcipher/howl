package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/howlcipher/howl/internal/component"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/pyruntime"
)

type fakeExecer struct {
	fail   bool
	output string
}

func (f fakeExecer) Exec(ctx context.Context, path string, args []string) ([]byte, error) {
	if f.fail {
		return nil, fmt.Errorf("simulated exec failure")
	}
	return []byte(f.output), nil
}

func testPaths(t *testing.T) platform.Paths {
	t.Helper()
	home := t.TempDir()
	p := platform.ResolvePaths(func(string) string { return "" }, home)
	if err := p.EnsureOwnedDirs(); err != nil {
		t.Fatal(err)
	}
	return p
}

func activateFakeBinary(t *testing.T, paths platform.Paths, name string, executable bool) {
	t.Helper()
	dir := paths.ComponentReleaseDir(name, "1.0.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0o644)
	if executable {
		mode = 0o755
	}
	if err := os.WriteFile(filepath.Join(dir, platform.ExeName(name)), []byte("bin"), mode); err != nil {
		t.Fatal(err)
	}
	if err := component.ActivateRelease(paths, name, "1.0.0", true); err != nil {
		t.Fatal(err)
	}
}

func TestCheckBinaryExistsPass(t *testing.T) {
	paths := testPaths(t)
	activateFakeBinary(t, paths, "howlchangeops", true)

	c := manifest.Component{Name: "howlchangeops", HealthCheck: manifest.HealthCheck{Type: manifest.HealthBinaryExists}}
	if err := (Checker{}).Check(context.Background(), c, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBinaryExistsFailsWhenNotActivated(t *testing.T) {
	paths := testPaths(t)
	c := manifest.Component{Name: "howlchangeops", HealthCheck: manifest.HealthCheck{Type: manifest.HealthBinaryExists}}
	if err := (Checker{}).Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected failure when nothing is activated")
	}
}

func TestCheckBinaryExistsFailsWhenNotExecutable(t *testing.T) {
	paths := testPaths(t)
	activateFakeBinary(t, paths, "howlchangeops", false)

	c := manifest.Component{Name: "howlchangeops", HealthCheck: manifest.HealthCheck{Type: manifest.HealthBinaryExists}}
	if err := (Checker{}).Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected failure for a non-executable file")
	}
}

func TestCheckExecVersionPass(t *testing.T) {
	paths := testPaths(t)
	activateFakeBinary(t, paths, "howlframe", true)

	c := manifest.Component{Name: "howlframe", HealthCheck: manifest.HealthCheck{Type: manifest.HealthExecVersion, Args: []string{"--version"}}}
	checker := Checker{Exec: fakeExecer{output: "howlframe 0.1.1\n"}}
	if err := checker.Check(context.Background(), c, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckExecVersionFailsOnEmptyOutput(t *testing.T) {
	paths := testPaths(t)
	activateFakeBinary(t, paths, "howlframe", true)

	c := manifest.Component{Name: "howlframe", HealthCheck: manifest.HealthCheck{Type: manifest.HealthExecVersion}}
	checker := Checker{Exec: fakeExecer{output: "   \n"}}
	if err := checker.Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected failure for empty version output")
	}
}

func TestCheckExecVersionFailsOnExecError(t *testing.T) {
	paths := testPaths(t)
	activateFakeBinary(t, paths, "howlframe", true)

	c := manifest.Component{Name: "howlframe", HealthCheck: manifest.HealthCheck{Type: manifest.HealthExecVersion}}
	checker := Checker{Exec: fakeExecer{fail: true}}
	if err := checker.Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected failure when the binary fails to execute")
	}
}

func TestCheckPythonImportPass(t *testing.T) {
	paths := testPaths(t)
	venvDir := pyruntime.VenvDir(paths.RuntimeDir("howlwriter"))
	pyPath := pyruntime.VenvPython(venvDir)
	if err := os.MkdirAll(filepath.Dir(pyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pyPath, []byte("fake python"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := manifest.Component{Name: "howlwriter", HealthCheck: manifest.HealthCheck{Type: manifest.HealthPythonImport, Module: "howlwriter"}}
	checker := Checker{Exec: fakeExecer{}}
	if err := checker.Check(context.Background(), c, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPythonImportFailsWithoutVenv(t *testing.T) {
	paths := testPaths(t)
	c := manifest.Component{Name: "howlwriter", HealthCheck: manifest.HealthCheck{Type: manifest.HealthPythonImport, Module: "howlwriter"}}
	checker := Checker{Exec: fakeExecer{}}
	if err := checker.Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected failure when no venv exists")
	}
}

func TestCheckUnsupportedType(t *testing.T) {
	paths := testPaths(t)
	c := manifest.Component{Name: "x", HealthCheck: manifest.HealthCheck{Type: "something-else"}}
	if err := (Checker{}).Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected error for unsupported health check type")
	}
}

func TestCheckFailsWhenEditableRuntimeDependenciesDrifted(t *testing.T) {
	// DOG-031: `--version` passed while the runtime lacked a dependency its
	// checkout had since declared; the health check must not stop there.
	paths := testPaths(t)
	activateFakeBinary(t, paths, "howlwriter", true)
	checkout := t.TempDir()
	if err := os.WriteFile(filepath.Join(checkout, "pyproject.toml"), []byte("[project]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtimeRoot := paths.RuntimeDir("howlwriter")
	distInfo := filepath.Join(pyruntime.VenvDir(runtimeRoot), "lib", "python3.14", "site-packages", "howlwriter-0.1.0.dist-info")
	if err := os.MkdirAll(distInfo, 0o755); err != nil {
		t.Fatal(err)
	}
	record := `{"dir_info": {"editable": true}, "url": "file://` + filepath.ToSlash(checkout) + `"}`
	if err := os.WriteFile(filepath.Join(distInfo, "direct_url.json"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pyruntime.RecordSourceDependencies(runtimeRoot, checkout); err != nil {
		t.Fatal(err)
	}
	c := manifest.Component{Name: "howlwriter", HealthCheck: manifest.HealthCheck{Type: manifest.HealthExecVersion}}
	checker := Checker{Exec: fakeExecer{output: "howlwriter 0.1.0\n"}}
	if err := checker.Check(context.Background(), c, paths); err != nil {
		t.Fatalf("in sync: %v", err)
	}

	if err := os.WriteFile(filepath.Join(checkout, "pyproject.toml"), []byte("[project]\ndependencies = [\"howl-provider-core\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checker.Check(context.Background(), c, paths); err == nil {
		t.Fatal("expected drift to fail the health check")
	}
}
