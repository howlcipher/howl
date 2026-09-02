package graph

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDefaultGraph(t *testing.T) {
	g := DefaultGraph()
	if len(g.Nodes) == 0 {
		t.Errorf("expected nodes in graph")
	}
	if len(g.Edges) == 0 {
		t.Errorf("expected edges in graph")
	}

	// Test Text render
	var textBuf bytes.Buffer
	RenderText(&textBuf)
	if !bytes.Contains(textBuf.Bytes(), []byte("HOWL ECOSYSTEM ARCHITECTURAL GRAPH")) {
		t.Errorf("missing header in text graph")
	}

	// Test Mermaid render
	var mermaidBuf bytes.Buffer
	RenderMermaid(&mermaidBuf, g)
	if !bytes.Contains(mermaidBuf.Bytes(), []byte("graph TD")) {
		t.Errorf("missing graph TD in mermaid graph")
	}

	// Test JSON render
	var jsonBuf bytes.Buffer
	if err := RenderJSON(&jsonBuf, g); err != nil {
		t.Fatalf("JSON render error: %v", err)
	}
	var parsed GraphData
	if err := json.Unmarshal(jsonBuf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(parsed.Nodes) != len(g.Nodes) {
		t.Errorf("expected %d nodes in JSON, got %d", len(g.Nodes), len(parsed.Nodes))
	}
}
