// Package doctor provides non-destructive diagnostics and health inspection for the Howl ecosystem.
package doctor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/howlcipher/howl/internal/contract"
	"github.com/howlcipher/howl/internal/discovery"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/version"
)

// CheckStatus represents the outcome of an individual diagnostic check.
type CheckStatus string

const (
	StatusPass    CheckStatus = "PASS"
	StatusWarn    CheckStatus = "WARN"
	StatusFail    CheckStatus = "FAIL"
	StatusUnknown CheckStatus = "UNKNOWN"
	StatusSkip    CheckStatus = "SKIP"
)

// SummaryStatus represents the overall ecosystem health.
type SummaryStatus string

const (
	HealthHealthy             SummaryStatus = "HEALTHY"
	HealthHealthyWithWarnings SummaryStatus = "HEALTHY WITH WARNINGS"
	HealthDegraded            SummaryStatus = "DEGRADED"
	HealthFailed              SummaryStatus = "FAILED"
)

// CheckResult records the evaluation of a single diagnostic check.
type CheckResult struct {
	Category string      `json:"category"`
	Name     string      `json:"name"`
	Status   CheckStatus `json:"status"`
	Message  string      `json:"message"`
	Details  string      `json:"details,omitempty"`
}

// DiagnosticReport contains the complete ecosystem diagnostic state.
type DiagnosticReport struct {
	Timestamp            string                          `json:"timestamp"`
	Summary              SummaryStatus                   `json:"summary"`
	Checks               []CheckResult                   `json:"checks"`
	DiscoveredComponents []discovery.DiscoveredComponent `json:"discovered_components"`
	Contracts            []contract.Contract             `json:"contracts"`
}

// Options configures doctor execution.
type Options struct {
	BaseDir      string
	ManifestPath string
	ConfigPath   string
	Strict       bool
}

// RunDiagnostics executes the full suite of non-destructive ecosystem health checks.
func RunDiagnostics(opts Options) (*DiagnosticReport, error) {
	report := &DiagnosticReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    make([]CheckResult, 0),
	}

	// 1. Platform & OS Check
	report.Checks = append(report.Checks, CheckResult{
		Category: "Platform",
		Name:     "OS / Architecture",
		Status:   StatusPass,
		Message:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})

	// 2. Go Toolchain Check
	goVerCheck := checkGoToolchain()
	report.Checks = append(report.Checks, goVerCheck)

	// 3. Git Toolchain Check
	gitCheck := checkGitToolchain()
	report.Checks = append(report.Checks, gitCheck)

	// 4. Ecosystem Manifest Check
	var m *manifest.Manifest
	var manifestSrc string
	var err error
	if opts.ManifestPath != "" {
		m, err = manifest.Load(opts.ManifestPath)
		manifestSrc = opts.ManifestPath
	} else {
		m, manifestSrc, err = manifest.LoadDefault(opts.BaseDir)
	}

	if err != nil {
		report.Checks = append(report.Checks, CheckResult{
			Category: "Manifest",
			Name:     "ecosystem.toml",
			Status:   StatusFail,
			Message:  "Failed to load ecosystem manifest",
			Details:  err.Error(),
		})
		report.Summary = HealthFailed
		return report, nil
	}

	report.Checks = append(report.Checks, CheckResult{
		Category: "Manifest",
		Name:     "ecosystem.toml",
		Status:   StatusPass,
		Message:  fmt.Sprintf("Valid manifest loaded from %s", manifestSrc),
	})

	// 5. Howl CLI Version Check
	vInfo := version.GetInfo()
	report.Checks = append(report.Checks, CheckResult{
		Category: "CLI",
		Name:     "Root Howl CLI",
		Status:   StatusPass,
		Message:  fmt.Sprintf("v%s (commit: %s, build: %s)", vInfo.Version, vInfo.GitCommit, vInfo.BuildDate),
	})

	// 6. Component Discovery Checks
	discEngine := discovery.NewEngine(discovery.DiscoveryOptions{
		BaseDir:    opts.BaseDir,
		ConfigPath: opts.ConfigPath,
	})
	discovered := discEngine.DiscoverAll(m)
	report.DiscoveredComponents = discovered

	for _, comp := range discovered {
		report.Checks = append(report.Checks, evaluateComponentHealth(comp))
	}

	// 7. Legacy 'ai' CLI Detection & Deprecation Check
	report.Checks = append(report.Checks, checkLegacyAICLI())

	// 8. Cross-Component Contract Checks
	contracts := contract.EvaluateContracts(discovered)
	report.Contracts = contracts
	report.Checks = append(report.Checks, evaluateContractHealth(contracts))

	// Compute Overall Summary
	report.Summary = computeSummary(report.Checks, opts.Strict)

	return report, nil
}

