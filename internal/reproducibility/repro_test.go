package reproducibility

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/doctor"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plane"
	"github.com/howlcipher/howl/internal/status"
	"github.com/howlcipher/howl/pkg/cmd"
)

// 1. Regression Test: Ensure no replace directive exists in go.mod
func TestNoCommittedRelativeModuleReplace(t *testing.T) {
	manifestPath, err := manifest.FindManifestPath(".")
	if err != nil {
		t.Fatalf("could not locate project root: %v", err)
	}
	repoRoot := filepath.Dir(manifestPath)
	goModPath := filepath.Join(repoRoot, "go.mod")

	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed to read go.mod at %s: %v", goModPath, err)
	}

	content := string(data)
	if strings.Contains(content, "replace ") {
		t.Errorf("go.mod contains a forbidden 'replace' directive:\n%s", content)
	}
	if strings.Contains(content, "github.com/howlcipher/howlplane") {
		t.Errorf("go.mod contains unpublished dependency github.com/howlcipher/howlplane:\n%s", content)
	}
}

// 2. Regression Test: Root CLI builds cleanly in an isolated environment without sibling repositories
func TestRootCLIBuildsWithoutSiblingRepos(t *testing.T) {
	manifestPath, err := manifest.FindManifestPath(".")
	if err != nil {
		t.Fatalf("could not locate project root: %v", err)
	}
	repoRoot := filepath.Dir(manifestPath)

	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "howl_test_bin")

	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/howl")
	buildCmd.Dir = repoRoot
	buildCmd.Env = append(os.Environ(), "HOWLPLANE_HOME=", "HOWLPLANE_DIR=")

	out, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build howl binary without siblings: %v\nOutput: %s", err, string(out))
	}

	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("built binary not found at %s", binPath)
	}
}

