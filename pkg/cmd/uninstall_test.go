package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/state"
)

func TestUninstallNothingInstalledIsNoop(t *testing.T) {
	sandboxHowlPaths(t)

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--yes"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Nothing is installed")) {
		t.Errorf("expected no-op message, got:\n%s", buf.String())
	}
}

func TestUninstallComponentRemovesOnlyHowlOwnedPaths(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlframe", "0.1.1")

	compDir := paths.ComponentDir("howlframe")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(compDir, "releases", "0.1.1", "howlframe")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	// A sibling directory that Howl does NOT own must never be touched.
	sibling := filepath.Join(filepath.Dir(paths.DataHome), "not-howl-owned")
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"uninstall", "howlframe", "--yes"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(compDir); !os.IsNotExist(err) {
		t.Errorf("expected component directory to be removed")
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Errorf("expected unrelated sibling directory to be left alone: %v", err)
	}

	data, err := os.ReadFile(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("howlframe")) {
		t.Errorf("expected howlframe removed from state, got:\n%s", data)
	}
}

func TestUninstallPurgeClearsRollbackHistory(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlframe", "0.2.0")

	// Simulate a prior successful update leaving rollback history behind.
	st, _, err := state.Load(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	st.EcosystemVersion = "0.2.0"
	st.PreviousEcosystem = "0.1.0"
	st.PreviousComponents = map[string]state.ComponentState{"howlframe": {Version: "0.1.0"}}
	if err := st.Save(paths.StateFile()); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"uninstall", "howlframe", "--purge", "--yes"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, existed, err := state.Load(paths.StateFile())
	if err != nil || !existed {
		t.Fatalf("expected state to still load, err=%v existed=%v", err, existed)
	}
	if len(after.PreviousComponents) != 0 {
		t.Errorf("expected --purge to clear rollback history, got %+v", after.PreviousComponents)
	}
	if after.EcosystemVersion != "" || after.PreviousEcosystem != "" {
		t.Errorf("expected ecosystem version fields reset after purging the only component, got %+v", after)
	}
}

func TestUninstallWithoutPurgeKeepsRollbackHistory(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlframe", "0.2.0")

	st, _, err := state.Load(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	st.PreviousComponents = map[string]state.ComponentState{"howlframe": {Version: "0.1.0"}}
	if err := st.Save(paths.StateFile()); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"uninstall", "howlframe", "--yes"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, _, err := state.Load(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	if len(after.PreviousComponents) == 0 {
		t.Errorf("expected rollback history to be kept without --purge")
	}
}

func TestUninstallFullPurgeClearsLeftoverHistoryWithNoActiveComponents(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)

	st, _, err := state.Load(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	// Nothing currently installed, but rollback history remains from an
	// earlier uninstall -- exactly the state a prior bug left behind.
	st.PreviousComponents = map[string]state.ComponentState{"howlframe": {Version: "0.1.0"}}
	st.PreviousEcosystem = "0.1.0"
	if err := st.Save(paths.StateFile()); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--purge", "--yes"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), "Nothing is installed") {
		t.Errorf("expected --purge to still act on leftover rollback history, got:\n%s", buf.String())
	}

	after, _, err := state.Load(paths.StateFile())
	if err != nil {
		t.Fatal(err)
	}
	if len(after.PreviousComponents) != 0 || after.PreviousEcosystem != "" {
		t.Errorf("expected leftover rollback history purged, got %+v", after)
	}
}

func TestUninstallCancelledWithoutYes(t *testing.T) {
	sandboxHowlPaths(t)
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlframe", "0.1.1")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetIn(strings.NewReader("n\n"))
	rootCmd.SetArgs([]string{"uninstall"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitCancelled {
		t.Errorf("expected ExitCancelled, got %v", err)
	}
}
