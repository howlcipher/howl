// Package discovery implements deterministic component discovery for the Howl ecosystem.
package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/pelletier/go-toml/v2"
)

// DiscoverySource denotes the mechanism through which a component was resolved.
type DiscoverySource string

const (
	SourceFlag        DiscoverySource = "flag"
	SourceEnv         DiscoverySource = "env"
	SourceConfig      DiscoverySource = "config"
	SourceCurrentRepo DiscoverySource = "current_repo"
	SourceSibling     DiscoverySource = "sibling"
	SourcePath        DiscoverySource = "path"
	SourceHint        DiscoverySource = "hint"
	SourceNone        DiscoverySource = "none"
)

// DiscoveredComponent contains all resolved ecosystem metadata for a single component.
type DiscoveredComponent struct {
	Name            string          `json:"name"`
	Role            string          `json:"role"`
	RepositoryURL   string          `json:"repository_url"`
	BinaryName      string          `json:"binary_name"`
	Contract        string          `json:"contract,omitempty"`
	Optional        bool            `json:"optional"`
	RepoPath        string          `json:"repo_path,omitempty"`
	ExecutablePath  string          `json:"executable_path,omitempty"`
	Found           bool            `json:"found"`
	DiscoverySource DiscoverySource `json:"discovery_source"`
	ManifestPath    string          `json:"manifest_path,omitempty"`
	GitBranch       string          `json:"git_branch,omitempty"`
	GitCommit       string          `json:"git_commit,omitempty"`
	Version         string          `json:"version,omitempty"`
}

// DiscoveryOptions controls the inputs and boundaries of component discovery.
type DiscoveryOptions struct {
	ExplicitDirs map[string]string // component name -> directory path
	BaseDir      string            // base directory for relative search (e.g. current repo root)
	ConfigPath   string            // optional custom path to ~/.config/howl/config.toml
}

// UserConfig represents the schema for ~/.config/howl/config.toml.
type UserConfig struct {
	Components map[string]string `toml:"components"`
}

// Engine performs deterministic discovery across known components.
type Engine struct {
	opts       DiscoveryOptions
	userConfig map[string]string
}

// NewEngine creates a new discovery engine with given options.
func NewEngine(opts DiscoveryOptions) *Engine {
	if opts.BaseDir == "" {
		opts.BaseDir = "."
	}
	absBase, err := filepath.Abs(opts.BaseDir)
	if err == nil {
		opts.BaseDir = absBase
	}

	cfg := loadUserConfig(opts.ConfigPath)
	return &Engine{
		opts:       opts,
		userConfig: cfg,
	}
}

// DiscoverAll scans for all components defined in the given manifest.
func (e *Engine) DiscoverAll(m *manifest.Manifest) []DiscoveredComponent {
	results := make([]DiscoveredComponent, 0, len(m.Components))
	for _, comp := range m.Components {
		results = append(results, e.DiscoverComponent(comp))
	}
	return results
}

// DiscoverComponent resolves a single component according to strict precedence.
func (e *Engine) DiscoverComponent(comp manifest.Component) DiscoveredComponent {
	res := DiscoveredComponent{
		Name:            comp.Name,
		Role:            comp.Role,
		RepositoryURL:   comp.Repository,
		BinaryName:      comp.Binary,
		Contract:        comp.Contract,
		Optional:        comp.Optional,
		DiscoverySource: SourceNone,
	}

	// 1. Explicit directory options/flags
	if e.opts.ExplicitDirs != nil {
		if path, ok := e.opts.ExplicitDirs[comp.Name]; ok && isDir(path) {
			res.RepoPath = cleanAbs(path)
			res.DiscoverySource = SourceFlag
			res.Found = true
		}
	}

	// 2. Environment variables
	if !res.Found {
		if path := checkEnvForComponent(comp.Name); path != "" && isDir(path) {
			res.RepoPath = cleanAbs(path)
			res.DiscoverySource = SourceEnv
			res.Found = true
		}
	}

	// 3. User configuration file
	if !res.Found && e.userConfig != nil {
		if path, ok := e.userConfig[comp.Name]; ok && isDir(path) {
			res.RepoPath = cleanAbs(path)
			res.DiscoverySource = SourceConfig
			res.Found = true
		}
	}

	// 4. Current repository (if running from inside the component's directory)
	if !res.Found {
		if isCurrentRepo(e.opts.BaseDir, comp.Name) {
			res.RepoPath = cleanAbs(e.opts.BaseDir)
			res.DiscoverySource = SourceCurrentRepo
			res.Found = true
		}
	}

	// 5. Sibling directory layout (e.g. ../<name>)
	if !res.Found {
		siblingCandidate := filepath.Join(filepath.Dir(e.opts.BaseDir), comp.Name)
		if isDir(siblingCandidate) {
			res.RepoPath = cleanAbs(siblingCandidate)
			res.DiscoverySource = SourceSibling
			res.Found = true
		}
	}

	// 6. Manifest discovery hints
	if !res.Found {
		for _, hint := range comp.DiscoveryHints {
			hintCandidate := filepath.Join(e.opts.BaseDir, hint)
			if isDir(hintCandidate) {
				res.RepoPath = cleanAbs(hintCandidate)
				res.DiscoverySource = SourceHint
				res.Found = true
				break
			}
		}
	}

	// Locate binary executable
	res.ExecutablePath = e.locateExecutable(comp, res.RepoPath)
	if res.ExecutablePath != "" && !res.Found {
		// If binary was found in PATH even without a repo checkout, mark as found via PATH
		res.Found = true
		res.DiscoverySource = SourcePath
	}

	// Check for project manifest and git metadata if repo was found
	if res.RepoPath != "" {
		res.ManifestPath = checkProjectManifest(res.RepoPath)
		res.GitBranch, res.GitCommit = getGitMetadata(res.RepoPath)
	}

	return res
}