func clearEnv(t *testing.T) {
	keys := []string{
		"HOWLPLANE_HOME", "HOWLPLANE_DIR", "HOWL_HOWLPLANE_HOME", "HOWL_HOWLPLANE_DIR",
		"HOWLFRAME_HOME", "HOWLFRAME_DIR", "HOWL_HOWLFRAME_HOME", "HOWL_HOWLFRAME_DIR",
		"HOWLCHANGEOPS_HOME", "HOWLCHANGEOPS_DIR", "HOWL_HOWLCHANGEOPS_HOME", "HOWL_HOWLCHANGEOPS_DIR",
		"HOWLWRITER_HOME", "HOWLWRITER_DIR", "HOWL_HOWLWRITER_HOME", "HOWL_HOWLWRITER_DIR",
		"HOWLBOARD_HOME", "HOWLBOARD_DIR", "HOWL_HOWLBOARD_HOME", "HOWL_HOWLBOARD_DIR",
		"HOWLNOTES_HOME", "HOWLNOTES_DIR", "HOWL_HOWLNOTES_HOME", "HOWL_HOWLNOTES_DIR",
		"HOWL_COMPONENTS_DIR",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

// 3. Regression Test: Missing HowlPlane reports truthful status and diagnostics
func TestMissingHowlPlaneTruthfulStatus(t *testing.T) {
	clearEnv(t)

	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	manifestContent := `
[ecosystem]
name = "Howl"
version = "0.1.1"
description = "Test"

[[components]]
name = "howlplane"
repository = "https://github.com/howlcipher/howlplane"
role = "Control Plane"
binary = "howlplane"
optional = true
`
	if err := os.WriteFile(manifestFile, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	rep, err := status.Collect(tempDir, manifestFile, "")
	if err != nil {
		t.Fatalf("status collection failed: %v", err)
	}

	if len(rep.Components) != 1 {
		t.Fatalf("expected 1 component status, got %d", len(rep.Components))
	}
	if rep.Components[0].Available || rep.Components[0].Runnable {
		t.Errorf("expected missing howlplane to not be available/runnable, got: %+v", rep.Components[0])
	}

	docRep, err := doctor.RunDiagnostics(doctor.Options{
		BaseDir:      tempDir,
		ManifestPath: manifestFile,
	})
	if err != nil {
		t.Fatalf("doctor execution failed: %v", err)
	}

	// In standalone mode, missing optional component should be UNKNOWN
	foundPlaneCheck := false
	for _, c := range docRep.Checks {
		if c.Name == "howlplane" {
			foundPlaneCheck = true
			if c.Status != doctor.StatusUnknown {
				t.Errorf("expected howlplane status UNKNOWN, got %s", c.Status)
			}
		}
	}
	if !foundPlaneCheck {
		t.Errorf("howlplane check not found in doctor report")
	}
}

// 4. Regression Test: Installed HowlPlane adapter works with argument forwarding
func TestInstalledHowlPlaneAdapterWorks(t *testing.T) {
	tempDir := t.TempDir()
	fakeBin := filepath.Join(tempDir, "howlplane")
	script := `#!/bin/sh
echo "howlplane args: $@"
`
	if err := os.WriteFile(fakeBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	runner := &plane.OSExecRunner{}
	var stdout, stderr bytes.Buffer

	code, err := runner.Run(nil, fakeBin, []string{"route", "task-alpha", "--verbose"}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	expected := "howlplane args: route task-alpha --verbose\n"
	if stdout.String() != expected {
		t.Errorf("expected stdout %q, got %q", expected, stdout.String())
	}
}

type mockAdapterRunner struct {
	lastArgs []string
	exitCode int
}

func (m *mockAdapterRunner) Run(ctx context.Context, executable string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	m.lastArgs = args
	io.WriteString(stdout, "mock validation of "+strings.Join(args, " ")+"\n")
	return m.exitCode, nil
}

// 5. Regression Test: Project validate delegates to HowlPlane
func TestProjectValidateDelegation(t *testing.T) {
	clearEnv(t)
	tempDir := t.TempDir()
	fakeBin := filepath.Join(tempDir, "howlplane")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho ok"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	origRunner := plane.DefaultRunner
	mock := &mockAdapterRunner{exitCode: 0}
	plane.DefaultRunner = mock
	defer func() {
		plane.DefaultRunner = origRunner
	}()

	rootCmd := cmd.NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"project", "validate", "/sample/path"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error executing project validate: %v", err)
	}

	expectedArgs := []string{"project", "validate", "/sample/path"}
	if len(mock.lastArgs) != len(expectedArgs) || mock.lastArgs[0] != expectedArgs[0] || mock.lastArgs[1] != expectedArgs[1] || mock.lastArgs[2] != expectedArgs[2] {
		t.Errorf("expected args %v, got %v", expectedArgs, mock.lastArgs)
	}
}

// 6. Regression Test: Exit codes remain preserved
func TestExitCodesPreserved(t *testing.T) {
	tempDir := t.TempDir()
	fakeBin := filepath.Join(tempDir, "howlplane")
	script := `#!/bin/sh
echo "failing with code 17" >&2
exit 17
`
	if err := os.WriteFile(fakeBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	runner := &plane.OSExecRunner{}
	var stdout, stderr bytes.Buffer

	code, err := runner.Run(nil, fakeBin, []string{"validate"}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}
	if code != 17 {
		t.Errorf("expected exit code 17, got %d", code)
	}
	if !strings.Contains(stderr.String(), "failing with code 17") {
		t.Errorf("expected stderr to contain error output, got %q", stderr.String())
	}
}

// 7. Regression Test: Doctor does not false-PASS an unavailable contract
func TestDoctorDoesNotFalsePassUnavailableContract(t *testing.T) {
	clearEnv(t)
	// Standard clean system PATH containing git and go, but no custom howlplane binaries
	t.Setenv("PATH", "/usr/bin:/bin")

	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	manifestContent := `
[ecosystem]
name = "Howl"
version = "0.1.1"
description = "Test"

[[components]]
name = "howlplane"
repository = "https://github.com/howlcipher/howlplane"
role = "Control Plane"
binary = "howlplane"
optional = true
`
	if err := os.WriteFile(manifestFile, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	docRep, err := doctor.RunDiagnostics(doctor.Options{
		BaseDir:      tempDir,
		ManifestPath: manifestFile,
	})
	if err != nil {
		t.Fatalf("doctor execution failed: %v", err)
	}

	for _, c := range docRep.Contracts {
		if c.Status == "PASS" || c.Status == "KNOWN" {
			t.Errorf("contract %s false-passed despite missing components: %+v", c.ID, c)
		}
	}

	for _, check := range docRep.Checks {
		if check.Category == "Contracts" {
			if check.Status == doctor.StatusPass {
				t.Errorf("contract check reported PASS when components were missing: %+v", check)
			}
			if check.Status != doctor.StatusUnknown {
				t.Errorf("expected contract check status UNKNOWN, got %s", check.Status)
			}
		}
	}
}

// 8. Regression Test: JSON output retains valid schema across commands
func TestJSONOutputSemantics(t *testing.T) {
	// Doctor JSON
	rootCmd := cmd.NewRootCommand()
	var docBuf bytes.Buffer
	rootCmd.SetOut(&docBuf)
	rootCmd.SetArgs([]string{"doctor", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(docBuf.Bytes(), &docMap); err != nil {
		t.Fatalf("doctor output is not valid JSON: %v", err)
	}
	for _, field := range []string{"timestamp", "summary", "checks", "discovered_components", "contracts"} {
		if _, ok := docMap[field]; !ok {
			t.Errorf("doctor JSON missing expected field %q", field)
		}
	}

	// Status JSON
	rootCmd = cmd.NewRootCommand()
	var statBuf bytes.Buffer
	rootCmd.SetOut(&statBuf)
	rootCmd.SetArgs([]string{"status", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var statMap map[string]interface{}
	if err := json.Unmarshal(statBuf.Bytes(), &statMap); err != nil {
		t.Fatalf("status output is not valid JSON: %v", err)
	}
	if _, ok := statMap["ecosystem_name"]; !ok {
		t.Errorf("status JSON missing ecosystem_name")
	}
	if _, ok := statMap["components"]; !ok {
		t.Errorf("status JSON missing components")
	}

	// Version JSON
	rootCmd = cmd.NewRootCommand()
	var verBuf bytes.Buffer
	rootCmd.SetOut(&verBuf)
	rootCmd.SetArgs([]string{"version", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version --json failed: %v", err)
	}

	var verMap map[string]interface{}
	if err := json.Unmarshal(verBuf.Bytes(), &verMap); err != nil {
		t.Fatalf("version output is not valid JSON: %v", err)
	}
	if _, ok := verMap["version"]; !ok {
		t.Errorf("version JSON missing version")
	}

	// Graph JSON
	rootCmd = cmd.NewRootCommand()
	var graphBuf bytes.Buffer
	rootCmd.SetOut(&graphBuf)
	rootCmd.SetArgs([]string{"graph", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("graph --json failed: %v", err)
	}

	var graphMap map[string]interface{}
	if err := json.Unmarshal(graphBuf.Bytes(), &graphMap); err != nil {
		t.Fatalf("graph output is not valid JSON: %v", err)
	}
	if _, ok := graphMap["nodes"]; !ok {
		t.Errorf("graph JSON missing nodes")
	}
	if _, ok := graphMap["edges"]; !ok {
		t.Errorf("graph JSON missing edges")
	}
}

// 9. Regression Test: No diagnostic command mutates any file in workspace or siblings
func TestNoDiagnosticMutatesFiles(t *testing.T) {
	tempRoot := t.TempDir()
	howlDir := filepath.Join(tempRoot, "howl")
	siblingDir := filepath.Join(tempRoot, "howlplane")

	if err := os.MkdirAll(howlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifestFile := filepath.Join(howlDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte("[ecosystem]\nname=\"Howl\"\nversion=\"0.1.1\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	siblingFile := filepath.Join(siblingDir, "sample.txt")
	if err := os.WriteFile(siblingFile, []byte("original sibling content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Compute snapshot before
	snapshotBefore := computeDirHash(t, tempRoot)

	// Run diagnostics
	rootCmd := cmd.NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"doctor", "--manifest", manifestFile})
	_ = rootCmd.Execute()

	rootCmd = cmd.NewRootCommand()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--manifest", manifestFile})
	_ = rootCmd.Execute()

	rootCmd = cmd.NewRootCommand()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"graph"})
	_ = rootCmd.Execute()

	rootCmd = cmd.NewRootCommand()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"version"})
	_ = rootCmd.Execute()

	// Compute snapshot after
	snapshotAfter := computeDirHash(t, tempRoot)

	if snapshotBefore != snapshotAfter {
		t.Errorf("diagnostic execution mutated workspace files! Before hash: %s, After hash: %s", snapshotBefore, snapshotAfter)
	}
}

func computeDirHash(t *testing.T, dir string) string {
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		fmt.Fprintf(h, "path:%s|isDir:%v|size:%d|", rel, info.IsDir(), info.Size())
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			h.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to compute directory hash: %v", err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
