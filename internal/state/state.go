// Package state persists Howl's installer state: what's installed, what
// version, the previous known-good release for rollback, and whether a
// lifecycle operation was left interrupted. This is the only durable
// record Howl keeps about the machine it's managing.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CurrentSchemaVersion is the state file schema version this build of Howl
// writes and expects to read.
const CurrentSchemaVersion = 1

// Operation identifies a lifecycle operation that mutates installed state.
type Operation string

const (
	OpInstall   Operation = "install"
	OpUpdate    Operation = "update"
	OpRollback  Operation = "rollback"
	OpUninstall Operation = "uninstall"
	OpDoctorFix Operation = "doctor_fix"
)

// ComponentState records what's actually installed for one component.
type ComponentState struct {
	Version     string `json:"version"`
	InstalledAt string `json:"installed_at"`
}

// InProgress records a lifecycle operation that has started but not yet
// been marked complete, so `howl doctor` can recognize an interrupted
// operation left behind by a crash or a killed process.
type InProgress struct {
	Operation Operation `json:"operation"`
	Component string    `json:"component,omitempty"`
	StartedAt string    `json:"started_at"`
}

// State is Howl's complete durable installer state.
type State struct {
	SchemaVersion    int    `json:"schema_version"`
	InstallerVersion string `json:"installer_version"`
	Channel          string `json:"channel"`
	EcosystemVersion string `json:"ecosystem_version,omitempty"`

	Components         map[string]ComponentState `json:"components"`
	PreviousEcosystem  string                    `json:"previous_ecosystem_version,omitempty"`
	PreviousComponents map[string]ComponentState `json:"previous_components,omitempty"`

	InProgress *InProgress `json:"in_progress,omitempty"`

	LastOperation   string `json:"last_operation,omitempty"`
	LastOperationAt string `json:"last_operation_at,omitempty"`
}

// New returns a fresh, empty state for a first-time installation.
func New(channel string) *State {
	return &State{
		SchemaVersion: CurrentSchemaVersion,
		Channel:       channel,
		Components:    map[string]ComponentState{},
	}
}

// Load reads state from path. A missing file is not an error: it returns a
// fresh, empty state with ok=false so callers can distinguish "no prior
// installation" from "corrupt state file".
func Load(path string) (s *State, existed bool, err error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(""), false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read installer state at %s: %w", path, err)
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, true, fmt.Errorf("installer state at %s is corrupt: %w", path, err)
	}
	if st.SchemaVersion != CurrentSchemaVersion {
		return nil, true, fmt.Errorf("installer state at %s has unsupported schema_version %d (expected %d)", path, st.SchemaVersion, CurrentSchemaVersion)
	}
	if st.Components == nil {
		st.Components = map[string]ComponentState{}
	}
	return &st, true, nil
}

// Save writes state to path atomically: it writes to a temp file in the
// same directory and renames over the destination, so a crash mid-write
// never leaves a truncated or partially-written state file.
func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal installer state: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary state file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary state file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to activate state file: %w", err)
	}
	return nil
}

// BeginOperation records that a lifecycle operation has started, so an
// interrupted run is detectable by `howl doctor` even if the process never
// gets to call CompleteOperation.
func (s *State) BeginOperation(op Operation, component string, now time.Time) {
	s.InProgress = &InProgress{
		Operation: op,
		Component: component,
		StartedAt: now.UTC().Format(time.RFC3339),
	}
}

// CompleteOperation clears the in-progress marker and records the
// operation as the last completed one.
func (s *State) CompleteOperation(op Operation, now time.Time) {
	s.InProgress = nil
	s.LastOperation = string(op)
	s.LastOperationAt = now.UTC().Format(time.RFC3339)
}

// SnapshotForRollback copies the current component versions into the
// previous-known-good slot, before an update mutates Components. Only one
// previous release is retained, per v1's scope.
func (s *State) SnapshotForRollback(ecosystemVersion string) {
	s.PreviousEcosystem = s.EcosystemVersion
	prev := make(map[string]ComponentState, len(s.Components))
	for k, v := range s.Components {
		prev[k] = v
	}
	s.PreviousComponents = prev
	s.EcosystemVersion = ecosystemVersion
}

// HasRollbackTarget reports whether a previous known-good release was
// recorded.
func (s *State) HasRollbackTarget() bool {
	return len(s.PreviousComponents) > 0
}
