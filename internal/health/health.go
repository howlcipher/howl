// Package health runs a component's declared health check to verify an
// installation actually succeeded. It implements engine.HealthChecker.
// Checks are deliberately shallow: Howl needs only enough signal to know
// whether install/update succeeded, not application-level monitoring.
package health

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/howlcipher/howl/internal/component"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/pyruntime"
)

// Execer runs a command and returns its combined output. Injected for
// testability.
type Execer interface {
	Exec(ctx context.Context, path string, args []string) ([]byte, error)
}

// Checker implements engine.HealthChecker.
type Checker struct {
	Exec Execer
}

// Check runs c's declared health check against its currently activated
// release.
func (h Checker) Check(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	switch c.HealthCheck.Type {
	case manifest.HealthBinaryExists:
		return checkBinaryExists(paths, c.Name)
	case manifest.HealthExecVersion:
		return h.checkExecVersion(ctx, paths, c)
	case manifest.HealthPythonImport:
		return h.checkPythonImport(ctx, paths, c)
	default:
		return fmt.Errorf("component %q declares unsupported health_check type %q", c.Name, c.HealthCheck.Type)
	}
}

func checkBinaryExists(paths platform.Paths, name string) error {
	binPath, ok := component.CurrentBinaryPath(paths, name)
	if !ok {
		return fmt.Errorf("no activated release found for %s", name)
	}
	info, err := os.Stat(binPath)
	if err != nil {
		return fmt.Errorf("expected executable %s not found: %w", binPath, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%s exists but is not executable", binPath)
	}
	return nil
}

func (h Checker) checkExecVersion(ctx context.Context, paths platform.Paths, c manifest.Component) error {
	binPath, ok := component.CurrentBinaryPath(paths, c.Name)
	if !ok {
		return fmt.Errorf("no activated release found for %s", c.Name)
	}
	args := c.HealthCheck.Args
	if len(args) == 0 {
		args = []string{"--version"}
	}
	out, err := h.Exec.Exec(ctx, binPath, args)
	if err != nil {
		return fmt.Errorf("%s %v failed: %w", binPath, args, err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return fmt.Errorf("%s %v produced no output", binPath, args)
	}
	return nil
}

func (h Checker) checkPythonImport(ctx context.Context, paths platform.Paths, c manifest.Component) error {
	venvPython := pyruntime.VenvPython(pyruntime.VenvDir(paths.RuntimeDir(c.Name)))
	if _, err := os.Stat(venvPython); err != nil {
		return fmt.Errorf("python runtime for %s not found: %w", c.Name, err)
	}
	_, err := h.Exec.Exec(ctx, venvPython, []string{"-c", "import " + c.HealthCheck.Module})
	if err != nil {
		return fmt.Errorf("failed to import %s in %s's runtime: %w", c.HealthCheck.Module, c.Name, err)
	}
	return nil
}
