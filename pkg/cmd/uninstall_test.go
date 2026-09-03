package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallNothingInstalledIsNoop(t *testing.T) {
	sandboxHowlPaths(t)

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--yes"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Nothing is installed")) {
		t.Errorf("expected no-op message, got:\n%s", buf.String())
	}
}

func TestUninstallComponentRemovesOnlyHowlOwnedPaths(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlframe", "0.1.1")

	compDir := paths.ComponentDir("howlframe")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(compDir, "releases", "0.1.1", "howlframe")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	// A sibling directory that Howl does NOT own must never be touched.
	sibling := filepath.Join(filepath.Dir(paths.DataHome), "not-howl-owned")
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"uninstall", "howlframe", "--yes"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(compDir); !os.IsNotExist(err) {
		t.Errorf("expected component directory to be removed")
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Errorf("expected unrelated sibling directory to be left alone: %v", err)
	}

	data, err := os.ReadFile(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("howlframe")) {
		t.Errorf("expected howlframe removed from state, got:\n%s", data)
	}
}

func TestUninstallCancelledWithoutYes(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlframe", "0.1.1")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetIn(strings.NewReader("n\n"))
	rootCmd.SetArgs([]string{"uninstall"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitCancelled {
		t.Errorf("expected ExitCancelled, got %v", err)
	}
}
