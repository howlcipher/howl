package pyruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// An editable (developer profile) install links a runtime to a source
// checkout, but pip resolves the checkout's dependencies only when the
// install runs. When the checkout later declares a new dependency, the
// runtime silently lacks it, and pip's own metadata (written at install
// time) still says nothing is missing. Dogfood DOG-031: HowlWriter gained a
// hard dependency on howl-provider-core, its runtime never received it, and
// `howl doctor` reported the ecosystem HEALTHY while `howlwriter native
// write` failed with ModuleNotFoundError.
//
// Howl therefore records, at install time, a hash of each file that
// declares the checkout's dependencies, and the health check compares it
// with the checkout as it is now.

// SourceDepsMarker is the file, in a component's runtime root, recording
// what an editable install's dependency declarations looked like.
const SourceDepsMarker = "source-dependencies.json"

// dependencyFiles are the files pip reads a project's dependencies from.
var dependencyFiles = []string{"pyproject.toml", "setup.cfg", "setup.py", "requirements.txt"}

type sourceDeps struct {
	PackageRoot string            `json:"package_root"`
	Files       map[string]string `json:"files"`
}

func hashDependencyFiles(packageRoot string) (map[string]string, error) {
	files := map[string]string{}
	for _, name := range dependencyFiles {
		data, err := os.ReadFile(filepath.Join(packageRoot, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filepath.Join(packageRoot, name), err)
		}
		sum := sha256.Sum256(data)
		files[name] = hex.EncodeToString(sum[:])
	}
	return files, nil
}

// RecordSourceDependencies writes the marker for an editable install of the
// project at packageRoot into runtimeRoot. Call it after a successful
// install.
func RecordSourceDependencies(runtimeRoot, packageRoot string) error {
	files, err := hashDependencyFiles(packageRoot)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(sourceDeps{PackageRoot: packageRoot, Files: files}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runtimeRoot, SourceDepsMarker), data, 0o644); err != nil {
		return fmt.Errorf("failed to record dependency declarations: %w", err)
	}
	return nil
}

// editableCheckouts returns the source directories of every editable
// install in venvDir, read from pip's PEP 610 direct_url.json records.
func editableCheckouts(venvDir string) []string {
	pattern := filepath.Join(venvDir, "lib", "python*", "site-packages", "*.dist-info", "direct_url.json")
	if runtime.GOOS == "windows" {
		pattern = filepath.Join(venvDir, "Lib", "site-packages", "*.dist-info", "direct_url.json")
	}
	records, _ := filepath.Glob(pattern)
	var checkouts []string
	for _, record := range records {
		data, err := os.ReadFile(record)
		if err != nil {
			continue
		}
		var direct struct {
			URL     string `json:"url"`
			DirInfo struct {
				Editable bool `json:"editable"`
			} `json:"dir_info"`
		}
		if json.Unmarshal(data, &direct) != nil || !direct.DirInfo.Editable {
			continue
		}
		if parsed, err := url.Parse(direct.URL); err == nil && parsed.Scheme == "file" {
			checkouts = append(checkouts, filepath.FromSlash(parsed.Path))
		}
	}
	sort.Strings(checkouts)
	return checkouts
}

// CheckSourceDependencies reports an editable runtime whose checkout's
// dependency declarations changed after it was installed, or that was
// installed before Howl recorded them. Runtimes without an editable
// install (wheel installs, no runtime at all) are not its concern.
func CheckSourceDependencies(runtimeRoot string) error {
	checkouts := editableCheckouts(VenvDir(runtimeRoot))
	if len(checkouts) == 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(runtimeRoot, SourceDepsMarker))
	if os.IsNotExist(err) {
		return fmt.Errorf("editable install from %s has unverified dependencies (installed before Howl recorded them); "+
			"run `howl doctor --fix` to resync the runtime with the checkout", strings.Join(checkouts, ", "))
	}
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", SourceDepsMarker, err)
	}
	var recorded sourceDeps
	if err := json.Unmarshal(data, &recorded); err != nil {
		return fmt.Errorf("%s is unreadable (%v); run `howl doctor --fix` to resync the runtime", SourceDepsMarker, err)
	}
	current, err := hashDependencyFiles(recorded.PackageRoot)
	if err != nil {
		return err
	}
	var changed []string
	for _, name := range dependencyFiles {
		if current[name] != recorded.Files[name] {
			changed = append(changed, name)
		}
	}
	if len(changed) > 0 {
		return fmt.Errorf("dependency declarations in %s changed since install (%s), so the runtime may lack what the "+
			"checkout now requires; run `howl doctor --fix` to resync it", recorded.PackageRoot, strings.Join(changed, ", "))
	}
	return nil
}

// InstalledCheckout returns the checkout an existing editable runtime was
// installed from, so a repair can reinstall from the same source without the
// operator re-supplying its location. packageDir is the manifest's package
// directory within the checkout; ok is false when the runtime has no
// editable install of such a checkout on disk.
func InstalledCheckout(runtimeRoot, packageDir string) (string, bool) {
	for _, packageRoot := range editableCheckouts(VenvDir(runtimeRoot)) {
		root := filepath.Clean(packageRoot)
		if rel := filepath.Clean(packageDir); rel != "." && rel != "" {
			if filepath.Base(root) != filepath.Base(rel) || !strings.HasSuffix(root, string(filepath.Separator)+rel) {
				continue
			}
			root = strings.TrimSuffix(root, string(filepath.Separator)+rel)
		}
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			return root, true
		}
	}
	return "", false
}
