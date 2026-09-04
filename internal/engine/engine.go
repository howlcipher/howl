// Package engine executes an approved plan.Plan. It is the only package
// that mutates a Howl installation: staging, activation, health-checking,
// and state persistence all happen here, always against a plan that was
// built and rendered first.
package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/plan"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
)

// Installer fetches/builds a component at its target version and activates
// it under paths.ComponentDir(c.Name). Concrete implementations live in
// internal/component (github-release download, Go source build, Python
// source build).
type Installer interface {
	Install(ctx context.Context, c manifest.Component, paths platform.Paths) error
}

// HealthChecker verifies an installed component is actually functional.
type HealthChecker interface {
	Check(ctx context.Context, c manifest.Component, paths platform.Paths) error
}

// ComponentResult records what happened to one component during
// execution.
type ComponentResult struct {
	Name       string
	Action     plan.Action
	Err        error
	RolledBack bool
}

// Result is the outcome of executing a plan.
type Result struct {
	Components []ComponentResult
	Err        error // set if execution stopped early
}

// Succeeded reports whether every component in scope completed without
// error.
func (r *Result) Succeeded() bool {
	return r.Err == nil
}

// Engine executes plans against a concrete Installer/HealthChecker and
// persists state as it goes.
type Engine struct {
	Installer Installer
	Health    HealthChecker
	Paths     platform.Paths
	Now       func() time.Time
}

// New builds an Engine with real time.Now; tests can override Now.
func New(installer Installer, health HealthChecker, paths platform.Paths) *Engine {
	return &Engine{Installer: installer, Health: health, Paths: paths, Now: time.Now}
}

// Install executes a fresh-install plan. On the first component failure,
// execution stops immediately: components already installed before the
// failure are left installed (nothing is torn down), matching the
// documented behavior that a failed install reports exactly what
// succeeded and what didn't rather than guessing at cleanup.
func (e *Engine) Install(ctx context.Context, p *plan.Plan, st *state.State, statePath string) (*Result, error) {
	return e.run(ctx, p, st, statePath, state.OpInstall)
}

// Update executes an update plan. Before mutating anything, it snapshots
// the current component versions as the rollback target. If any
// component's install or health check fails, Update automatically
// restores the previous state and re-verifies it before reporting the
// outcome -- it never claims a rollback succeeded without a passing
// health check.
func (e *Engine) Update(ctx context.Context, p *plan.Plan, m *manifest.Manifest, st *state.State, statePath string) (*Result, error) {
	st.SnapshotForRollback(p.EcosystemVersion)
	if err := st.Save(statePath); err != nil {
		return nil, fmt.Errorf("failed to persist rollback snapshot before update: %w", err)
	}

	result, err := e.run(ctx, p, st, statePath, state.OpUpdate)
	if err == nil {
		return result, nil
	}

	// Automatic rollback on failure.
	rbResult, rbErr := e.Rollback(ctx, m, st, statePath, "")
	if rbErr != nil {
		return result, fmt.Errorf("update failed (%w) and automatic rollback also failed: %v", err, rbErr)
	}
	for i := range result.Components {
		for _, rc := range rbResult.Components {
			if rc.Name == result.Components[i].Name {
				result.Components[i].RolledBack = rc.Err == nil
			}
		}
	}
	return result, fmt.Errorf("update failed and was automatically rolled back: %w", err)
}

