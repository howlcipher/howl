// Package manifest provides parsing, schema validation, and dependency
// ordering for the canonical Howl ecosystem release manifest
// (ecosystem.toml). The manifest describes a tested, compatible Howl
// ecosystem release: which components to install, in what order, by what
// method, and what has to be true on the machine for each to work.
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

// CurrentSchemaVersion is the manifest schema version this build of Howl
// understands.
const CurrentSchemaVersion = 1

// InstallMethod identifies how a component is installed.
type InstallMethod string

const (
	MethodGithubRelease      InstallMethod = "github_release"
	MethodSourceBuild        InstallMethod = "source_build"
	MethodGithubReleaseWheel InstallMethod = "github_release_python_wheel"
)

// HealthCheckType identifies how a component's installation is verified.
type HealthCheckType string

const (
	// HealthExecVersion runs the installed binary with the given args
	// (typically ["--version"]) and requires it to exit successfully with
	// non-empty output.
	HealthExecVersion HealthCheckType = "exec_version"
	// HealthBinaryExists only checks that the expected executable exists
	// and is executable. This is the conservative default for components
	// that don't yet expose a confirmed-safe self-check invocation.
	HealthBinaryExists HealthCheckType = "binary_exists"
	// HealthPythonImport imports the given module using the component's
	// managed virtualenv interpreter.
	HealthPythonImport HealthCheckType = "python_import"
)

// Manifest is the top-level ecosystem release manifest.
type Manifest struct {
	SchemaVersion        int                  `toml:"schema_version" json:"schema_version"`
	Ecosystem            EcosystemMeta        `toml:"ecosystem" json:"ecosystem"`
	Components           []Component          `toml:"components" json:"components"`
	OptionalCapabilities []ExternalDependency `toml:"optional_capabilities,omitempty" json:"optional_capabilities,omitempty"`
}

// EcosystemMeta provides global metadata about a Howl ecosystem release.
type EcosystemMeta struct {
	Name        string `toml:"name" json:"name"`
	Version     string `toml:"version" json:"version"`
	Channel     string `toml:"channel" json:"channel"`
	Description string `toml:"description" json:"description"`
}

// Dependency is a required relationship on another component within the
// same manifest.
type Dependency struct {
	Component  string `toml:"component" json:"component"`
	MinVersion string `toml:"min_version,omitempty" json:"min_version,omitempty"`
}

// ExternalDependency describes a system-level prerequisite outside Howl's
// own component set (e.g. git, go, python3, ollama).
type ExternalDependency struct {
	Name       string `toml:"name" json:"name"`
	Required   bool   `toml:"required" json:"required"`
	Capability string `toml:"capability,omitempty" json:"capability,omitempty"`
	MinVersion string `toml:"min_version,omitempty" json:"min_version,omitempty"`
}

// HealthCheck declares how Howl verifies a component installed correctly.
type HealthCheck struct {
	Type   HealthCheckType `toml:"type" json:"type"`
	Args   []string        `toml:"args,omitempty" json:"args,omitempty"`
	Module string          `toml:"module,omitempty" json:"module,omitempty"`
}

// BuildStep is a single typed post-build invocation of an already-installed
// ecosystem component's own CLI (never an arbitrary shell string).
type BuildStep struct {
	RunComponent string   `toml:"run_component" json:"run_component"`
	Args         []string `toml:"args" json:"args"`
}

// GithubReleaseSource describes how to fetch a component from GitHub
// Releases.
type GithubReleaseSource struct {
	Repository      string `toml:"repository" json:"repository"`
	ArtifactPattern string `toml:"artifact_pattern" json:"artifact_pattern"`
	ChecksumFile    string `toml:"checksum_file" json:"checksum_file"`
}

// ForProfile returns c as installed under profile: the developer profile
// swaps in DeveloperInstall where the manifest declares one. Install
// planning and doctor's repair must agree on this, or `howl doctor --fix`
// would reinstall a developer's editable checkout as a release artifact.
func (c Component) ForProfile(profile string) Component {
	if profile == "developer" && c.DeveloperInstall != nil {
		c.Install = *c.DeveloperInstall
	}
	return c
}

