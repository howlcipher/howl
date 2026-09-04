package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	tempDir := t.TempDir()
	fakeBin := filepath.Join(tempDir, "howlplane")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho ok"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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

func TestPlaneCommandMissingExecutable(t *testing.T) {
	// Empty PATH so howlplane is not discovered
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOWLPLANE_HOME", "")
	t.Setenv("HOWLPLANE_DIR", "")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"plane", "route", "test-objective"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error when howlplane executable is missing, got nil")
	}
	if !strings.Contains(err.Error(), "howlplane executable not found") {
		t.Errorf("expected missing executable error, got: %v", err)
	}
}

func TestPlaneCommandExitCodePreserved(t *testing.T) {
	tempDir := t.TempDir()
	fakeBin := filepath.Join(tempDir, "howlplane")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho ok"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	origRunner := plane.DefaultRunner
	mock := &mockPlaneRunner{exitCode: 42}
	plane.DefaultRunner = mock
	defer func() {
		plane.DefaultRunner = origRunner
	}()

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"plane", "route", "test-objective"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected non-zero exit error, got nil")
	}
	if !strings.Contains(err.Error(), "code 42") {
		t.Errorf("expected code 42 in error, got: %v", err)
	}
}

// hermeticHowlplane installs a stand-in howlplane executable and pins component
// discovery to it. Without this the engine resolves whatever HOWLPLANE_HOME the
// developer's shell happens to export, so the tests below would exercise the
// real control plane instead of a controlled stub.
func hermeticHowlplane(t *testing.T, script string) {
	t.Helper()

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "howlplane"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"HOWL_HOWLPLANE_DIR", "HOWL_HOWLPLANE_HOME", "HOWLPLANE_DIR", "HOWL_COMPONENTS_DIR"} {
		t.Setenv(key, "")
	}
	t.Setenv("HOWLPLANE_HOME", root)
}

// forwardedArgs runs `howl plane <argv...>` against a mock runner and returns
// exactly what the plane boundary handed to the howlplane executable.
func forwardedArgs(t *testing.T, argv ...string) []string {
	t.Helper()

	hermeticHowlplane(t, "#!/bin/sh\necho ok\n")

	origRunner := plane.DefaultRunner
	mock := &mockPlaneRunner{exitCode: 0}
	plane.DefaultRunner = mock
	t.Cleanup(func() { plane.DefaultRunner = origRunner })

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{"plane"}, argv...))

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected forwarding error for %v: %v", argv, err)
	}
	return mock.lastArgs
}

func assertForwarded(t *testing.T, argv []string) {
	t.Helper()
	got := forwardedArgs(t, argv...)
	if !reflect.DeepEqual(got, argv) {
		t.Errorf("argv not forwarded verbatim:\n  want %#v\n  got  %#v", argv, got)
	}
}

// The canonical dogfood invocation. Every one of these flags was previously
// consumed by cobra's unknown-flag allowlist, so howlplane received only
// ["marathon"] and rejected the call for a missing --authority-profile.
func TestPlaneForwardsMarathonInvocationVerbatim(t *testing.T) {
	assertForwarded(t, []string{
		"marathon",
		"--authority-profile", "howlframe-overnight",
		"--target-repo", "/tmp/example",
		"--repo-slug", "howlcipher/howlframe",
		"--max-tasks", "5",
		"--max-runtime-hours", "4",
	})
}

// `--repo` names a repository for howlplane to inspect. Dropping it silently
// made `howl plane status --repo X` report on the current directory instead.
func TestPlaneForwardsRepoFlagOnStatus(t *testing.T) {
	assertForwarded(t, []string{"status", "--repo", "/tmp/example"})
}

func TestPlaneForwardsFlagValueForms(t *testing.T) {
	cases := map[string][]string{
		"equals form":            {"marathon", "--authority-profile=howlframe-overnight"},
		"short flag with value":  {"status", "-R", "/tmp/example"},
		"boolean flag":           {"marathon", "--dry-run"},
		"boolean then value":     {"marathon", "--dry-run", "--max-tasks", "5"},
		"repeated flag":          {"work", "--label", "one", "--label", "two"},
		"positional after flags": {"route", "--json", "an objective"},
		"double dash separator":  {"work", "--", "--not-a-howl-flag"},
		"path value":             {"verify", "--repo", "/run/media/system/x/howlframe"},
	}
	for name, argv := range cases {
		t.Run(name, func(t *testing.T) { assertForwarded(t, argv) })
	}
}

