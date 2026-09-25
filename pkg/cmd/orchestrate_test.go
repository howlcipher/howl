package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrchestrateForwardsArgsStreamsAndExit(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "howlplane")
	script := "#!/bin/sh\nprintf 'args:%s\\n' \"$*\"\ncat\nprintf 'error\\n' >&2\nexit 7\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := NewRootCommand()
	command.SetArgs([]string{"orchestrate", "inspect", "--json"})
	command.SetIn(strings.NewReader("input\n"))
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	err := command.Execute()
	if err == nil {
		t.Fatal("expected forwarded exit")
	}
	exit, ok := err.(ExitCoder)
	if !ok || exit.ExitCode() != 7 {
		t.Fatalf("wrong exit: %v", err)
	}
	if !strings.Contains(stdout.String(), "args:orchestrate inspect --json") || !strings.Contains(stdout.String(), "input") {
		t.Fatalf("stdout/argv/stdin not forwarded: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error") {
		t.Fatalf("stderr not forwarded: %q", stderr.String())
	}
}

func runForwarded(t *testing.T, args ...string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf 'args:%s\\n' \"$*\"\nexit 3\n"
	if err := os.WriteFile(filepath.Join(dir, "howlplane"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	command := NewRootCommand()
	command.SetArgs(args)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	err := command.Execute()
	return stdout.String(), err
}

func TestAgentsDoctorAndFactoryPrepareForwardVerbatim(t *testing.T) {
	cases := map[string][]string{
		"args:agents doctor":                         {"agents", "doctor"},
		"args:agents doctor --repo /r --live --json": {"agents", "doctor", "--repo", "/r", "--live", "--json"},
		"args:factory prepare --repo /r --yes":       {"factory", "prepare", "--repo", "/r", "--yes"},
		"args:factory prepare --revoke":              {"factory", "prepare", "--revoke"},
	}
	for want, args := range cases {
		out, err := runForwarded(t, args...)
		if !strings.Contains(out, want) {
			t.Fatalf("%v forwarded as %q, want %q", args, out, want)
		}
		exit, ok := err.(ExitCoder)
		if !ok || exit.ExitCode() != 3 {
			t.Fatalf("%v: exit status not forwarded: %v", args, err)
		}
	}
}

func TestOnlyFixedHowlPlanePathsAreForwarded(t *testing.T) {
	for _, args := range [][]string{{"agents", "anything"}, {"factory", "start"}} {
		out, err := runForwarded(t, args...)
		if strings.Contains(out, "args:") {
			t.Fatalf("%v must not reach howlplane, got %q (err %v)", args, out, err)
		}
	}
}

func TestForwardReportsMissingHowlPlane(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	command := NewRootCommand()
	command.SetArgs([]string{"agents", "doctor"})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "howl install howlplane") {
		t.Fatalf("expected install hint, got %v", err)
	}
}
