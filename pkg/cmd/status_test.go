package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/status"
)

func TestStatusCommand(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	content := testManifestTOML
	if err := os.WriteFile(manifestFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--manifest", manifestFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected status failure: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "HOWL ECOSYSTEM STATUS") {
		t.Errorf("expected status header, got:\n%s", out)
	}
}

func TestStatusJSONCommand(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	content := testManifestTOML
	if err := os.WriteFile(manifestFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--manifest", manifestFile, "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected status --json failure: %v", err)
	}

	var report status.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("failed to parse status JSON: %v", err)
	}

	if report.EcosystemName != "Howl" {
		t.Errorf("expected Howl ecosystem name, got %s", report.EcosystemName)
	}
}
