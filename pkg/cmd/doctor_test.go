package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/doctor"
)

func TestDoctorCommand(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	content := `
[ecosystem]
name = "Howl"
version = "0.1.0"
description = "Test"

[[components]]
name = "howlplane"
repository = "https://github.com/howlcipher/howlplane"
role = "Control plane"
binary = "howlplane"
`
	if err := os.WriteFile(manifestFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"doctor", "--manifest", manifestFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected doctor failure: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "HOWL ECOSYSTEM DOCTOR") {
		t.Errorf("expected doctor header, got:\n%s", out)
	}
	if !strings.Contains(out, "ECOSYSTEM STATUS:") {
		t.Errorf("expected summary status line, got:\n%s", out)
	}
}

func TestDoctorJSONCommand(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	content := `
[ecosystem]
name = "Howl"
version = "0.1.0"
description = "Test"

[[components]]
name = "howlplane"
repository = "https://github.com/howlcipher/howlplane"
role = "Control plane"
binary = "howlplane"
`
	if err := os.WriteFile(manifestFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"doctor", "--manifest", manifestFile, "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected doctor --json failure: %v", err)
	}

	var report doctor.DiagnosticReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("failed to parse doctor JSON: %v", err)
	}

	if len(report.Checks) == 0 {
		t.Errorf("expected check results in JSON report")
	}
}
