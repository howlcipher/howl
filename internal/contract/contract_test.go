package contract

import (
	"testing"

	"github.com/howlcipher/howl/internal/discovery"
)

func TestEvaluateContracts(t *testing.T) {
	discoveredAll := []discovery.DiscoveredComponent{
		{Name: "howlplane", Found: true},
		{Name: "howlchangeops", Found: true},
		{Name: "howlframe", Found: true},
		{Name: "howlwriter", Found: true},
		{Name: "howlboard", Found: true},
		{Name: "howlnotes", Found: true},
	}

	contracts := EvaluateContracts(discoveredAll)
	if len(contracts) != 6 {
		t.Fatalf("expected 6 contracts, got %d", len(contracts))
	}

	for _, c := range contracts {
		if c.Status != StatusUnversioned {
			t.Errorf("contract %s expected UNVERSIONED when all components found, got %s", c.ID, c.Status)
		}
	}

	discoveredPartial := []discovery.DiscoveredComponent{
		{Name: "howlplane", Found: true},
		{Name: "howlchangeops", Found: false},
	}
	contractsPartial := EvaluateContracts(discoveredPartial)
	foundUnknown := false
	for _, c := range contractsPartial {
		if c.ID == "PLANE-CHANGEOPS-AUTH" {
			if c.Status != StatusUnknown {
				t.Errorf("expected UNKNOWN for missing callee, got %s", c.Status)
			}
			foundUnknown = true
		}
	}
	if !foundUnknown {
		t.Errorf("PLANE-CHANGEOPS-AUTH not found in evaluated contracts")
	}
}
