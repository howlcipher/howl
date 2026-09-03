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
	if err := component.ActivateRelease(paths, name, "1.0.0"); err != nil {
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
