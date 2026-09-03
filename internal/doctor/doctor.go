// Package doctor implements installer-owned diagnostics and repair:
// environment support, required/optional dependency availability, managed
// runtime health, installer state integrity, installation lock staleness,
// and per-installed-component health checks. It never diagnoses or
// repairs anything outside what Howl itself owns.
package doctor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/howlcipher/howl/internal/engine"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/pyruntime"
	"github.com/howlcipher/howl/internal/state"
)

// CheckStatus is the outcome of a single diagnostic check.
type CheckStatus string

const (
	StatusPass CheckStatus = "PASS"
	StatusWarn CheckStatus = "WARN"
	StatusFail CheckStatus = "FAIL"
)

// SummaryStatus is the overall installer health.
type SummaryStatus string

const (
	HealthHealthy             SummaryStatus = "HEALTHY"
	HealthHealthyWithWarnings SummaryStatus = "HEALTHY WITH WARNINGS"
	HealthDegraded            SummaryStatus = "DEGRADED"
	HealthFailed              SummaryStatus = "FAILED"
)

// CheckResult records one diagnostic check's outcome.
type CheckResult struct {
	Category string      `json:"category"`
	Name     string      `json:"name"`
	Status   CheckStatus `json:"status"`
	Message  string      `json:"message"`
	Details  string      `json:"details,omitempty"`
	Fixed    bool        `json:"fixed,omitempty"`
}

// DiagnosticReport is the complete diagnostic output of one doctor run.
type DiagnosticReport struct {
	Timestamp string        `json:"timestamp"`
	Summary   SummaryStatus `json:"summary"`
	Checks    []CheckResult `json:"checks"`
}

// Options configures a doctor run.
type Options struct {
	Manifest  *manifest.Manifest
	State     *state.State
	StatePath string
	Paths     platform.Paths
	Strict    bool
	Fix       bool

	Detector  plan.Detector
	Health    engine.HealthChecker
	Installer engine.Installer // only required when Fix is set
	Now       func() time.Time
}

// Run executes the full suite of installer-owned diagnostics, optionally
// applying deterministic repairs when opts.Fix is set.
func Run(ctx context.Context, opts Options) *DiagnosticReport {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	report := &DiagnosticReport{Timestamp: now().UTC().Format(time.RFC3339)}

	report.Checks = append(report.Checks, checkEnvironment(opts.Paths)...)
	report.Checks = append(report.Checks, checkDependencies(opts)...)
	report.Checks = append(report.Checks, checkManagedRuntimes(opts)...)
	report.Checks = append(report.Checks, checkManifest(opts.Manifest))
	report.Checks = append(report.Checks, checkCompatibility(opts.Manifest, opts.State))
	report.Checks = append(report.Checks, checkLock(opts, now)...)
	report.Checks = append(report.Checks, checkInterruptedOperation(opts, now)...)
	report.Checks = append(report.Checks, checkComponentHealth(ctx, opts)...)

	report.Summary = computeSummary(report.Checks, opts.Strict)
	return report
}

func checkEnvironment(paths platform.Paths) []CheckResult {
	info := platform.Detect()
	var checks []CheckResult

	osCheck := CheckResult{Category: "Environment", Name: "OS / Architecture"}
	if info.Supported {
		osCheck.Status = StatusPass
		label := info.OS
		if info.IsBazzite {
			label = "Bazzite"
		} else if info.Distro.PrettyName != "" {
			label = info.Distro.PrettyName
		}
		osCheck.Message = fmt.Sprintf("%s (%s)", label, info.Arch)
	} else {
		osCheck.Status = StatusFail
		osCheck.Message = info.UnsupportedReason
	}
	checks = append(checks, osCheck)

	writable := CheckResult{Category: "Environment", Name: "Installation directory writable"}
	if err := paths.EnsureOwnedDirs(); err != nil {
		writable.Status = StatusFail
		writable.Message = "cannot create or write to Howl's data directory"
		writable.Details = err.Error()
	} else {
		writable.Status = StatusPass
		writable.Message = paths.DataHome
	}
	checks = append(checks, writable)

	return checks
}

