// Package pyruntime provisions isolated Python virtual environments for
// Python-based ecosystem components. It never touches the user's global
// Python environment: every install happens inside a venv Howl creates
// and owns under its own runtimes directory.
package pyruntime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/howlcipher/howl/internal/manifest"
)

// Runner executes a command and returns its combined output. Injected so
// provisioning logic is testable without actually creating a venv or
// invoking pip.
type Runner interface {
	Run(ctx context.Context, dir, name string, args []string) ([]byte, error)
}

// ExecRunner is the real Runner used outside tests.
type ExecRunner struct{}

// Run implements Runner.
func (ExecRunner) Run(ctx context.Context, dir, name string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// DetectPython finds a system python3 interpreter satisfying minVersion,
// trying "python3" then "python" on PATH. It never installs Python itself
// -- the interpreter is a required external dependency in v1.
func DetectPython(ctx context.Context, r Runner, minVersion string) (path string, version string, err error) {
	for _, candidate := range []string{"python3", "python"} {
		resolved, lookErr := exec.LookPath(candidate)
		if lookErr != nil {
			continue
		}
		out, runErr := r.Run(ctx, "", resolved, []string{"--version"})
		if runErr != nil {
			continue
		}
		ver := parsePythonVersion(string(out))
		if ver == "" {
			continue
		}
		if minVersion != "" {
			cmp, cmpErr := manifest.CompareVersions(ver, minVersion)
			if cmpErr != nil || cmp < 0 {
				continue
			}
		}
		return resolved, ver, nil
	}
	return "", "", fmt.Errorf("no python3 interpreter satisfying >=%s found on PATH", minVersion)
}

func parsePythonVersion(output string) string {
	// `python3 --version` prints "Python 3.11.6\n" (sometimes to stderr,
	// which CombinedOutput already captures).
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) != 2 {
		return ""
	}
	return fields[1]
}

// VenvDir returns the conventional venv directory under a component's
// runtime root.
func VenvDir(runtimeRoot string) string {
	return filepath.Join(runtimeRoot, "venv")
}

// VenvPython returns the path to the venv's own interpreter.
func VenvPython(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

// VenvConsoleScript returns the path to a console-script entry point
// installed inside the venv (e.g. "howlwriter" -> venv/bin/howlwriter).
func VenvConsoleScript(venvDir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}

// CreateVenv creates a fresh isolated virtualenv at venvDir using
// pythonPath as the base interpreter.
func CreateVenv(ctx context.Context, r Runner, pythonPath, venvDir string) error {
	if err := os.MkdirAll(filepath.Dir(venvDir), 0o755); err != nil {
		return fmt.Errorf("failed to create runtime directory: %w", err)
	}
	if _, err := r.Run(ctx, "", pythonPath, []string{"-m", "venv", venvDir}); err != nil {
		return fmt.Errorf("failed to create python virtualenv at %s: %w", venvDir, err)
	}
	return nil
}

// PipInstallEditable installs the package found at checkoutDir (a
// directory containing pyproject.toml/setup.py) into the venv in editable
// mode, with the given optional extras (e.g. ["dev", "web"] for
// `pip install -e ".[dev,web]"`).
func PipInstallEditable(ctx context.Context, r Runner, venvDir, checkoutDir, packageDir string, extras []string) error {
	target := "."
	if len(extras) > 0 {
		target = fmt.Sprintf(".[%s]", strings.Join(extras, ","))
	}
	dir := checkoutDir
	if packageDir != "" && packageDir != "." {
		dir = filepath.Join(checkoutDir, packageDir)
	}

	python := VenvPython(venvDir)
	if _, err := r.Run(ctx, dir, python, []string{"-m", "pip", "install", "--quiet", "--upgrade", "pip"}); err != nil {
		return fmt.Errorf("failed to upgrade pip in venv: %w", err)
	}
	if _, err := r.Run(ctx, dir, python, []string{"-m", "pip", "install", "--quiet", "-e", target}); err != nil {
		return fmt.Errorf("failed to install package into venv: %w", err)
	}
	return nil
}

// PipInstallWheel installs a single prebuilt, already-checksum-verified
// wheel file into the venv (non-editable), with the given optional extras
// (e.g. ["web"] for `pip install "<wheel>[web]"`). Unlike
// PipInstallEditable, there is no source checkout involved: wheelPath is
// the whole install unit.
func PipInstallWheel(ctx context.Context, r Runner, venvDir, wheelPath string, extras []string) error {
	target := wheelPath
	if len(extras) > 0 {
		target = fmt.Sprintf("%s[%s]", wheelPath, strings.Join(extras, ","))
	}

	python := VenvPython(venvDir)
	if _, err := r.Run(ctx, "", python, []string{"-m", "pip", "install", "--quiet", "--upgrade", "pip"}); err != nil {
		return fmt.Errorf("failed to upgrade pip in venv: %w", err)
	}
	if _, err := r.Run(ctx, "", python, []string{"-m", "pip", "install", "--quiet", target}); err != nil {
		return fmt.Errorf("failed to install wheel into venv: %w", err)
	}
	return nil
}

// WriteWrapperScript writes a thin, platform-appropriate wrapper at
// scriptPath that execs the venv's console-script entry point. This lets
// Python-installed components be located the same way as any other
// component's binary (via the release directory's <component-name>
// executable), regardless of install method.
func WriteWrapperScript(scriptPath, venvDir, consoleScript string) error {
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for wrapper script: %w", err)
	}

	target := VenvConsoleScript(venvDir, consoleScript)
	var content string
	if runtime.GOOS == "windows" {
		content = fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", target)
	} else {
		content = fmt.Sprintf("#!/bin/sh\nexec \"%s\" \"$@\"\n", target)
	}

	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("failed to write wrapper script %s: %w", scriptPath, err)
	}
	return nil
}