func (e *Engine) locateExecutable(comp manifest.Component, repoPath string) string {
	// First check local repository build artifacts/binaries
	if repoPath != "" && comp.Binary != "" {
		candidates := []string{
			filepath.Join(repoPath, "bin", comp.Binary),
			filepath.Join(repoPath, comp.Binary),
			filepath.Join(repoPath, "cmd", comp.Binary, comp.Binary),
		}
		for _, cand := range candidates {
			if isExecutable(cand) {
				return cand
			}
		}
	}

	// Second check system PATH
	if comp.Binary != "" {
		if path, err := exec.LookPath(comp.Binary); err == nil {
			return path
		}
	}

	return ""
}

func checkEnvForComponent(name string) string {
	cleanName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	// Check HOWL_<NAME>_DIR, HOWL_<NAME>_HOME, <NAME>_HOME, <NAME>_DIR
	envKeys := []string{
		"HOWL_" + cleanName + "_DIR",
		"HOWL_" + cleanName + "_HOME",
		cleanName + "_HOME",
		cleanName + "_DIR",
	}

	for _, k := range envKeys {
		if val := os.Getenv(k); val != "" && isDir(val) {
			return val
		}
	}

	// Global components directory e.g. HOWL_COMPONENTS_DIR
	if globalDir := os.Getenv("HOWL_COMPONENTS_DIR"); globalDir != "" {
		candidate := filepath.Join(globalDir, name)
		if isDir(candidate) {
			return candidate
		}
	}

	return ""
}

func loadUserConfig(customPath string) map[string]string {
	configPath := customPath
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		configPath = filepath.Join(home, ".config", "howl", "config.toml")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg UserConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cfg.Components
}

func isCurrentRepo(dir, componentName string) bool {
	base := filepath.Base(dir)
	if base == componentName {
		return true
	}
	return false
}

func checkProjectManifest(repoDir string) string {
	candidates := []string{
		filepath.Join(repoDir, ".ai-project.toml"),
		filepath.Join(repoDir, "ecosystem.toml"),
	}
	for _, cand := range candidates {
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}
	return ""
}

func getGitMetadata(repoDir string) (branch, commit string) {
	headPath := filepath.Join(repoDir, ".git", "HEAD")
	headData, err := os.ReadFile(headPath)
	if err != nil {
		return "", ""
	}

	content := strings.TrimSpace(string(headData))
	if strings.HasPrefix(content, "ref: ") {
		ref := strings.TrimPrefix(content, "ref: ")
		branch = filepath.Base(ref)
		refPath := filepath.Join(repoDir, ".git", filepath.FromSlash(ref))
		if refData, err := os.ReadFile(refPath); err == nil {
			commit = strings.TrimSpace(string(refData))
			if len(commit) > 7 {
				commit = commit[:7]
			}
		}
	} else if len(content) >= 7 {
		commit = content[:7]
		branch = "detached"
	}

	return branch, commit
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0111 != 0
}

func cleanAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
