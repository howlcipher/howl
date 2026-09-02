// Package contract defines and evaluates cross-component architectural contracts in the Howl ecosystem.
package contract

import (
	"github.com/howlcipher/howl/internal/discovery"
)

// Status represents the verified condition of an architectural contract.
type Status string

const (
	StatusKnown        Status = "KNOWN"
	StatusUnversioned  Status = "UNVERSIONED"
	StatusUnknown      Status = "UNKNOWN"
	StatusIncompatible Status = "INCOMPATIBLE"
)

// Contract represents an integration contract between two ecosystem components.
type Contract struct {
	ID          string `json:"id"`
	Caller      string `json:"caller"`
	Callee      string `json:"callee"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      Status `json:"status"`
	Details     string `json:"details,omitempty"`
}

// DefaultContracts returns the canonical list of known ecosystem integration contracts.
func DefaultContracts() []Contract {
	return []Contract{
		{
			ID:          "HOWL-PLANE-CLI",
			Caller:      "howl",
			Callee:      "howlplane",
			Name:        "CLI Command Delegation & Forwarding",
			Description: "Canonical CLI subprocess delegation and project validation forwarding to howlplane binary",
		},
		{
			ID:          "PLANE-CHANGEOPS-AUTH",
			Caller:      "howlplane",
			Callee:      "howlchangeops",
			Name:        "HMAC Cryptographic Human Authority Gate",
			Description: "Cryptographic human approvals and bounded Git mutations for consequential actions",
		},
		{
			ID:          "PLANE-FRAME-VERIFY",
			Caller:      "howlplane",
			Callee:      "howlframe",
			Name:        "HFIR Verification & VM Gate",
			Description: "Adversarial verification, AI DSL compilation, and capability-bounded VM execution",
		},
		{
			ID:          "PLANE-WRITER-REVIEW",
			Caller:      "howlplane",
			Callee:      "howlwriter",
			Name:        "Prose, Style & Voice Review",
			Description: "Voice preservation, style linting, humanization, and prose review",
		},
		{
			ID:          "PLANE-BOARD-TELEMETRY",
			Caller:      "howlplane",
			Callee:      "howlboard",
			Name:        "Telemetry Presentation & Evaluation",
			Description: "Telemetry console and state machine evaluation",
		},
		{
			ID:          "FRAME-NOTES-RUNTIME",
			Caller:      "howlframe",
			Callee:      "howlnotes",
			Name:        "AI DSL Browser VM & Storage Consumer",
			Description: "Browser compilation and native store persistence",
		},
	}
}

// EvaluateContracts evaluates contract availability based on discovered components.
func EvaluateContracts(discovered []discovery.DiscoveredComponent) []Contract {
	compMap := make(map[string]discovery.DiscoveredComponent)
	for _, c := range discovered {
		compMap[c.Name] = c
	}

	contracts := DefaultContracts()
	results := make([]Contract, 0, len(contracts))

	for _, c := range contracts {
		callerFound := (c.Caller == "howl")
		var callerComp discovery.DiscoveredComponent
		if !callerFound {
			callerComp, callerFound = compMap[c.Caller]
			callerFound = callerFound && callerComp.Found
		}

		calleeComp, calleeFound := compMap[c.Callee]
		calleeFound = calleeFound && calleeComp.Found

		eval := c
		if callerFound && calleeFound {
			if c.Caller == "howl" {
				if calleeComp.ExecutablePath != "" {
					eval.Status = StatusKnown
					eval.Details = "Subprocess delegation endpoint available (" + calleeComp.ExecutablePath + ")"
				} else {
					eval.Status = StatusUnversioned
					eval.Details = "Source repository discovered; executable not built"
				}
			} else {
				if callerComp.ExecutablePath != "" && calleeComp.ExecutablePath != "" {
					eval.Status = StatusKnown
					eval.Details = "Component executables discovered"
				} else {
					eval.Status = StatusUnversioned
					eval.Details = "Integration endpoints available (unversioned contract)"
				}
			}
		} else if !calleeFound {
			eval.Status = StatusUnknown
			eval.Details = "Callee component '" + c.Callee + "' not discovered"
		} else {
			eval.Status = StatusUnknown
			eval.Details = "Caller component '" + c.Caller + "' not discovered"
		}
		results = append(results, eval)
	}

	return results
}