func checkGoToolchain() CheckResult {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Category: "Tooling",
			Name:     "Go Toolchain",
			Status:   StatusWarn,
			Message:  "Go compiler not found in PATH",
			Details:  "Go is required for source-based component development",
		}
	}
	verStr := strings.TrimSpace(string(out))
	return CheckResult{
		Category: "Tooling",
		Name:     "Go Toolchain",
		Status:   StatusPass,
		Message:  verStr,
	}
}

func checkGitToolchain() CheckResult {
	cmd := exec.Command("git", "--version")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Category: "Tooling",
			Name:     "Git Toolchain",
			Status:   StatusFail,
			Message:  "Git executable not found in PATH",
			Details:  "Git is required for ecosystem repository management",
		}
	}
	verStr := strings.TrimSpace(string(out))
	return CheckResult{
		Category: "Tooling",
		Name:     "Git Toolchain",
		Status:   StatusPass,
		Message:  verStr,
	}
}

func evaluateComponentHealth(comp discovery.DiscoveredComponent) CheckResult {
	name := comp.Name

	if !comp.Found {
		if comp.Optional {
			return CheckResult{
				Category: "Component",
				Name:     name,
				Status:   StatusUnknown,
				Message:  "Optional component not discovered",
				Details:  comp.RepositoryURL,
			}
		}
		return CheckResult{
			Category: "Component",
			Name:     name,
			Status:   StatusWarn,
			Message:  "Component repository or executable not discovered",
			Details:  comp.RepositoryURL,
		}
	}

	var details []string
	if comp.RepoPath != "" {
		details = append(details, fmt.Sprintf("repo: %s", comp.RepoPath))
	}
	if comp.ExecutablePath != "" {
		details = append(details, fmt.Sprintf("bin: %s", comp.ExecutablePath))
	}
	if comp.GitBranch != "" {
		details = append(details, fmt.Sprintf("git: %s@%s", comp.GitBranch, comp.GitCommit))
	}

	detailStr := strings.Join(details, ", ")
	if comp.ExecutablePath != "" {
		return CheckResult{
			Category: "Component",
			Name:     name,
			Status:   StatusPass,
			Message:  fmt.Sprintf("Discovered via %s (runnable)", comp.DiscoverySource),
			Details:  detailStr,
		}
	}

	return CheckResult{
		Category: "Component",
		Name:     name,
		Status:   StatusPass,
		Message:  fmt.Sprintf("Discovered source via %s", comp.DiscoverySource),
		Details:  detailStr,
	}
}

func checkLegacyAICLI() CheckResult {
	path, err := exec.LookPath("ai")
	if err != nil {
		return CheckResult{
			Category: "Compatibility",
			Name:     "Legacy 'ai' CLI",
			Status:   StatusPass,
			Message:  "No legacy 'ai' command conflict on PATH",
		}
	}

	// Check if this is indeed the HowlPlane legacy launcher or a symlink
	realPath, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = realPath
	}

	return CheckResult{
		Category: "Compatibility",
		Name:     "Legacy 'ai' CLI",
		Status:   StatusWarn,
		Message:  "Deprecated compatibility command detected on PATH",
		Details:  fmt.Sprintf("Found at %s. Use 'howl' or direct 'howlplane' entry point.", path),
	}
}

func evaluateContractHealth(contracts []contract.Contract) CheckResult {
	unknownCount := 0
	incompatibleCount := 0

	for _, c := range contracts {
		if c.Status == contract.StatusIncompatible {
			incompatibleCount++
		} else if c.Status == contract.StatusUnknown {
			unknownCount++
		}
	}

	if incompatibleCount > 0 {
		return CheckResult{
			Category: "Contracts",
			Name:     "Component Contracts",
			Status:   StatusFail,
			Message:  fmt.Sprintf("%d incompatible contract(s) detected", incompatibleCount),
		}
	}

	if unknownCount > 0 {
		return CheckResult{
			Category: "Contracts",
			Name:     "Component Contracts",
			Status:   StatusWarn,
			Message:  fmt.Sprintf("%d unverified contract(s) due to missing components", unknownCount),
		}
	}

	return CheckResult{
		Category: "Contracts",
		Name:     "Component Contracts",
		Status:   StatusPass,
		Message:  "All architectural contracts resolved",
	}
}

func computeSummary(checks []CheckResult, strict bool) SummaryStatus {
	hasFail := false
	hasWarn := false

	for _, c := range checks {
		switch c.Status {
		case StatusFail:
			hasFail = true
		case StatusWarn, StatusUnknown:
			hasWarn = true
		}
	}

	if hasFail {
		return HealthFailed
	}
	if hasWarn {
		if strict {
			return HealthDegraded
		}
		return HealthHealthyWithWarnings
	}
	return HealthHealthy
}