// GoSourceBuild describes building a Go component from a local checkout.
type GoSourceBuild struct {
	Package      string      `toml:"package" json:"package"`
	BuildOutput  string      `toml:"build_output" json:"build_output"`
	ExtraSteps   []BuildStep `toml:"extra_steps,omitempty" json:"extra_steps,omitempty"`
	SidecarFiles []string    `toml:"sidecar_files,omitempty" json:"sidecar_files,omitempty"`
}

// PythonSourceBuild describes installing a Python component from a local
// checkout into an isolated virtualenv.
type PythonSourceBuild struct {
	PackageDir    string   `toml:"package_dir" json:"package_dir"`
	Extras        []string `toml:"extras,omitempty" json:"extras,omitempty"`
	ConsoleScript string   `toml:"console_script" json:"console_script"`
	MinPython     string   `toml:"min_python" json:"min_python"`
}

// SourceBuild describes building/installing a component from a local
// sibling source checkout.
type SourceBuild struct {
	CheckoutName string             `toml:"checkout_name" json:"checkout_name"`
	Language     string             `toml:"language" json:"language"`
	Go           *GoSourceBuild     `toml:"go,omitempty" json:"go,omitempty"`
	Python       *PythonSourceBuild `toml:"python,omitempty" json:"python,omitempty"`
}

// GithubReleaseWheel describes fetching a prebuilt, checksummed pure-Python
// wheel from GitHub Releases and installing it (non-editable) into a
// Howl-managed, isolated virtualenv. Unlike SourceBuild's python variant,
// this never checks out or builds from source: the wheel is the install
// unit, and its checksum is verified before pip ever touches it.
type GithubReleaseWheel struct {
	Repository      string   `toml:"repository" json:"repository"`
	ArtifactPattern string   `toml:"artifact_pattern" json:"artifact_pattern"`
	ChecksumFile    string   `toml:"checksum_file" json:"checksum_file"`
	ConsoleScript   string   `toml:"console_script" json:"console_script"`
	Extras          []string `toml:"extras,omitempty" json:"extras,omitempty"`
	MinPython       string   `toml:"min_python" json:"min_python"`
}

// Install describes how a component is obtained and installed.
type Install struct {
	Method             InstallMethod        `toml:"method" json:"method"`
	GithubRelease      *GithubReleaseSource `toml:"github_release,omitempty" json:"github_release,omitempty"`
	SourceBuild        *SourceBuild         `toml:"source_build,omitempty" json:"source_build,omitempty"`
	GithubReleaseWheel *GithubReleaseWheel  `toml:"github_release_python_wheel,omitempty" json:"github_release_python_wheel,omitempty"`
}

// Component defines a single registered ecosystem component.
type Component struct {
	Name        string `toml:"name" json:"name"`
	DisplayName string `toml:"display_name" json:"display_name"`
	Role        string `toml:"role" json:"role"`
	Repository  string `toml:"repository" json:"repository"`
	Version     string `toml:"version" json:"version"`
	Optional    bool   `toml:"optional,omitempty" json:"optional,omitempty"`
	// Internal marks a component that exists to be installed and health
	// checked, but is never itself exposed on the user's PATH (e.g. a
	// Python engine driven by a sibling CLI's binary, not invoked directly).
	Internal  bool     `toml:"internal,omitempty" json:"internal,omitempty"`
	Platforms []string `toml:"platforms" json:"platforms"`
	Archs     []string `toml:"archs" json:"archs"`
	Install   Install  `toml:"install" json:"install"`
	// DeveloperInstall, if set, replaces Install when the active profile is
	// "developer". Standard-profile planning never looks at this field, so
	// there is no code path by which a missing or broken release artifact
	// can silently fall back to a source build.
	DeveloperInstall     *Install             `toml:"developer_install,omitempty" json:"developer_install,omitempty"`
	HealthCheck          HealthCheck          `toml:"health_check" json:"health_check"`
	DependsOn            []Dependency         `toml:"depends_on,omitempty" json:"depends_on,omitempty"`
	ExternalDependencies []ExternalDependency `toml:"external_dependencies,omitempty" json:"external_dependencies,omitempty"`
	// PythonImports names components whose Python package this component
	// imports in its own process (HowlWriter runs HowlPlane's role
	// dispatcher in-process). Each one's package, and so its dependencies,
	// is installed into this component's runtime too, as that component is
	// installed under the active profile. Each must also be a depends_on.
	PythonImports []string `toml:"python_imports,omitempty" json:"python_imports,omitempty"`
	// Imports holds the components PythonImports names, as Manifest.Resolve
	// resolved them for one profile. It is never read from the manifest.
	Imports []Component `toml:"-" json:"-"`
}

