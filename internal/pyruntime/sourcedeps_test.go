package pyruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeEditableRuntime lays out a runtime whose venv holds pip's PEP 610
// record for an editable install of checkout (or a non-editable one).
func fakeEditableRuntime(t *testing.T, checkout string, editable bool) string {
	t.Helper()
	runtimeRoot := t.TempDir()
	site := filepath.Join(VenvDir(runtimeRoot), "lib", "python3.14", "site-packages")
	if runtime.GOOS == "windows" {
		site = filepath.Join(VenvDir(runtimeRoot), "Lib", "site-packages")
	}
	distInfo := filepath.Join(site, "howlwriter-0.1.0.dist-info")
	if err := os.MkdirAll(distInfo, 0o755); err != nil {
		t.Fatal(err)
	}
	record := `{"dir_info": {"editable": ` + map[bool]string{true: "true", false: "false"}[editable] +
		`}, "url": "file://` + filepath.ToSlash(checkout) + `"}`
	if err := os.WriteFile(filepath.Join(distInfo, "direct_url.json"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	return runtimeRoot
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceDependenciesIgnoreRuntimesWithoutEditableInstalls(t *testing.T) {
	if err := CheckSourceDependencies(t.TempDir()); err != nil {
		t.Fatalf("no runtime: %v", err)
	}
	if err := CheckSourceDependencies(fakeEditableRuntime(t, t.TempDir(), false)); err != nil {
		t.Fatalf("non-editable install: %v", err)
	}
}

func TestSourceDependenciesUnrecordedEditableInstallNeedsResync(t *testing.T) {
	checkout := t.TempDir()
	err := CheckSourceDependencies(fakeEditableRuntime(t, checkout, true))
	if err == nil || !strings.Contains(err.Error(), "unverified dependencies") || !strings.Contains(err.Error(), "howl doctor --fix") {
		t.Fatalf("expected an actionable unverified error, got %v", err)
	}
}

func TestSourceDependenciesDetectEveryDeclarationChange(t *testing.T) {
	cases := map[string]func(checkout string){
		"pyproject.toml": func(c string) {
			writeFile(t, filepath.Join(c, "pyproject.toml"), "[project]\ndependencies = [\"PyYAML\", \"howl-provider-core\"]\n")
		},
		"setup.cfg": func(c string) {
			writeFile(t, filepath.Join(c, "setup.cfg"), "[options]\ninstall_requires = requests\n")
		},
		"requirements.txt": func(c string) { writeFile(t, filepath.Join(c, "requirements.txt"), "httpx\n") },
	}
	for changed, change := range cases {
		t.Run(changed, func(t *testing.T) {
			checkout := t.TempDir()
			writeFile(t, filepath.Join(checkout, "pyproject.toml"), "[project]\ndependencies = [\"PyYAML\"]\n")
			runtimeRoot := fakeEditableRuntime(t, checkout, true)
			if err := RecordSourceDependencies(runtimeRoot, checkout); err != nil {
				t.Fatal(err)
			}
			if err := CheckSourceDependencies(runtimeRoot); err != nil {
				t.Fatalf("freshly recorded install: %v", err)
			}

			change(checkout)

			err := CheckSourceDependencies(runtimeRoot)
			if err == nil || !strings.Contains(err.Error(), changed) || !strings.Contains(err.Error(), "howl doctor --fix") {
				t.Fatalf("expected drift naming %s, got %v", changed, err)
			}
			if err := RecordSourceDependencies(runtimeRoot, checkout); err != nil {
				t.Fatal(err)
			}
			if err := CheckSourceDependencies(runtimeRoot); err != nil {
				t.Fatalf("after resync: %v", err)
			}
		})
	}
}

func TestInstalledCheckoutRecoversTheSourceOfAnEditableRuntime(t *testing.T) {
	checkout := t.TempDir()
	if got, ok := InstalledCheckout(fakeEditableRuntime(t, checkout, true), "."); !ok || got != filepath.Clean(checkout) {
		t.Fatalf("package at checkout root: got %q, %v", got, ok)
	}

	pkg := filepath.Join(checkout, "python")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := InstalledCheckout(fakeEditableRuntime(t, pkg, true), "python"); !ok || got != filepath.Clean(checkout) {
		t.Fatalf("package in a subdirectory: got %q, %v", got, ok)
	}

	if _, ok := InstalledCheckout(fakeEditableRuntime(t, filepath.Join(checkout, "gone"), true), "."); ok {
		t.Fatal("a checkout that no longer exists must not be reused")
	}
	if _, ok := InstalledCheckout(fakeEditableRuntime(t, checkout, false), "."); ok {
		t.Fatal("a non-editable install records no checkout to reuse")
	}
}
