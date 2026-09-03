package health

import (
	"context"
	"os/exec"
)

// OSExecer runs commands directly on the host system.
type OSExecer struct{}

// Exec implements Execer.
func (OSExecer) Exec(ctx context.Context, path string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, path, args...).CombinedOutput()
}