// Resolve returns c as installed under profile, with the components it
// imports (PythonImports) resolved for the same profile. Install planning,
// rollback and doctor's repair all install a component through this, so a
// runtime always receives the packages it imports.
func (m *Manifest) Resolve(c Component, profile string) Component {
	c = c.ForProfile(profile)
	c.Imports = nil
	for _, name := range c.PythonImports {
		if imported, ok := m.GetComponent(name); ok {
			c.Imports = append(c.Imports, imported.ForProfile(profile))
		}
	}
	return c
}

// IsPython reports whether inst installs a Python package into a venv.
func (inst Install) IsPython() bool {
	switch inst.Method {
	case MethodGithubReleaseWheel:
		return inst.GithubReleaseWheel != nil
	case MethodSourceBuild:
		return inst.SourceBuild != nil && inst.SourceBuild.Language == "python"
	}
	return false
}

var validPlatforms = map[string]bool{"linux": true, "darwin": true, "windows": true}
var validArchs = map[string]bool{"amd64": true, "arm64": true}
var validChannels = map[string]bool{"stable": true, "beta": true, "dev": true}

// Validate checks the structural, semantic, and cross-component integrity
// of the manifest.
func (m *Manifest) Validate() error {
	if m.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("manifest validation error: unsupported schema_version %d (expected %d)", m.SchemaVersion, CurrentSchemaVersion)
	}
	if strings.TrimSpace(m.Ecosystem.Name) == "" {
		return fmt.Errorf("manifest validation error: ecosystem name cannot be empty")
	}
	if _, err := ParseVersion(m.Ecosystem.Version); err != nil {
		return fmt.Errorf("manifest validation error: invalid ecosystem version %q: %w", m.Ecosystem.Version, err)
	}
	if m.Ecosystem.Channel != "" && !validChannels[m.Ecosystem.Channel] {
		return fmt.Errorf("manifest validation error: invalid channel %q", m.Ecosystem.Channel)
	}
	if len(m.Components) == 0 {
		return fmt.Errorf("manifest validation error: no components defined")
	}

	seen := make(map[string]Component, len(m.Components))
	for _, c := range m.Components {
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("manifest validation error: a component has an empty name")
		}
		if _, dup := seen[c.Name]; dup {
			return fmt.Errorf("manifest validation error: duplicate component name %q", c.Name)
		}
		seen[c.Name] = c
	}

	for _, c := range m.Components {
		if err := validateComponent(c, seen); err != nil {
			return err
		}
	}

	if _, err := m.TopoOrder(); err != nil {
		return err
	}

	for _, ext := range m.OptionalCapabilities {
		if strings.TrimSpace(ext.Name) == "" {
			return fmt.Errorf("manifest validation error: optional_capabilities entry has an empty name")
		}
	}

	return nil
}

