// Package plan builds an inspectable, typed installation/update plan from
// a release manifest and the current installer state, before anything
// touches the filesystem. Execution (internal/engine) always consumes a
// Plan that was rendered and, outside --yes, approved first.
package plan

import (
	"fmt"
	"io"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/state"
)

// Action describes what will happen to a single component.
type Action string

const (
	ActionInstall Action = "install"
	ActionUpdate  Action = "update"
	ActionSkip    Action = "already installed"
)

// ComponentPlan is the resolved action for one component.
type ComponentPlan struct {
	Name        string
	DisplayName string
	Action      Action
	FromVersion string // empty if not currently installed
	ToVersion   string
	Component   manifest.Component
}

// DependencyCheck is the resolved status of one required or optional
// dependency (a component's own external_dependencies, or an
// ecosystem-level optional capability).
type DependencyCheck struct {
	Name       string
	Required   bool
	Capability string
	Found      bool
	Detail     string
}

// Plan is the complete, human-inspectable set of actions Howl intends to
// take. Nothing is mutated by building a Plan.
type Plan struct {
	Profile          string
	Channel          string
	EcosystemVersion string
	Components       []ComponentPlan
	Dependencies     []DependencyCheck
	Warnings         []string
}

// HasWork reports whether the plan does anything at all.
func (p *Plan) HasWork() bool {
	for _, c := range p.Components {
		if c.Action != ActionSkip {
			return true
		}
	}
	return false
}

// MissingRequired reports the required dependencies that were not found.
func (p *Plan) MissingRequired() []DependencyCheck {
	var out []DependencyCheck
	for _, d := range p.Dependencies {
		if d.Required && !d.Found {
			out = append(out, d)
		}
	}
	return out
}

// Detector probes the local machine for an external dependency's
// availability. Implementations are injected so planning is testable
// without depending on what happens to be installed on the test machine.
type Detector interface {
	Detect(name string) (found bool, versionInfo string)
}

// BuildOptions configures how a Plan is constructed.
type BuildOptions struct {
	// Profile is one of "standard", "local-ai", or "developer". Only
	// "local-ai" currently changes behavior: it adds the manifest's
	// optional_capabilities to the dependency check list.
	Profile string
	// Components restricts planning to specific component names. Empty
	// means every component in the manifest.
	Components []string
	Detector   Detector
}

// Build resolves a manifest and the current state into an ordered,
// typed Plan.
func Build(m *manifest.Manifest, st *state.State, opts BuildOptions) (*Plan, error) {
	order, err := m.TopoOrder()
	if err != nil {
		return nil, err
	}

	scope := map[string]bool{}
	if len(opts.Components) == 0 {
		for _, name := range order {
			scope[name] = true
		}
	} else {
		for _, name := range opts.Components {
			if _, ok := m.GetComponent(name); !ok {
				return nil, fmt.Errorf("unknown component %q", name)
			}
			scope[name] = true
		}
	}

	p := &Plan{
		Profile:          opts.Profile,
		Channel:          m.Ecosystem.Channel,
		EcosystemVersion: m.Ecosystem.Version,
	}

	seenDep := map[string]bool{}
	for _, name := range order {
		if !scope[name] {
			continue
		}
		c, _ := m.GetComponent(name)

		cp := ComponentPlan{
			Name:        c.Name,
			DisplayName: displayName(c),
			ToVersion:   c.Version,
			Component:   c,
		}
		if existing, ok := st.Components[c.Name]; ok {
			cp.FromVersion = existing.Version
			if existing.Version == c.Version {
				cp.Action = ActionSkip
			} else {
				cp.Action = ActionUpdate
			}
		} else {
			cp.Action = ActionInstall
		}
		p.Components = append(p.Components, cp)

		if cp.Action == ActionSkip {
			continue
		}
		for _, ext := range c.ExternalDependencies {
			key := ext.Name
			if seenDep[key] {
				continue
			}
			seenDep[key] = true
			p.Dependencies = append(p.Dependencies, resolveDependency(opts.Detector, ext.Name, ext.Required, ext.Capability))
		}
	}

	if opts.Profile == "local-ai" {
		for _, cap := range m.OptionalCapabilities {
			if seenDep[cap.Name] {
				continue
			}
			seenDep[cap.Name] = true
			p.Dependencies = append(p.Dependencies, resolveDependency(opts.Detector, cap.Name, cap.Required, cap.Capability))
		}
	}

	for _, d := range p.MissingRequired() {
		p.Warnings = append(p.Warnings, fmt.Sprintf("required dependency %q not found (%s)", d.Name, d.Capability))
	}

	return p, nil
}

func resolveDependency(det Detector, name string, required bool, capability string) DependencyCheck {
	dc := DependencyCheck{Name: name, Required: required, Capability: capability}
	if det == nil {
		return dc
	}
	found, detail := det.Detect(name)
	dc.Found = found
	dc.Detail = detail
	return dc
}

func displayName(c manifest.Component) string {
	if c.DisplayName != "" {
		return c.DisplayName
	}
	return c.Name
}

// Render writes a human-readable summary of the plan, in the shape a user
// approves before Howl touches their machine.
func Render(w io.Writer, p *Plan) {
	fmt.Fprintf(w, "Howl Ecosystem %s\n\n", p.EcosystemVersion)
	fmt.Fprintf(w, "Profile\n  %s\n\n", p.Profile)

	fmt.Fprintf(w, "Components\n")
	for _, c := range p.Components {
		line := string(c.Action)
		switch c.Action {
		case ActionUpdate:
			line = fmt.Sprintf("update %s -> %s", c.FromVersion, c.ToVersion)
		case ActionInstall:
			line = fmt.Sprintf("install %s", c.ToVersion)
		}
		fmt.Fprintf(w, "  %-16s %s\n", c.DisplayName, line)
	}

	if len(p.Dependencies) > 0 {
		fmt.Fprintf(w, "\nDependencies\n")
		for _, d := range p.Dependencies {
			status := "found"
			if !d.Found {
				status = "not found"
				if !d.Required {
					status += fmt.Sprintf(" (optional: %s unavailable)", d.Capability)
				}
			} else if d.Detail != "" {
				status = d.Detail
			}
			req := "optional"
			if d.Required {
				req = "required"
			}
			fmt.Fprintf(w, "  %-16s %-9s %s\n", d.Name, req, status)
		}
	}

	if len(p.Warnings) > 0 {
		fmt.Fprintf(w, "\nWarnings\n")
		for _, warn := range p.Warnings {
			fmt.Fprintf(w, "  - %s\n", warn)
		}
	}
}
