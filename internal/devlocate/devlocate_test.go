package devlocate

import (
	"os"
	"path/filepath"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLocateExplicitOverrideWins(t *testing.T) {
	dir := t.TempDir()
	got, ok := Locate("howlframe", Options{ExplicitDirs: map[string]string{"howlframe": dir}}, envFrom(nil))
	if !ok || got != dir {
		t.Fatalf("expected explicit dir %s, got %s (ok=%v)", dir, got, ok)
	}
}

func TestLocateEnvVarOverride(t *testing.T) {
	dir := t.TempDir()
	got, ok := Locate("howlframe", Options{}, envFrom(map[string]string{"HOWL_HOWLFRAME_DIR": dir}))
	if !ok || got != dir {
		t.Fatalf("expected env dir %s, got %s (ok=%v)", dir, got, ok)
	}
}

func TestLocateDevWorkspace(t *testing.T) {
	workspace := t.TempDir()
	checkoutDir := filepath.Join(workspace, "howlframe")
	if err := os.MkdirAll(checkoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := Locate("howlframe", Options{}, envFrom(map[string]string{"HOWL_DEV_WORKSPACE": workspace}))
	if !ok || got != checkoutDir {
		t.Fatalf("expected workspace-derived dir %s, got %s (ok=%v)", checkoutDir, got, ok)
	}
}

func TestLocateSiblingFallback(t *testing.T) {
	parent := t.TempDir()
	howlDir := filepath.Join(parent, "howl")
	frameDir := filepath.Join(parent, "howlframe")
	if err := os.MkdirAll(howlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := Locate("howlframe", Options{BaseDir: howlDir}, envFrom(nil))
	if !ok || got != frameDir {
		t.Fatalf("expected sibling dir %s, got %s (ok=%v)", frameDir, got, ok)
	}
}

func TestLocateNotFound(t *testing.T) {
	_, ok := Locate("nonexistent-component", Options{BaseDir: t.TempDir()}, envFrom(nil))
	if ok {
		t.Fatal("expected not found")
	}
}

func TestLocatePrecedenceExplicitBeatsEnv(t *testing.T) {
	explicitDir := t.TempDir()
	envDir := t.TempDir()

	got, ok := Locate("howlframe", Options{ExplicitDirs: map[string]string{"howlframe": explicitDir}}, envFrom(map[string]string{"HOWL_HOWLFRAME_DIR": envDir}))
	if !ok || got != explicitDir {
		t.Fatalf("expected explicit dir to win over env, got %s", got)
	}
}
