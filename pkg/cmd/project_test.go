package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectValidateEquivalence(t *testing.T) {
	tempDir := t.TempDir()
	// Create .git directory so project.DiscoverRoot identifies tempDir as project root
	if err := os.MkdirAll(filepath.Join(tempDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	manifestFile := filepath.Join(tempDir, ".ai-project.toml")
	manifestContent := `schema_version = 1
name = "test-project"
project_type = ["go"]

[commands]
test = ["go", "test"]
`
	if err := os.WriteFile(manifestFile, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run via 'howl project validate'
	rootCmd1 := NewRootCommand()
	var buf1 bytes.Buffer
	rootCmd1.SetOut(&buf1)
	rootCmd1.SetArgs([]string{"project", "validate", tempDir})

	err1 := rootCmd1.Execute()
	if err1 != nil {
		t.Fatalf("unexpected error running 'howl project validate': %v", err1)
	}

	// Run via 'howl plane project validate'
	rootCmd2 := NewRootCommand()
	var buf2 bytes.Buffer
	rootCmd2.SetOut(&buf2)
	rootCmd2.SetArgs([]string{"plane", "project", "validate", tempDir})

	err2 := rootCmd2.Execute()
	if err2 != nil {
		t.Fatalf("unexpected error running 'howl plane project validate': %v", err2)
	}

	// Verify both pass and output contains project validation success
	out1 := buf1.String()
	out2 := buf2.String()

	if !strings.Contains(out1, "Validation passed.") {
		t.Errorf("expected 'Validation passed.' in howl project validate output, got:\n%s", out1)
	}
	if !strings.Contains(out2, "Validation passed.") {
		t.Errorf("expected 'Validation passed.' in howl plane project validate output, got:\n%s", out2)
	}
	if out1 != out2 {
		t.Errorf("expected identical output between 'howl project validate' and 'howl plane project validate'\nOut1:\n%s\nOut2:\n%s", out1, out2)
	}
}

func TestProjectValidateMissingManifest(t *testing.T) {
	tempDir := t.TempDir()
	// Create .git directory so root is discovered, but no .ai-project.toml
	if err := os.MkdirAll(filepath.Join(tempDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"project", "validate", tempDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error for missing .ai-project.toml, got nil")
	}
	if !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "manifest file not found") {
		t.Errorf("expected manifest not found error message, got: %v", err)
	}
}
