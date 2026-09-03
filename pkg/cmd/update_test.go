package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCheckReportsNoUpdate(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlplane", "1.0.0")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"update", "--check", "--manifest", manifestFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("expected up-to-date message, got:\n%s", buf.String())
	}
}

func TestUpdateCheckReportsAvailableUpdate(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlplane", "0.9.0") // older than manifest's 1.0.0

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"update", "--check", "--manifest", manifestFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "0.9.0 -> 1.0.0") {
		t.Errorf("expected version delta in output, got:\n%s", out)
	}
}

func TestUpdateCheckIsReadOnly(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlplane", "0.9.0")

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"update", "--check", "--manifest", manifestFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// State on disk must be untouched by a --check run.
	data, err := os.ReadFile(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "0.9.0") {
		t.Errorf("expected --check to leave installed state untouched, got:\n%s", data)
	}
}
