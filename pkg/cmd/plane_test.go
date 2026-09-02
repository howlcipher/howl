package cmd

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/plane"
)

type mockPlaneRunner struct {
	lastArgs []string
	exitCode int
}

func (m *mockPlaneRunner) Run(ctx context.Context, executable string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	m.lastArgs = args
	io.WriteString(stdout, "mocked execution of "+strings.Join(args, " ")+"\n")
	return m.exitCode, nil
}

func TestPlaneCommandHelp(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"plane", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected plane --help error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "HowlPlane") {
		t.Errorf("expected HowlPlane in plane help output, got:\n%s", out)
	}
	if !strings.Contains(out, "project") {
		t.Errorf("expected project in plane help output, got:\n%s", out)
	}
}

func TestPlaneCommandForwarding(t *testing.T) {
	origRunner := plane.DefaultRunner
	mock := &mockPlaneRunner{exitCode: 0}
	plane.DefaultRunner = mock
	defer func() {
		plane.DefaultRunner = origRunner
	}()

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"plane", "route", "test-objective"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected forwarding error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "mocked execution of route test-objective") {
		t.Errorf("expected mock output in stdout, got:\n%s", out)
	}
	if len(mock.lastArgs) != 2 || mock.lastArgs[0] != "route" || mock.lastArgs[1] != "test-objective" {
		t.Errorf("expected args [route test-objective], got %v", mock.lastArgs)
	}
}
