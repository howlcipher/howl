// Package plane provides a structured integration adapter for HowlPlane.
package plane

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Runner abstracts command execution for HowlPlane subprocess forwarding.
type Runner interface {
	Run(ctx context.Context, executable string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error)
}

// OSExecRunner executes commands directly on the host system.
type OSExecRunner struct{}

// Run executes the command using os/exec with full stream and exit-code fidelity.
func (r *OSExecRunner) Run(ctx context.Context, executable string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	if executable == "" {
		return 1, fmt.Errorf("howlplane executable not found or not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = os.Environ()

	err := cmd.Run()
	if err == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), nil
		}
		return exitErr.ExitCode(), nil
	}

	return 1, err
}

// DefaultRunner is the default system runner instance.
var DefaultRunner Runner = &OSExecRunner{}
