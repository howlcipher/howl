package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsDefaults(t *testing.T) {
	home := "/home/testuser"
	p := ResolvePaths(envFrom(nil), home)

	if p.DataHome != filepath.Join(home, ".local", "share", "howl") {
		t.Errorf("unexpected DataHome: %s", p.DataHome)
	}
	if p.ConfigHome != filepath.Join(home, ".config", "howl") {
		t.Errorf("unexpected ConfigHome: %s", p.ConfigHome)
	}
	if p.CacheHome != filepath.Join(home, ".cache", "howl") {
		t.Errorf("unexpected CacheHome: %s", p.CacheHome)
	}
	if p.BinDir != filepath.Join(home, ".local", "bin") {
		t.Errorf("unexpected BinDir: %s", p.BinDir)
	}
}

func TestResolvePathsRespectsXDGOverrides(t *testing.T) {
	home := "/home/testuser"
	env := map[string]string{
		"XDG_DATA_HOME":   "/mnt/data",
		"XDG_CONFIG_HOME": "/mnt/config",
		"XDG_CACHE_HOME":  "/mnt/cache",
	}
	p := ResolvePaths(envFrom(env), home)

	if p.DataHome != filepath.Join("/mnt/data", "howl") {
		t.Errorf("expected XDG_DATA_HOME override, got %s", p.DataHome)
	}
	if p.ConfigHome != filepath.Join("/mnt/config", "howl") {
		t.Errorf("expected XDG_CONFIG_HOME override, got %s", p.ConfigHome)
	}
	if p.CacheHome != filepath.Join("/mnt/cache", "howl") {
		t.Errorf("expected XDG_CACHE_HOME override, got %s", p.CacheHome)
	}
}

func TestResolvePathsIgnoresRelativeXDGOverride(t *testing.T) {
	home := "/home/testuser"
	env := map[string]string{"XDG_DATA_HOME": "relative/path"}
	p := ResolvePaths(envFrom(env), home)

	if p.DataHome != filepath.Join(home, ".local", "share", "howl") {
		t.Errorf("expected relative XDG_DATA_HOME to be ignored, got %s", p.DataHome)
	}
}

func TestEnsureOwnedDirsCreatesLayout(t *testing.T) {
	home := t.TempDir()
	p := ResolvePaths(envFrom(nil), home)

	if err := p.EnsureOwnedDirs(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, dir := range []string{p.StateDir(), p.DownloadCacheDir(), p.ConfigHome, p.BinDir} {
		info, err := statDir(dir)
		if err != nil {
			t.Errorf("expected %s to exist: %v", dir, err)
			continue
		}
		if !info {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestPathsOwns(t *testing.T) {
	home := "/home/testuser"
	p := ResolvePaths(envFrom(nil), home)

	owned := []string{
		p.DataHome,
		filepath.Join(p.DataHome, "components", "howlframe"),
		p.CacheHome,
		filepath.Join(p.CacheHome, "downloads", "x.tar.gz"),
	}
	for _, path := range owned {
		if !p.Owns(path) {
			t.Errorf("expected Owns(%s) to be true", path)
		}
	}

	notOwned := []string{
		"/home/testuser",
		"/home/testuser/Documents/important.txt",
		"/etc/passwd",
		p.ConfigHome, // config is not part of the removable data footprint
	}
	for _, path := range notOwned {
		if p.Owns(path) {
			t.Errorf("expected Owns(%s) to be false", path)
		}
	}
}

func statDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