// A value may legally begin with '-' (a negative number, a reason string).
// Reordering or dropping it would corrupt the HowlPlane CLI contract.
func TestPlaneForwardsValuesBeginningWithDash(t *testing.T) {
	assertForwarded(t, []string{"reject", "TASK-1", "--reason", "-not a flag-"})
}

// Ordering is part of the contract: howlplane's argparse subcommand must stay
// first and every flag must keep its position relative to its value.
func TestPlaneForwardsPreserveArgumentOrder(t *testing.T) {
	argv := []string{"marathon", "--max-tasks", "5", "--authority-profile", "strict"}
	got := forwardedArgs(t, argv...)
	if len(got) != len(argv) {
		t.Fatalf("expected %d args, got %d: %#v", len(argv), len(got), got)
	}
	for i := range argv {
		if got[i] != argv[i] {
			t.Errorf("arg %d: want %q, got %q", i, argv[i], got[i])
		}
	}
}

// Disabling flag parsing also disables cobra's own --help handling, so this
// guards that `plane --help` still documents the plane command rather than
// being forwarded to howlplane.
func TestPlaneHelpIsNotForwarded(t *testing.T) {
	for _, helpArg := range []string{"--help", "-h"} {
		t.Run(helpArg, func(t *testing.T) {
			hermeticHowlplane(t, "#!/bin/sh\necho ok\n")

			origRunner := plane.DefaultRunner
			mock := &mockPlaneRunner{exitCode: 0}
			plane.DefaultRunner = mock
			t.Cleanup(func() { plane.DefaultRunner = origRunner })

			rootCmd := NewRootCommand()
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs([]string{"plane", helpArg})

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("plane %s returned error: %v", helpArg, err)
			}
			if mock.lastArgs != nil {
				t.Errorf("plane %s was forwarded to howlplane as %#v", helpArg, mock.lastArgs)
			}
			if !strings.Contains(buf.String(), "HowlPlane") {
				t.Errorf("plane %s did not render plane help, got:\n%s", helpArg, buf.String())
			}
		})
	}
}

// `plane marathon --help` is HowlPlane's help, not this CLI's.
func TestPlaneForwardsSubcommandHelp(t *testing.T) {
	assertForwarded(t, []string{"marathon", "--help"})
}

// `project` is a real cobra subcommand of `plane`; disabling flag parsing on
// the parent must not stop cobra from routing to it.
func TestPlaneProjectSubcommandStillRoutes(t *testing.T) {
	got := forwardedArgs(t, "project", "validate", "/tmp/example")
	want := []string{"project", "validate", "/tmp/example"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("plane project validate did not route to the structured subcommand:\n  want %#v\n  got  %#v", want, got)
	}
}

// End-to-end through the real OSExecRunner rather than the mock: argv arrives
// intact at a genuine process, and its stdout, stderr, and exit code all
// survive the boundary.
func TestPlaneStreamAndExitCodeFidelityThroughRealExec(t *testing.T) {
	hermeticHowlplane(t, `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a"; done
echo "on stderr" >&2
exit 7
`)

	argv := []string{
		"marathon",
		"--authority-profile", "howlframe-overnight",
		"--target-repo", "/tmp/example",
		"--repo-slug", "howlcipher/howlframe",
		"--max-tasks", "5",
		"--max-runtime-hours", "4",
	}

	rootCmd := NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(append([]string{"plane"}, argv...))

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected the child's non-zero exit to surface as an error")
	}
	if !strings.Contains(err.Error(), "code 7") {
		t.Errorf("exit code not preserved, got: %v", err)
	}

	gotArgs := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if !reflect.DeepEqual(gotArgs, argv) {
		t.Errorf("argv did not survive a real exec:\n  want %#v\n  got  %#v", argv, gotArgs)
	}
	if !strings.Contains(stderr.String(), "on stderr") {
		t.Errorf("stderr not preserved, got: %q", stderr.String())
	}
}
