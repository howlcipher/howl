package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/howlcipher/howl/internal/manifest"
)

func clearEnv(t *testing.T) {
	keys := []string{
		"HOWLPLANE_HOME", "HOWLPLANE_DIR", "HOWL_HOWLPLANE_HOME", "HOWL_HOWLPLANE_DIR",
		"HOWLFRAME_HOME", "HOWLFRAME_DIR", "HOWL_HOWLFRAME_HOME", "HOWL_HOWLFRAME_DIR",
		"HOWLCHANGEOPS_HOME", "HOWLCHANGEOPS_DIR", "HOWL_HOWLCHANGEOPS_HOME", "HOWL_HOWLCHANGEOPS_DIR",
		"HOWLWRITER_HOME", "HOWLWRITER_DIR", "HOWL_HOWLWRITER_HOME", "HOWL_HOWLWRITER_DIR",
		"HOWLBOARD_HOME", "HOWLBOARD_DIR", "HOWL_HOWLBOARD_HOME", "HOWL_HOWLBOARD_DIR",
		"HOWLNOTES_HOME", "HOWLNOTES_DIR", "HOWL_HOWLNOTES_HOME", "HOWL_HOWLNOTES_DIR",
		"HOWL_COMPONENTS_DIR",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestDiscoveryPrecedence(t *testing.T) {
	clearEnv(t)
	tempRoot := t.TempDir()

	// Create directories
	flagDir := filepath.Join(tempRoot, "flag_howlplane")
	envDir := filepath.Join(tempRoot, "env_howlplane")
	configDir := filepath.Join(tempRoot, "config_howlplane")
	siblingDir := filepath.Join(tempRoot, "howlplane")
	baseDir := filepath.Join(tempRoot, "howl")

	for _, d := range []string{flagDir, envDir, configDir, siblingDir, baseDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	comp := manifest.Component{
		Name:       "howlplane",
		Repository: "https://github.com/howlcipher/howlplane",
		Role:       "Control Plane",
	}

	configFile := filepath.Join(tempRoot, "config.toml")
	configContent := "[components]\nhowlplane = \"" + configDir + "\"\n"
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Sibling only
	engine := NewEngine(DiscoveryOptions{
		BaseDir: baseDir,
	})
	res := engine.DiscoverComponent(comp)
	if !res.Found || res.DiscoverySource != SourceSibling || res.RepoPath != siblingDir {
		t.Errorf("expected sibling discovery, got %+v", res)
	}

	// 2. Config overrides Sibling
	engine = NewEngine(DiscoveryOptions{
		BaseDir:    baseDir,
		ConfigPath: configFile,
	})
	res = engine.DiscoverComponent(comp)
	if !res.Found || res.DiscoverySource != SourceConfig || res.RepoPath != configDir {
		t.Errorf("expected config discovery, got %+v", res)
	}

	// 3. Env overrides Config
	t.Setenv("HOWLPLANE_HOME", envDir)
	engine = NewEngine(DiscoveryOptions{
		BaseDir:    baseDir,
		ConfigPath: configFile,
	})
	res = engine.DiscoverComponent(comp)
	if !res.Found || res.DiscoverySource != SourceEnv || res.RepoPath != envDir {
		t.Errorf("expected env discovery, got %+v", res)
	}

	// 4. Flag overrides Env
	engine = NewEngine(DiscoveryOptions{
		BaseDir:    baseDir,
		ConfigPath: configFile,
		ExplicitDirs: map[string]string{
			"howlplane": flagDir,
		},
	})
	res = engine.DiscoverComponent(comp)
	if !res.Found || res.DiscoverySource != SourceFlag || res.RepoPath != flagDir {
		t.Errorf("expected flag discovery, got %+v", res)
	}
}

func TestCurrentRepoDiscovery(t *testing.T) {
	clearEnv(t)
	tempRoot := t.TempDir()
	planeDir := filepath.Join(tempRoot, "howlplane")
	if err := os.MkdirAll(planeDir, 0755); err != nil {
		t.Fatal(err)
	}

	comp := manifest.Component{
		Name:       "howlplane",
		Repository: "https://github.com/howlcipher/howlplane",
		Role:       "Control Plane",
	}

	engine := NewEngine(DiscoveryOptions{
		BaseDir: planeDir,
	})
	res := engine.DiscoverComponent(comp)
	if !res.Found || res.DiscoverySource != SourceCurrentRepo || res.RepoPath != planeDir {
		t.Errorf("expected current_repo discovery, got %+v", res)
	}
}

func TestExecutableDiscoveryInRepo(t *testing.T) {
	clearEnv(t)
	tempRoot := t.TempDir()
	planeDir := filepath.Join(tempRoot, "howlplane")
	binDir := filepath.Join(planeDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	binPath := filepath.Join(binDir, "howlplane")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	comp := manifest.Component{
		Name:       "howlplane",
		Repository: "https://github.com/howlcipher/howlplane",
		Role:       "Control Plane",
	}

	engine := NewEngine(DiscoveryOptions{
		ExplicitDirs: map[string]string{
			"howlplane": planeDir,
		},
	})
	res := engine.DiscoverComponent(comp)
	if res.ExecutablePath != binPath {
		t.Errorf("expected executable path %s, got %s", binPath, res.ExecutablePath)
	}
}

func TestMissingComponent(t *testing.T) {
	clearEnv(t)
	tempRoot := t.TempDir()
	baseDir := filepath.Join(tempRoot, "empty")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}

	comp := manifest.Component{
		Name:       "nonexistent",
		Repository: "https://github.com/howlcipher/nonexistent",
		Role:       "Unknown",
	}

	engine := NewEngine(DiscoveryOptions{
		BaseDir: baseDir,
	})
	res := engine.DiscoverComponent(comp)
	if res.Found || res.DiscoverySource != SourceNone {
		t.Errorf("expected missing component, got %+v", res)
	}
}