func validateComponent(c Component, all map[string]Component) error {
	if strings.TrimSpace(c.Role) == "" {
		return fmt.Errorf("manifest validation error: component %q has empty role", c.Name)
	}
	if _, err := ParseVersion(c.Version); err != nil {
		return fmt.Errorf("manifest validation error: component %q has invalid version %q: %w", c.Name, c.Version, err)
	}
	if len(c.Platforms) == 0 {
		return fmt.Errorf("manifest validation error: component %q declares no supported platforms", c.Name)
	}
	for _, p := range c.Platforms {
		if !validPlatforms[p] {
			return fmt.Errorf("manifest validation error: component %q declares unsupported platform %q", c.Name, p)
		}
	}
	if len(c.Archs) == 0 {
		return fmt.Errorf("manifest validation error: component %q declares no supported architectures", c.Name)
	}
	for _, a := range c.Archs {
		if !validArchs[a] {
			return fmt.Errorf("manifest validation error: component %q declares unsupported architecture %q", c.Name, a)
		}
	}

	if err := validateInstallStanza(c.Name, c.Install); err != nil {
		return err
	}
	if c.DeveloperInstall != nil {
		if err := validateInstallStanza(c.Name, *c.DeveloperInstall); err != nil {
			return err
		}
	}
	if err := validateHealthCheck(c); err != nil {
		return err
	}

	for _, dep := range c.DependsOn {
		if dep.Component == c.Name {
			return fmt.Errorf("manifest validation error: component %q cannot depend on itself", c.Name)
		}
		target, ok := all[dep.Component]
		if !ok {
			return fmt.Errorf("manifest validation error: component %q depends on unknown component %q", c.Name, dep.Component)
		}
		if dep.MinVersion != "" {
			cmp, err := CompareVersions(target.Version, dep.MinVersion)
			if err != nil {
				return fmt.Errorf("manifest validation error: component %q has invalid min_version %q for dependency %q: %w", c.Name, dep.MinVersion, dep.Component, err)
			}
			if cmp < 0 {
				return fmt.Errorf("manifest validation error: compatibility failure: %q requires %q >= %s, but manifest declares %s", c.Name, dep.Component, dep.MinVersion, target.Version)
			}
		}
	}

	for _, ext := range c.ExternalDependencies {
		if strings.TrimSpace(ext.Name) == "" {
			return fmt.Errorf("manifest validation error: component %q has an external dependency with an empty name", c.Name)
		}
	}

	return validatePythonImports(c, all)
}

func validatePythonImports(c Component, all map[string]Component) error {
	for _, name := range c.PythonImports {
		target, ok := all[name]
		if !ok {
			return fmt.Errorf("manifest validation error: component %q imports unknown component %q", c.Name, name)
		}
		if name == c.Name {
			return fmt.Errorf("manifest validation error: component %q cannot import itself", c.Name)
		}
		dependsOn := false
		for _, dep := range c.DependsOn {
			dependsOn = dependsOn || dep.Component == name
		}
		if !dependsOn {
			return fmt.Errorf("manifest validation error: component %q imports %q but does not depend on it", c.Name, name)
		}
		// Under every profile both must be Python packages installed the same
		// way: otherwise there is no venv to install into, no package to
		// install, or an editable checkout in a runtime that tracks none.
		for _, profile := range []string{"standard", "developer"} {
			own, imported := c.ForProfile(profile).Install, target.ForProfile(profile).Install
			if !own.IsPython() || !imported.IsPython() || own.Method != imported.Method {
				return fmt.Errorf("manifest validation error: component %q imports %q, so both must install as Python "+
					"packages by the same method (%s profile)", c.Name, name, profile)
			}
		}
	}
	return nil
}

func validateInstallStanza(name string, inst Install) error {
	switch inst.Method {
	case MethodGithubRelease:
		gr := inst.GithubRelease
		if gr == nil {
			return fmt.Errorf("manifest validation error: component %q uses github_release but has no github_release block", name)
		}
		if strings.TrimSpace(gr.Repository) == "" || strings.TrimSpace(gr.ArtifactPattern) == "" || strings.TrimSpace(gr.ChecksumFile) == "" {
			return fmt.Errorf("manifest validation error: component %q github_release block is missing required fields", name)
		}
	case MethodSourceBuild:
		sb := inst.SourceBuild
		if sb == nil {
			return fmt.Errorf("manifest validation error: component %q uses source_build but has no source_build block", name)
		}
		if strings.TrimSpace(sb.CheckoutName) == "" {
			return fmt.Errorf("manifest validation error: component %q source_build block is missing checkout_name", name)
		}
		switch sb.Language {
		case "go":
			if sb.Go == nil || strings.TrimSpace(sb.Go.Package) == "" || strings.TrimSpace(sb.Go.BuildOutput) == "" {
				return fmt.Errorf("manifest validation error: component %q go source_build block is missing required fields", name)
			}
			if err := ValidateGoSidecarFiles(sb.Go, name); err != nil {
				return err
			}
		case "python":
			if sb.Python == nil || strings.TrimSpace(sb.Python.ConsoleScript) == "" {
				return fmt.Errorf("manifest validation error: component %q python source_build block is missing required fields", name)
			}
		default:
			return fmt.Errorf("manifest validation error: component %q source_build declares unsupported language %q", name, sb.Language)
		}
	case MethodGithubReleaseWheel:
		grw := inst.GithubReleaseWheel
		if grw == nil {
			return fmt.Errorf("manifest validation error: component %q uses github_release_python_wheel but has no github_release_python_wheel block", name)
		}
		if strings.TrimSpace(grw.Repository) == "" || strings.TrimSpace(grw.ArtifactPattern) == "" || strings.TrimSpace(grw.ChecksumFile) == "" {
			return fmt.Errorf("manifest validation error: component %q github_release_python_wheel block is missing required fields", name)
		}
		if strings.TrimSpace(grw.ConsoleScript) == "" || strings.TrimSpace(grw.MinPython) == "" {
			return fmt.Errorf("manifest validation error: component %q github_release_python_wheel block is missing console_script or min_python", name)
		}
	default:
		return fmt.Errorf("manifest validation error: component %q declares unsupported install method %q", name, inst.Method)
	}
	return nil
}

