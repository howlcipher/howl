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
