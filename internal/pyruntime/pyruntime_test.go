package pyruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type call struct {
	dir  string
	name string
	args []string
}

type fakeRunner struct {
	calls   []call
	outputs map[string]string // name -> output
	fail    map[string]bool
}

func (f *fakeRunner) Run(ctx context.Context, dir, name string, args []string) ([]byte, error) {
	f.calls = append(f.calls, call{dir: dir, name: name, args: args})
	if f.fail[name] {
		return nil, fmt.Errorf("simulated failure for %s", name)
	}
	if out, ok := f.outputs[name]; ok {
		return []byte(out), nil
	}
	return []byte(""), nil
}

func TestParsePythonVersion(t *testing.T) {
	if got := parsePythonVersion("Python 3.11.6\n"); got != "3.11.6" {
		t.Errorf("expected 3.11.6, got %q", got)
	}
	if got := parsePythonVersion("garbage"); got != "" {
		t.Errorf("expected empty for unparseable output, got %q", got)
	}
}

func TestCreateVenvInvokesCorrectCommand(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{}}
	venvDir := filepath.Join(t.TempDir(), "runtime", "venv")

	if err := CreateVenv(context.Background(), r, "/usr/bin/python3", venvDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(r.calls))
	}
	c := r.calls[0]
	if c.name != "/usr/bin/python3" || len(c.args) != 3 || c.args[0] != "-m" || c.args[1] != "venv" || c.args[2] != venvDir {
		t.Errorf("unexpected venv invocation: %+v", c)
	}
}

func TestCreateVenvPropagatesFailure(t *testing.T) {
	r := &fakeRunner{fail: map[string]bool{"/usr/bin/python3": true}}
	err := CreateVenv(context.Background(), r, "/usr/bin/python3", filepath.Join(t.TempDir(), "venv"))
	if err == nil {
		t.Fatal("expected error propagated from failed venv creation")
	}
}

func TestPipInstallEditableInvokesExpectedCommands(t *testing.T) {
	r := &fakeRunner{}
	venvDir := t.TempDir()
	checkoutDir := t.TempDir()

	if err := PipInstallEditable(context.Background(), r, venvDir, checkoutDir, ".", []string{"dev", "web"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.calls) != 2 {
		t.Fatalf("expected 2 calls (pip upgrade + install), got %d: %+v", len(r.calls), r.calls)
	}
	install := r.calls[1]
	wantTarget := ".[dev,web]"
	found := false
	for _, a := range install.args {
		if a == wantTarget {
			found = true
		}
	}
	if !found {
		t.Errorf("expected install args to include %q, got %v", wantTarget, install.args)
	}
	if install.dir != checkoutDir {
		t.Errorf("expected pip install to run in checkout dir %s, got %s", checkoutDir, install.dir)
	}
}

func TestPipInstallEditableUsesPackageSubdir(t *testing.T) {
	r := &fakeRunner{}
	venvDir := t.TempDir()
	checkoutDir := t.TempDir()

	if err := PipInstallEditable(context.Background(), r, venvDir, checkoutDir, "python", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	install := r.calls[1]
	want := filepath.Join(checkoutDir, "python")
	if install.dir != want {
		t.Errorf("expected pip install to run in %s, got %s", want, install.dir)
	}
}

func TestWriteWrapperScriptCreatesExecutableFile(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "bin", "howlwriter")
	venvDir := "/fake/venv"

	if err := WriteWrapperScript(scriptPath, venvDir, "howlwriter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("expected wrapper script to exist: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected wrapper script to be executable, got mode %v", info.Mode())
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), venvDir) {
		t.Errorf("expected wrapper script to reference venv path, got:\n%s", data)
	}
}

func TestVenvDirAndPythonPaths(t *testing.T) {
	root := "/data/runtimes/howlwriter"
	venv := VenvDir(root)
	if venv != filepath.Join(root, "venv") {
		t.Errorf("unexpected venv dir: %s", venv)
	}
	py := VenvPython(venv)
	if py == "" {
		t.Errorf("expected non-empty venv python path")
	}
}
