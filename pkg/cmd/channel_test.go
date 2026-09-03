package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestChannelDefaultsToStable(t *testing.T) {
	sandboxHowlPaths(t)

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"channel"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "stable" {
		t.Errorf("expected default channel 'stable', got %q", buf.String())
	}
}

func TestChannelSetAndGet(t *testing.T) {
	sandboxHowlPaths(t)

	setCmd := NewRootCommand()
	setCmd.SetOut(&bytes.Buffer{})
	setCmd.SetArgs([]string{"channel", "beta"})
	if err := setCmd.Execute(); err != nil {
		t.Fatalf("unexpected error setting channel: %v", err)
	}

	getCmd := NewRootCommand()
	var buf bytes.Buffer
	getCmd.SetOut(&buf)
	getCmd.SetArgs([]string{"channel"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("unexpected error getting channel: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "beta" {
		t.Errorf("expected channel 'beta' after set, got %q", buf.String())
	}
}

func TestChannelInvalidRejected(t *testing.T) {
	sandboxHowlPaths(t)

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"channel", "nightly-experimental"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid channel name")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitValidationFailure {
		t.Errorf("expected ExitValidationFailure, got %v", err)
	}
}
