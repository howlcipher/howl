package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRollbackWithNoTargetFails(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"rollback", "--manifest", manifestFile, "--yes"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error with no rollback target recorded")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitRollbackFailure {
		t.Errorf("expected ExitRollbackFailure, got %v", err)
	}
}
