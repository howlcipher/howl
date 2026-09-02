package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunDiagnostics(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "ecosystem.toml")
	manifestContent := `
[ecosystem]
name = "Howl"
version = "0.1.0"
description = "Test"

[[components]]
name = "howlplane"
repository = "https://github.com/howlcipher/howlplane"
role = "AI engineering control plane"
binary = "howlplane"
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := RunDiagnostics(Options{
		BaseDir:      tempDir,
		ManifestPath: manifestPath,
	})
	if err != nil {
		t.Fatalf("unexpected error running diagnostics: %v", err)
	}

	if len(report.Checks) == 0 {
		t.Errorf("expected diagnostic checks, got 0")
	}

	// Verify JSON rendering
	var jsonBuf bytes.Buffer
	if err := RenderJSON(&jsonBuf, report); err != nil {
		t.Fatalf("failed to render JSON: %v", err)
	}

	var parsed DiagnosticReport
	if err := json.Unmarshal(jsonBuf.Bytes(), &parsed); err != nil {
		t.Fatalf("rendered invalid JSON: %v", err)
	}

	// Verify Human rendering
	var humanBuf bytes.Buffer
	RenderHuman(&humanBuf, report, true)
	if !bytes.Contains(humanBuf.Bytes(), []byte("HOWL ECOSYSTEM DOCTOR")) {
		t.Errorf("expected header in human output")
	}
}

func TestInvalidManifestDoctor(t *testing.T) {
	tempDir := t.TempDir()
	invalidManifest := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(invalidManifest, []byte("invalid toml [["), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := RunDiagnostics(Options{
		BaseDir:      tempDir,
		ManifestPath: invalidManifest,
	})
	if err != nil {
		t.Fatalf("expected report even with invalid manifest, got error: %v", err)
	}
	if report.Summary != HealthFailed {
		t.Errorf("expected FAILED summary for invalid manifest, got %s", report.Summary)
	}
}

func TestSummaryCalculation(t *testing.T) {
	checksHealthy := []CheckResult{
		{Status: StatusPass},
		{Status: StatusPass},
	}
	if computeSummary(checksHealthy, false) != HealthHealthy {
		t.Errorf("expected HEALTHY")
	}

	checksUnknown := []CheckResult{
		{Status: StatusPass},
		{Status: StatusUnknown},
	}
	if computeSummary(checksUnknown, false) != HealthHealthy {
		t.Errorf("expected HEALTHY for unknown optional components in non-strict mode")
	}
	if computeSummary(checksUnknown, true) != HealthHealthyWithWarnings {
		t.Errorf("expected HEALTHY WITH WARNINGS for unknown optional components in strict mode")
	}

	checksWarn := []CheckResult{
		{Status: StatusPass},
		{Status: StatusWarn},
	}
	if computeSummary(checksWarn, false) != HealthHealthyWithWarnings {
		t.Errorf("expected HEALTHY WITH WARNINGS")
	}
	if computeSummary(checksWarn, true) != HealthDegraded {
		t.Errorf("expected DEGRADED under strict mode")
	}

	checksFail := []CheckResult{
		{Status: StatusPass},
		{Status: StatusFail},
	}
	if computeSummary(checksFail, false) != HealthFailed {
		t.Errorf("expected FAILED")
	}
}
