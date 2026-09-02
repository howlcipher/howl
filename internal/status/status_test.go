package status

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCollectStatus(t *testing.T) {
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

	report, err := Collect(tempDir, manifestPath, "")
	if err != nil {
		t.Fatalf("unexpected error collecting status: %v", err)
	}

	if report.EcosystemName != "Howl" {
		t.Errorf("expected Howl, got %s", report.EcosystemName)
	}

	if len(report.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(report.Components))
	}

	// Test JSON
	var jsonBuf bytes.Buffer
	if err := RenderJSON(&jsonBuf, report); err != nil {
		t.Fatalf("JSON render error: %v", err)
	}
	var parsed Report
	if err := json.Unmarshal(jsonBuf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Test Human
	var humanBuf bytes.Buffer
	RenderHuman(&humanBuf, report)
	if !bytes.Contains(humanBuf.Bytes(), []byte("HOWL ECOSYSTEM STATUS")) {
		t.Errorf("human output missing header")
	}
}