func checkDependencies(opts Options) []CheckResult {
	var checks []CheckResult
	if opts.Manifest == nil || opts.Detector == nil {
		return checks
	}

	seen := map[string]bool{}
	for _, c := range opts.Manifest.Components {
		for _, ext := range c.ExternalDependencies {
			if seen[ext.Name] {
				continue
			}
			seen[ext.Name] = true
			found, detail := opts.Detector.Detect(ext.Name)
			cr := CheckResult{Category: "Dependencies", Name: ext.Name}
			switch {
			case found:
				cr.Status = StatusPass
				cr.Message = detail
				if cr.Message == "" {
					cr.Message = "found"
				}
			case ext.Required:
				cr.Status = StatusFail
				cr.Message = fmt.Sprintf("required dependency not found (needed for: %s)", ext.Capability)
			default:
				cr.Status = StatusWarn
				cr.Message = fmt.Sprintf("optional dependency not found; capability %q unavailable", ext.Capability)
			}
			checks = append(checks, cr)
		}
	}

	for _, cap := range opts.Manifest.OptionalCapabilities {
		found, detail := opts.Detector.Detect(cap.Name)
		cr := CheckResult{Category: "Optional Capabilities", Name: cap.Name}
		if found {
			cr.Status = StatusPass
			cr.Message = detail
			if cr.Message == "" {
				cr.Message = "found"
			}
		} else {
			cr.Status = StatusWarn
			cr.Message = fmt.Sprintf("not installed; %s unavailable", cap.Capability)
		}
		checks = append(checks, cr)
	}

	return checks
}

func checkManagedRuntimes(opts Options) []CheckResult {
	var checks []CheckResult
	if opts.Manifest == nil {
		return checks
	}
	for _, c := range opts.Manifest.Components {
		if c.Install.Method != manifest.MethodSourceBuild || c.Install.SourceBuild == nil || c.Install.SourceBuild.Language != "python" {
			continue
		}
		if _, installed := opts.State.Components[c.Name]; !installed {
			continue
		}
		venvPython := pyruntime.VenvPython(pyruntime.VenvDir(opts.Paths.RuntimeDir(c.Name)))
		cr := CheckResult{Category: "Managed Runtimes", Name: fmt.Sprintf("%s Python environment", c.Name)}
		if _, err := os.Stat(venvPython); err != nil {
			cr.Status = StatusFail
			cr.Message = "virtualenv missing or incomplete"
			cr.Details = venvPython
		} else {
			cr.Status = StatusPass
			cr.Message = "valid"
		}
		checks = append(checks, cr)
	}
	return checks
}

func checkManifest(m *manifest.Manifest) CheckResult {
	if m == nil {
		return CheckResult{Category: "Manifest", Name: "ecosystem release manifest", Status: StatusFail, Message: "failed to load"}
	}
	return CheckResult{Category: "Manifest", Name: "ecosystem release manifest", Status: StatusPass, Message: fmt.Sprintf("valid (ecosystem %s, channel %s)", m.Ecosystem.Version, m.Ecosystem.Channel)}
}

func checkCompatibility(m *manifest.Manifest, st *state.State) CheckResult {
	cr := CheckResult{Category: "Compatibility", Name: "installed component versions"}
	if m == nil || st == nil {
		cr.Status = StatusWarn
		cr.Message = "unable to evaluate: manifest or state unavailable"
		return cr
	}

	for _, c := range m.Components {
		installed, ok := st.Components[c.Name]
		if !ok {
			continue
		}
		for _, dep := range c.DependsOn {
			if dep.MinVersion == "" {
				continue
			}
			depInstalled, depOk := st.Components[dep.Component]
			if !depOk {
				continue
			}
			cmp, err := manifest.CompareVersions(depInstalled.Version, dep.MinVersion)
			if err != nil || cmp < 0 {
				cr.Status = StatusFail
				cr.Message = fmt.Sprintf("%s@%s requires %s>=%s, but %s@%s is installed", c.Name, installed.Version, dep.Component, dep.MinVersion, dep.Component, depInstalled.Version)
				return cr
			}
		}
	}

	cr.Status = StatusPass
	cr.Message = "compatible"
	return cr
}

