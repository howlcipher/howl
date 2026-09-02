package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/plane"
)

func TestProjectValidateEquivalence(t *testing.T) {
	tempDir := t.TempDir()
	fakeBin := filepath.Join(tempDir, "howlplane")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho 'Manifest loaded successfully:'\necho 'Validation passed.'"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	origRunner := plane.DefaultRunner
	mock := &mockPlaneRunner{exitCode: 0}
	plane.DefaultRunner = mock
	defer func() {
		plane.DefaultRunner = origRunner
	}()

	// Run via 'howl project validate'
	rootCmd1 := NewRootCommand()
	var buf1 bytes.Buffer
	rootCmd1.SetOut(&buf1)
	rootCmd1.SetArgs([]string{"project", "validate", "/sample/path"})

	err1 := rootCmd1.Execute()
	if err1 != nil {
		t.Fatalf("unexpected error running 'howl project validate': %v", err1)
	}
	args1 := append([]string{}, mock.lastArgs...)

	// Run via 'howl plane project validate'
	rootCmd2 := NewRootCommand()
	var buf2 bytes.Buffer
	rootCmd2.SetOut(&buf2)
	rootCmd2.SetArgs([]string{"plane", "project", "validate", "/sample/path"})

	err2 := rootCmd2.Execute()
	if err2 != nil {
		t.Fatalf("unexpected error running 'howl plane project validate': %v", err2)
	}
	args2 := append([]string{}, mock.lastArgs...)

	// Verify both invoke howlplane with identical arguments ["project", "validate", "/sample/path"]
	expectedArgs := []string{"project", "validate", "/sample/path"}
	if len(args1) != len(expectedArgs) || args1[0] != expectedArgs[0] || args1[1] != expectedArgs[1] || args1[2] != expectedArgs[2] {
		t.Errorf("howl project validate forwarded args %v, expected %v", args1, expectedArgs)
	}
	if len(args2) != len(expectedArgs) || args2[0] != expectedArgs[0] || args2[1] != expectedArgs[1] || args2[2] != expectedArgs[2] {
		t.Errorf("howl plane project validate forwarded args %v, expected %v", args2, expectedArgs)
	}

	out1 := buf1.String()
	out2 := buf2.String()
	if out1 != out2 {
		t.Errorf("expected identical output between 'howl project validate' and 'howl plane project validate'\nOut1:\n%s\nOut2:\n%s", out1, out2)
	}
}

func TestProjectValidateMissingExecutable(t *testing.T) {
	// Empty PATH so howlplane is not discovered
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOWLPLANE_HOME", "")
	t.Setenv("HOWLPLANE_DIR", "")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"project", "validate", "/some/path"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error when howlplane executable is missing, got nil")
	}
	if !strings.Contains(err.Error(), "howlplane executable not found") {
		t.Errorf("expected missing executable error message, got: %v", err)
	}
}
