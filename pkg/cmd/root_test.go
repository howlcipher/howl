package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error running root --help: %v", err)
	}

	out := buf.String()
	expectedSubcommands := []string{
		"version",
		"doctor",
		"status",
	}

	for _, sub := range expectedSubcommands {
		if !strings.Contains(out, sub) {
			t.Errorf("expected root help to list subcommand %q, got output:\n%s", sub, out)
		}
	}
}