func validateHealthCheck(c Component) error {
	switch c.HealthCheck.Type {
	case HealthExecVersion, HealthBinaryExists:
		return nil
	case HealthPythonImport:
		if strings.TrimSpace(c.HealthCheck.Module) == "" {
			return fmt.Errorf("manifest validation error: component %q python_import health check requires module", c.Name)
		}
		return nil
	default:
		return fmt.Errorf("manifest validation error: component %q declares unsupported health_check type %q", c.Name, c.HealthCheck.Type)
	}
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

// TopoOrder returns component names ordered so that every component's
// dependencies appear before it, or an error if a dependency cycle exists.
func (m *Manifest) TopoOrder() ([]string, error) {
	inDegree := make(map[string]int, len(m.Components))
	dependents := make(map[string][]string, len(m.Components))
	for _, c := range m.Components {
		if _, ok := inDegree[c.Name]; !ok {
			inDegree[c.Name] = 0
		}
	}
	for _, c := range m.Components {
		for _, dep := range c.DependsOn {
			inDegree[c.Name]++
			dependents[dep.Component] = append(dependents[dep.Component], c.Name)
		}
	}

	var queue []string
	for _, c := range m.Components {
		if inDegree[c.Name] == 0 {
			queue = append(queue, c.Name)
		}
	}

	var order []string
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		order = append(order, next)
		for _, d := range dependents[next] {
			inDegree[d]--
			if inDegree[d] == 0 {
				queue = append(queue, d)
			}
		}
	}

	if len(order) != len(m.Components) {
		return nil, fmt.Errorf("manifest validation error: dependency cycle detected among components")
	}
	return order, nil
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

// LoadDefault tries to locate ecosystem.toml near startDir, falling back to
// the embedded default release manifest.
func LoadDefault(startDir string) (*Manifest, string, error) {
	path, err := FindManifestPath(startDir)
	if err == nil {
		m, err := Load(path)
		if err != nil {
			return nil, path, err
		}
		return m, path, nil
	}

	m, err := LoadBytes(defaultManifestBytes)
	if err != nil {
		return nil, "<embedded>", fmt.Errorf("failed to load embedded manifest: %w", err)
	}
	return m, "<embedded>", nil
}

// ValidateGoSidecarFiles limits installation to regular files in the checkout root.
// Sidecars cannot replace executables or escape the component release directory.
func ValidateGoSidecarFiles(goBuild *GoSourceBuild, componentName string) error {
	seen := make(map[string]bool)
	for _, name := range goBuild.SidecarFiles {
		if strings.TrimSpace(name) != name || name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\:"<>|?*`) || strings.HasSuffix(name, ".") {
			return fmt.Errorf("invalid sidecar file %q: expected a checkout-root filename", name)
		}
		key := strings.ToLower(name)
		if key == strings.ToLower(componentName) || key == strings.ToLower(componentName+".exe") || key == strings.ToLower(goBuild.BuildOutput) || key == strings.ToLower(goBuild.BuildOutput+".exe") || seen[key] {
			return fmt.Errorf("invalid sidecar file %q: duplicate or executable collision", name)
		}
		seen[key] = true
	}
	return nil
}
