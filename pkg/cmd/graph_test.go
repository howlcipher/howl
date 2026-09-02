package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/graph"
)

func TestGraphCommand(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"graph"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected graph error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "HOWL ECOSYSTEM ARCHITECTURAL GRAPH") {
		t.Errorf("expected graph header in text output, got:\n%s", out)
	}
}

func TestGraphMermaidCommand(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"graph", "--mermaid"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected graph --mermaid error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "```mermaid") {
		t.Errorf("expected ```mermaid in output, got:\n%s", out)
	}
}

func TestGraphJSONCommand(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"graph", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected graph --json error: %v", err)
	}

	var data graph.GraphData
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse graph JSON: %v", err)
	}

	if len(data.Nodes) == 0 || len(data.Edges) == 0 {
		t.Errorf("expected non-empty graph data")
	}
}
