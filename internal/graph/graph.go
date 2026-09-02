// Package graph provides rendering of the Howl ecosystem architectural relationship graph.
package graph

import (
	"encoding/json"
	"fmt"
	"io"
)

// GraphNode represents a node in the ecosystem architecture.
type GraphNode struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Role        string `json:"role"`
	Category    string `json:"category"`
	Repository  string `json:"repository,omitempty"`
}

// GraphEdge represents a directed relationship between architectural nodes.
type GraphEdge struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// GraphData holds the full nodes and edges structure.
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// DefaultGraph returns the canonical architectural graph of the Howl ecosystem.
func DefaultGraph() GraphData {
	return GraphData{
		Nodes: []GraphNode{
			{ID: "human", Label: "Human Operator", Role: "Sovereign Authority", Category: "authority"},
			{ID: "howl", Label: "Howl CLI", Role: "Ecosystem Front Door & Entry Point", Category: "cli", Repository: "https://github.com/howlcipher/howl"},
			{ID: "howlplane", Label: "HowlPlane", Role: "AI Engineering Control Plane", Category: "control", Repository: "https://github.com/howlcipher/howlplane"},
			{ID: "howlchangeops", Label: "HowlChangeOps", Role: "Authority Boundary & Release Controller", Category: "gate", Repository: "https://github.com/howlcipher/howlchangeops"},
			{ID: "howlframe", Label: "HowlFrame", Role: "Language, HFIR Gate & VM", Category: "runtime", Repository: "https://github.com/howlcipher/howlframe"},
			{ID: "howlwriter", Label: "HowlWriter", Role: "Writing Control & Review System", Category: "domain", Repository: "https://github.com/howlcipher/howlwriter"},
			{ID: "howlboard", Label: "HowlBoard", Role: "Telemetry Console & Evaluation", Category: "domain", Repository: "https://github.com/howlcipher/howlboard"},
			{ID: "howlnotes", Label: "HowlNotes", Role: "Knowledge Notebook & Dogfood", Category: "application", Repository: "https://github.com/howlcipher/howlnotes"},
		},
		Edges: []GraphEdge{
			{From: "human", To: "howl", Label: "CLI Command Invocations", Description: "Operator invokes unprivileged CLI commands, diagnostics, and routing"},
			{From: "human", To: "howlchangeops", Label: "Cryptographic HMAC Approval", Description: "Human retains sovereign authority over consequential mutations"},
			{From: "howl", To: "howlplane", Label: "Command Routing & Orchestration", Description: "High-level delegation to AI engineering control plane"},
			{From: "howl", To: "howlframe", Label: "DSL / VM Tooling", Description: "Compiler and verification access"},
			{From: "howl", To: "howlchangeops", Label: "Authority Coordination", Description: "Release and mutation verification"},
			{From: "howlplane", To: "howlchangeops", Label: "Bounded Release Action", Description: "Delegates consequential actions to HMAC approval gate"},
			{From: "howlplane", To: "howlframe", Label: "HFIR Gate & VM Verification", Description: "Verifies DSL outputs and runs deterministic tests"},
			{From: "howlplane", To: "howlwriter", Label: "Prose & Review Routing", Description: "Routes writing tasks, style linting, and voice review"},
			{From: "howlplane", To: "howlboard", Label: "Telemetry & State Evaluation", Description: "Exports task state machines and evaluation metrics"},
			{From: "howlframe", To: "howlnotes", Label: "AI DSL Browser VM & Storage", Description: "Dogfood consumer for HowlFrame compilation"},
		},
	}
}

// RenderText writes the human-readable hierarchical tree diagram.
func RenderText(w io.Writer) {
	fmt.Fprint(w, `HOWL ECOSYSTEM ARCHITECTURAL GRAPH

[CLI ROUTING ARCHITECTURE]
Human Operator / Developer
  │
  ▼
Howl CLI (Ecosystem Front Door & Entry Point)
  │
  ├── HowlPlane (AI Engineering Control Plane)
  │     ├── AI Providers & Resource Pool
  │     ├── Multi-Agent Task Routing & Evidence Ledgers
  │     ├── HowlWriter (Prose, Voice Preservation & Review)
  │     └── HowlBoard (Telemetry Console & Evaluation)
  │
  └── HowlFrame (Language, HFIR Gate & Capability-Bounded VM)
        └── HowlNotes (Knowledge Notebook & Dogfood Consumer)

[CONSEQUENTIAL AUTHORITY FLOW]
HowlPlane / Autonomous Workflows ──► Proposed Consequential Action
                                          │
                                          ▼
                                    HowlChangeOps (Release Controller)
                                          ▲
                                          │ [Cryptographic HMAC Signature]
                                    Human Operator (Sovereign Authority)
`)
}

// RenderMermaid writes a GitHub-flavored Mermaid graph.
func RenderMermaid(w io.Writer, data GraphData) {
	fmt.Fprintln(w, "```mermaid")
	fmt.Fprintln(w, "graph TD")
	for _, n := range data.Nodes {
		fmt.Fprintf(w, "    %s[\"%s<br/><i>%s</i>\"]\n", n.ID, n.Label, n.Role)
	}
	fmt.Fprintln(w, "")
	for _, e := range data.Edges {
		fmt.Fprintf(w, "    %s -->|\"%s\"| %s\n", e.From, e.Label, e.To)
	}
	fmt.Fprintln(w, "```")
}

// RenderJSON writes the structured graph as JSON.
func RenderJSON(w io.Writer, data GraphData) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