func (e *Engine) run(ctx context.Context, p *plan.Plan, st *state.State, statePath string, op state.Operation) (*Result, error) {
	result := &Result{}

	for _, cp := range p.Components {
		if cp.Action == plan.ActionSkip {
			result.Components = append(result.Components, ComponentResult{Name: cp.Name, Action: cp.Action})
			continue
		}

		now := e.Now()
		st.BeginOperation(op, cp.Name, now)
		if err := st.Save(statePath); err != nil {
			result.Err = fmt.Errorf("failed to persist in-progress marker for %s: %w", cp.Name, err)
			return result, result.Err
		}

		cr := ComponentResult{Name: cp.Name, Action: cp.Action}

		if err := e.Installer.Install(ctx, cp.Component, e.Paths); err != nil {
			cr.Err = fmt.Errorf("failed to install %s: %w", cp.Name, err)
			result.Components = append(result.Components, cr)
			result.Err = cr.Err
			return result, result.Err
		}

		if err := e.Health.Check(ctx, cp.Component, e.Paths); err != nil {
			cr.Err = fmt.Errorf("health check failed for %s: %w", cp.Name, err)
			result.Components = append(result.Components, cr)
			result.Err = cr.Err
			return result, result.Err
		}

		st.Components[cp.Name] = state.ComponentState{
			Version:     cp.ToVersion,
			InstalledAt: now.UTC().Format(time.RFC3339),
		}
		st.CompleteOperation(op, now)
		if err := st.Save(statePath); err != nil {
			cr.Err = fmt.Errorf("failed to persist state after installing %s: %w", cp.Name, err)
			result.Components = append(result.Components, cr)
			result.Err = cr.Err
			return result, result.Err
		}

		result.Components = append(result.Components, cr)
	}

	st.EcosystemVersion = p.EcosystemVersion
	st.CompleteOperation(op, e.Now())
	if err := st.Save(statePath); err != nil {
		result.Err = fmt.Errorf("failed to persist final state: %w", err)
		return result, result.Err
	}

	return result, nil
}

// Rollback restores components to the previously recorded known-good
// versions. If component is non-empty, only that component is rolled
// back; otherwise every component with a recorded previous version is.
// Rollback re-runs each component's health check against the restored
// version and only reports success for components that actually pass.
func (e *Engine) Rollback(ctx context.Context, m *manifest.Manifest, st *state.State, statePath string, component string) (*Result, error) {
	if !st.HasRollbackTarget() {
		return nil, fmt.Errorf("no rollback target recorded: nothing to roll back to")
	}

	result := &Result{}
	now := e.Now()
	st.BeginOperation(state.OpRollback, component, now)
	if err := st.Save(statePath); err != nil {
		return nil, fmt.Errorf("failed to persist in-progress rollback marker: %w", err)
	}

	targets := st.PreviousComponents
	names := []string{}
	if component != "" {
		if _, ok := targets[component]; !ok {
			return nil, fmt.Errorf("no rollback target recorded for component %q", component)
		}
		names = []string{component}
	} else {
		for name := range targets {
			names = append(names, name)
		}
	}

	var firstErr error
	for _, name := range names {
		target := targets[name]
		c, ok := m.GetComponent(name)
		if !ok {
			continue
		}
		c.Version = target.Version
		if st.Profile == "developer" && c.DeveloperInstall != nil {
			c.Install = *c.DeveloperInstall
		}

		cr := ComponentResult{Name: name, Action: "rollback"}
		if err := e.Installer.Install(ctx, c, e.Paths); err != nil {
			cr.Err = fmt.Errorf("failed to restore %s to %s: %w", name, target.Version, err)
		} else if err := e.Health.Check(ctx, c, e.Paths); err != nil {
			cr.Err = fmt.Errorf("restored %s to %s but health check failed: %w", name, target.Version, err)
		} else {
			st.Components[name] = state.ComponentState{Version: target.Version, InstalledAt: now.UTC().Format(time.RFC3339)}
		}
		if cr.Err != nil && firstErr == nil {
			firstErr = cr.Err
		}
		result.Components = append(result.Components, cr)
	}

	if component == "" && firstErr == nil {
		st.EcosystemVersion = st.PreviousEcosystem
	}
	st.CompleteOperation(state.OpRollback, e.Now())
	if err := st.Save(statePath); err != nil {
		return result, fmt.Errorf("failed to persist state after rollback: %w", err)
	}

	result.Err = firstErr
	return result, firstErr
}
