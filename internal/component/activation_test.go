package component

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/howlcipher/howl/internal/platform"
)

func stageFakeRelease(t *testing.T, paths platform.Paths, name, version string) {
	t.Helper()
	dir := paths.ComponentReleaseDir(name, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, platform.ExeName(name)), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestActivateReleaseSymlinksIntoBinDir(t *testing.T) {
	paths := testPaths(t)
	stageFakeRelease(t, paths, "howlplane", "1.0.0")

	if err := ActivateRelease(paths, "howlplane", "1.0.0", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binLink := paths.ComponentBinLink("howlplane")
	info, err := os.Lstat(binLink)
	if err != nil {
		t.Fatalf("expected a bin link at %s: %v", binLink, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink, got mode %v", binLink, info.Mode())
	}
	target, err := os.Readlink(binLink)
	if err != nil {
		t.Fatal(err)
	}
	wantTarget := filepath.Join(paths.ComponentReleaseDir("howlplane", "1.0.0"), platform.ExeName("howlplane"))
	if target != wantTarget {
		t.Errorf("expected bin link to point at %s, got %s", wantTarget, target)
	}
}

func TestActivateReleaseSkipsBinDirForInternalComponent(t *testing.T) {
	paths := testPaths(t)
	stageFakeRelease(t, paths, "howlplane-engine", "1.0.0")

	if err := ActivateRelease(paths, "howlplane-engine", "1.0.0", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binLink := paths.ComponentBinLink("howlplane-engine")
	if _, err := os.Lstat(binLink); err == nil {
		t.Errorf("did not expect an internal component to be exposed on PATH at %s", binLink)
	}
}

func TestActivateReleaseUpdatesBinDirOnReactivation(t *testing.T) {
	paths := testPaths(t)
	stageFakeRelease(t, paths, "howlplane", "1.0.0")
	if err := ActivateRelease(paths, "howlplane", "1.0.0", true); err != nil {
		t.Fatal(err)
	}

	stageFakeRelease(t, paths, "howlplane", "1.1.0")
	if err := ActivateRelease(paths, "howlplane", "1.1.0", true); err != nil {
		t.Fatal(err)
	}

	binLink := paths.ComponentBinLink("howlplane")
	target, err := os.Readlink(binLink)
	if err != nil {
		t.Fatal(err)
	}
	wantTarget := filepath.Join(paths.ComponentReleaseDir("howlplane", "1.1.0"), platform.ExeName("howlplane"))
	if target != wantTarget {
		t.Errorf("expected bin link to be updated to point at %s, got %s", wantTarget, target)
	}
}
