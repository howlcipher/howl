// Package manifest provides parsing, schema validation, and discovery for
// the canonical Howl ecosystem manifest (ecosystem.toml).
package manifest

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

//go:embed default_ecosystem.toml
var defaultManifestBytes []byte

// Manifest represents the top-level ecosystem manifest schema.
type Manifest struct {
	Ecosystem  EcosystemMeta `toml:"ecosystem" json:"ecosystem"`
	Components []Component   `toml:"components" json:"components"`
}

// EcosystemMeta provides global metadata about the Howl ecosystem.
type EcosystemMeta struct {
	Name        string `toml:"name" json:"name"`
	Version     string `toml:"version" json:"version"`
	Description string `toml:"description" json:"description"`
}

// Component defines a recognized ecosystem component repository and its role.
type Component struct {
	Name           string   `toml:"name" json:"name"`
	Repository     string   `toml:"repository" json:"repository"`
	Role           string   `toml:"role" json:"role"`
	Binary         string   `toml:"binary" json:"binary"`
	Contract       string   `toml:"contract,omitempty" json:"contract,omitempty"`
	Optional       bool     `toml:"optional,omitempty" json:"optional,omitempty"`
	DiscoveryHints []string `toml:"discovery_hints,omitempty" json:"discovery_hints,omitempty"`
}

// Validate checks the structural and semantic integrity of the manifest.
func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.Ecosystem.Name) == "" {
		return fmt.Errorf("manifest validation error: ecosystem name cannot be empty")
	}
	if len(m.Components) == 0 {
		return fmt.Errorf("manifest validation error: no components defined")
	}

	seen := make(map[string]bool)
	for i, c := range m.Components {
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("manifest validation error: component at index %d has empty name", i)
		}
		if strings.TrimSpace(c.Repository) == "" {
			return fmt.Errorf("manifest validation error: component '%s' has empty repository", c.Name)
		}
		if strings.TrimSpace(c.Role) == "" {
			return fmt.Errorf("manifest validation error: component '%s' has empty role", c.Name)
		}
		if seen[c.Name] {
			return fmt.Errorf("manifest validation error: duplicate component name '%s'", c.Name)
		}
		seen[c.Name] = true
	}
	return nil
}

// GetComponent finds a component by name from the manifest.
func (m *Manifest) GetComponent(name string) (Component, bool) {
	for _, c := range m.Components {
		if c.Name == name {
			return c, true
		}
	}
	return Component{}, false
}

// Load loads and validates a manifest from a file path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ecosystem manifest at %s: %w", path, err)
	}
	return LoadBytes(data)
}

// LoadBytes parses and validates a manifest from raw TOML bytes.
func LoadBytes(data []byte) (*Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse ecosystem manifest TOML: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// FindManifestPath searches upward from startDir for ecosystem.toml.
func FindManifestPath(startDir string) (string, error) {
	if startDir == "" {
		startDir = "."
	}
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	curr := abs
	for {
		candidate := filepath.Join(curr, "ecosystem.toml")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", os.ErrNotExist
}

// LoadDefault tries to locate ecosystem.toml near startDir, falling back to embedded defaults.
func LoadDefault(startDir string) (*Manifest, string, error) {
	path, err := FindManifestPath(startDir)
	if err == nil {
		m, err := Load(path)
		if err != nil {
			return nil, path, err
		}
		return m, path, nil
	}

	// Fallback to embedded manifest
	m, err := LoadBytes(defaultManifestBytes)
	if err != nil {
		return nil, "<embedded>", fmt.Errorf("failed to load embedded manifest: %w", err)
	}
	return m, "<embedded>", nil
}
