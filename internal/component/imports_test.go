package component

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/artifact"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/pyruntime"
)

func pythonSourceComponent(name string) manifest.Component {
	return manifest.Component{
		Name:    name,
		Version: "0.1.0",
		Install: manifest.Install{
			Method: manifest.MethodSourceBuild,
			SourceBuild: &manifest.SourceBuild{
				CheckoutName: name,
				Language:     "python",
				Python:       &manifest.PythonSourceBuild{PackageDir: ".", ConsoleScript: name, MinPython: "3.0.0"},
			},
		},
	}
}

func checkoutWithPyproject(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \""+name+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// editableInstallDirs lists the directories pip ran `install -e` in, for the given venv.
func editableInstallDirs(calls []call, venvDir string) []string {
	var dirs []string
	for _, c := range calls {
		if c.name == pyruntime.VenvPython(venvDir) && strings.Contains(strings.Join(c.args, " "), "install --quiet -e") {
			dirs = append(dirs, c.dir)
		}
	}
	return dirs
}

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err := exec.LookPath("python"); err != nil {
			t.Skip("a python interpreter on PATH is required to plan a Python runtime")
		}
	}
}

func TestInstallPythonSourceInstallsImportedCheckoutIntoItsRuntime(t *testing.T) {
	requirePython(t)
	writerDir, engineDir := checkoutWithPyproject(t, "writer"), checkoutWithPyproject(t, "engine")
	writer := pythonSourceComponent("writer")
	writer.Imports = []manifest.Component{pythonSourceComponent("engine")}

	paths := testPaths(t)
	runner := &fakeRunner{}
	inst := &Installer{
		Runner:               runner,
		Getenv:               func(string) string { return "" },
		ExplicitCheckoutDirs: map[string]string{"writer": writerDir, "engine": engineDir},
	}
	if err := inst.Install(context.Background(), writer, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	venvDir := pyruntime.VenvDir(paths.RuntimeDir("writer"))
	got := editableInstallDirs(runner.calls, venvDir)
	if len(got) != 2 || got[0] != writerDir || got[1] != engineDir {
		t.Fatalf("expected editable installs of writer then engine into writer's venv, got %v (calls %+v)", got, runner.calls)
	}

	data, err := os.ReadFile(filepath.Join(paths.RuntimeDir("writer"), pyruntime.SourceDepsMarker))
	if err != nil {
		t.Fatal(err)
	}
	var marker struct {
		PackageRoot string `json:"package_root"`
		Imports     []struct {
			PackageRoot string `json:"package_root"`
		} `json:"imports"`
	}
	if err := json.Unmarshal(data, &marker); err != nil {
		t.Fatal(err)
	}
	if marker.PackageRoot != writerDir || len(marker.Imports) != 1 || marker.Imports[0].PackageRoot != engineDir {
		t.Fatalf("marker must record writer and its imported engine checkout, got %s", data)
	}
}

func TestInstallPythonSourceReusesTheImportedComponentsOwnCheckout(t *testing.T) {
	// A repair without HOWL_<NAME>_DIR finds the imported checkout the way
	// the imported component's own runtime records it.
	requirePython(t)
	writerDir, engineDir := checkoutWithPyproject(t, "writer"), checkoutWithPyproject(t, "engine")
	paths := testPaths(t)
	if err := os.MkdirAll(paths.RuntimeDir("engine"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := pyruntime.RecordSourceDependencies(paths.RuntimeDir("engine"), engineDir); err != nil {
		t.Fatal(err)
	}
	writer := pythonSourceComponent("writer")
	writer.Imports = []manifest.Component{pythonSourceComponent("engine")}
	runner := &fakeRunner{}
	inst := &Installer{
		Runner:               runner,
		Getenv:               func(string) string { return "" },
		ExplicitCheckoutDirs: map[string]string{"writer": writerDir},
	}
	if err := inst.Install(context.Background(), writer, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := editableInstallDirs(runner.calls, pyruntime.VenvDir(paths.RuntimeDir("writer")))
	if len(got) != 2 || got[1] != engineDir {
		t.Fatalf("expected the engine's recorded checkout to be installed, got %v", got)
	}

	// With no checkout recorded anywhere, the install fails rather than guess.
	os.Remove(filepath.Join(paths.RuntimeDir("engine"), pyruntime.SourceDepsMarker))
	err := inst.Install(context.Background(), writer, paths)
	if err == nil || !strings.Contains(err.Error(), "imported by writer") {
		t.Fatalf("expected a failure naming the import, got %v", err)
	}
}

// wheelServer serves each named wheel with a SHA256SUMS whose entry for it
// is correct unless that wheel is listed in corrupt.
func wheelServer(t *testing.T, names []string, corrupt map[string]bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, name := range names {
		wheel := fmt.Sprintf("%s-1.0.0-py3-none-any.whl", name)
		content := []byte("fake wheel for " + name)
		sum := sha256.Sum256(content)
		if corrupt[name] {
			sum = sha256.Sum256([]byte("something else"))
		}
		sums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), wheel)
		mux.HandleFunc("/x/"+name+"/releases/download/v1.0.0/"+wheel, func(w http.ResponseWriter, r *http.Request) {
			w.Write(content)
		})
		mux.HandleFunc("/x/"+name+"/releases/download/v1.0.0/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(sums))
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func wheelComponent(name string) manifest.Component {
	return manifest.Component{
		Name:    name,
		Version: "1.0.0",
		Install: manifest.Install{
			Method: manifest.MethodGithubReleaseWheel,
			GithubReleaseWheel: &manifest.GithubReleaseWheel{
				Repository:      "x/" + name,
				ArtifactPattern: name + "-{pep440_version}-py3-none-any.whl",
				ChecksumFile:    "SHA256SUMS",
				ConsoleScript:   name,
				MinPython:       "3.0.0",
			},
		},
	}
}

func TestInstallGithubReleaseWheelInstallsImportedWheelIntoItsRuntime(t *testing.T) {
	requirePython(t)
	srv := wheelServer(t, []string{"writer", "engine"}, nil)
	writer := wheelComponent("writer")
	writer.Imports = []manifest.Component{wheelComponent("engine")}
	paths := testPaths(t)
	runner := &fakeRunner{}
	inst := &Installer{Downloader: artifact.HTTPDownloader{Client: srv.Client()}, GithubBaseURL: srv.URL, Runner: runner}

	if err := inst.Install(context.Background(), writer, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	venvPython := pyruntime.VenvPython(pyruntime.VenvDir(paths.RuntimeDir("writer")))
	engineWheel := filepath.Join(paths.DownloadCacheDir(), "engine-1.0.0-py3-none-any.whl")
	installed := false
	for _, c := range runner.calls {
		installed = installed || (c.name == venvPython && c.args[len(c.args)-1] == engineWheel)
	}
	if !installed {
		t.Fatalf("expected the verified engine wheel to be installed into writer's venv, got %+v", runner.calls)
	}
}

func TestInstallGithubReleaseWheelImportedChecksumMismatchFailsClosed(t *testing.T) {
	requirePython(t)
	srv := wheelServer(t, []string{"writer", "engine"}, map[string]bool{"engine": true})
	writer := wheelComponent("writer")
	writer.Imports = []manifest.Component{wheelComponent("engine")}
	paths := testPaths(t)
	runner := &fakeRunner{}
	inst := &Installer{Downloader: artifact.HTTPDownloader{Client: srv.Client()}, GithubBaseURL: srv.URL, Runner: runner}

	err := inst.Install(context.Background(), writer, paths)
	if err == nil || !strings.Contains(err.Error(), "imported by writer") {
		t.Fatalf("expected the corrupt imported wheel to fail the install, got %v", err)
	}
	engineWheel := filepath.Join(paths.DownloadCacheDir(), "engine-1.0.0-py3-none-any.whl")
	for _, c := range runner.calls {
		for _, a := range c.args {
			if a == engineWheel {
				t.Fatalf("pip must never see an unverified wheel, got call %+v", c)
			}
		}
	}
	if _, ok := CurrentBinaryPath(paths, "writer"); ok {
		t.Fatal("a failed install must not activate the component")
	}
}
