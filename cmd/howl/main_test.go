package main

import (
	"os"
	"strings"
	"testing"
)

// TestGoModHasNoLocalReplaceOrUnpublishedDependency guards against
// reintroducing the exact circular-bootstrap bug this repository was
// rewritten to eliminate: howl requiring another Howl repository's source
// (via a relative go.mod replace directive) in order to build itself.
func TestGoModHasNoLocalReplaceOrUnpublishedDependency(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "replace ") {
		t.Errorf("go.mod contains a replace directive -- howl must never require a local checkout of another Howl repository to build itself:\n%s", content)
	}
	for _, forbidden := range []string{"howlcipher/howlplane", "howlcipher/howlframe", "howlcipher/howlchangeops", "howlcipher/howlwriter"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("go.mod depends on %s -- howl installs ecosystem components, it must not import their Go packages", forbidden)
		}
	}
}
