package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreativeForwardsArgsStreamsAndExit(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "howlplane")
	script := "#!/bin/sh\nprintf 'args:%s\\n' \"$*\"\ncat\nprintf 'creative-error\\n' >&2\nexit 5\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := NewRootCommand()
	command.SetArgs([]string{"creative", "run", "--run-dir", "/tmp/run", "--json"})
	command.SetIn(strings.NewReader("creative-input\n"))
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	err := command.Execute()
	if err == nil {
		t.Fatal("expected forwarded exit")
	}
	exit, ok := err.(ExitCoder)
	if !ok || exit.ExitCode() != 5 {
		t.Fatalf("wrong exit: %v", err)
	}
	if !strings.Contains(stdout.String(), "args:creative run --run-dir /tmp/run --json") || !strings.Contains(stdout.String(), "creative-input") {
		t.Fatalf("stdout/argv/stdin not forwarded: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "creative-error") {
		t.Fatalf("stderr not forwarded: %q", stderr.String())
	}
}

func TestCreativeResumeForwardsArgsAndExit(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "howlplane")
	script := "#!/bin/sh\nprintf 'args:%s\\n' \"$*\"\nexit 0\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := NewRootCommand()
	command.SetArgs([]string{"creative", "resume", "--run-dir", "/tmp/run", "--json"})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	err := command.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "args:creative resume --run-dir /tmp/run --json") {
		t.Fatalf("arguments not forwarded: %q", stdout.String())
	}
}

func TestCreativeMissingHowlPlaneReportsHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	command := NewRootCommand()
	command.SetArgs([]string{"creative", "run", "--json"})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "howl install howlplane") {
		t.Fatalf("expected install hint, got %v", err)
	}
}