func checkLock(opts Options, now func() time.Time) []CheckResult {
	info, found, err := state.Inspect(opts.Paths.LockFile())
	if err != nil {
		return []CheckResult{{Category: "Lock", Name: "installation lock", Status: StatusWarn, Message: "failed to inspect lock file", Details: err.Error()}}
	}
	if !found {
		return []CheckResult{{Category: "Lock", Name: "installation lock", Status: StatusPass, Message: "not held"}}
	}

	cr := CheckResult{Category: "Lock", Name: "installation lock"}
	if !state.IsStale(info) {
		cr.Status = StatusWarn
		cr.Message = fmt.Sprintf("held by active process %d", info.PID)
		return []CheckResult{cr}
	}

	cr.Status = StatusWarn
	cr.Message = fmt.Sprintf("stale lock from dead process %d", info.PID)
	if opts.Fix {
		if err := state.ForceRelease(opts.Paths.LockFile()); err != nil {
			cr.Status = StatusFail
			cr.Message = "failed to clear stale lock: " + err.Error()
		} else {
			cr.Fixed = true
			cr.Message += " (cleared)"
		}
	} else {
		cr.Message += "; run `howl doctor --fix` to clear it"
	}
	return []CheckResult{cr}
}

func checkInterruptedOperation(opts Options, now func() time.Time) []CheckResult {
	if opts.State == nil || opts.State.InProgress == nil {
		return []CheckResult{{Category: "State", Name: "interrupted operation", Status: StatusPass, Message: "none"}}
	}

	ip := opts.State.InProgress
	cr := CheckResult{Category: "State", Name: "interrupted operation"}
	cr.Status = StatusWarn
	cr.Message = fmt.Sprintf("%s on %s left incomplete", ip.Operation, ip.Component)

	if opts.Fix {
		// Only safe to discard if no live process still holds the lock.
		lockInfo, found, err := state.Inspect(opts.Paths.LockFile())
		if err == nil && (!found || state.IsStale(lockInfo)) {
			opts.State.InProgress = nil
			if err := opts.State.Save(opts.StatePath); err != nil {
				cr.Status = StatusFail
				cr.Message = "failed to clear interrupted-operation marker: " + err.Error()
			} else {
				cr.Fixed = true
				cr.Message += " (marker cleared; re-run the operation to complete it)"
			}
		} else {
			cr.Message += "; an active lock is still held, cannot safely clear"
		}
	} else {
		cr.Message += "; run `howl doctor --fix` to clear the marker, then retry the operation"
	}
	return []CheckResult{cr}
}

func checkComponentHealth(ctx context.Context, opts Options) []CheckResult {
	var checks []CheckResult
	if opts.Manifest == nil || opts.State == nil || opts.Health == nil {
		return checks
	}

	for _, c := range opts.Manifest.Components {
		if _, installed := opts.State.Components[c.Name]; !installed {
			continue
		}
		cr := CheckResult{Category: "Component Health", Name: c.Name}
		err := opts.Health.Check(ctx, c, opts.Paths)
		if err == nil {
			cr.Status = StatusPass
			cr.Message = "healthy"
			checks = append(checks, cr)
			continue
		}

		cr.Status = StatusFail
		cr.Message = "health check failed"
		cr.Details = err.Error()

		if opts.Fix && opts.Installer != nil {
			if reinstallErr := opts.Installer.Install(ctx, c, opts.Paths); reinstallErr != nil {
				cr.Message += fmt.Sprintf("; repair attempt failed: %v", reinstallErr)
			} else if recheckErr := opts.Health.Check(ctx, c, opts.Paths); recheckErr != nil {
				cr.Message += fmt.Sprintf("; reinstalled but still unhealthy: %v", recheckErr)
			} else {
				cr.Status = StatusWarn
				cr.Fixed = true
				cr.Message = "was unhealthy, reinstalled and now passes health check"
			}
		}
		checks = append(checks, cr)
	}
	return checks
}

func computeSummary(checks []CheckResult, strict bool) SummaryStatus {
	hasFail, hasWarn := false, false
	for _, c := range checks {
		switch c.Status {
		case StatusFail:
			hasFail = true
		case StatusWarn:
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
