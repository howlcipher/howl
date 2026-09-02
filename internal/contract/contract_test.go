package contract

import (
	"testing"

	"github.com/howlcipher/howl/internal/discovery"
)

func TestEvaluateContracts(t *testing.T) {
	// 1. All components found with executables -> KNOWN
	discoveredExec := []discovery.DiscoveredComponent{
		{Name: "howlplane", Found: true, ExecutablePath: "/bin/howlplane"},
		{Name: "howlchangeops", Found: true, ExecutablePath: "/bin/howlchangeops"},
		{Name: "howlframe", Found: true, ExecutablePath: "/bin/howlframe"},
		{Name: "howlwriter", Found: true, ExecutablePath: "/bin/howlwriter"},
		{Name: "howlboard", Found: true, ExecutablePath: "/bin/howlboard"},
		{Name: "howlnotes", Found: true, ExecutablePath: "/bin/howlnotes"},
	}

	contracts := EvaluateContracts(discoveredExec)
	if len(contracts) != 6 {
		t.Fatalf("expected 6 contracts, got %d", len(contracts))
	}

	for _, c := range contracts {
		if c.Status != StatusKnown {
			t.Errorf("contract %s expected KNOWN when all executables found, got %s", c.ID, c.Status)
		}
	}

	// 2. All components source-only -> UNVERSIONED
	discoveredSource := []discovery.DiscoveredComponent{
		{Name: "howlplane", Found: true, RepoPath: "/src/howlplane"},
		{Name: "howlchangeops", Found: true, RepoPath: "/src/howlchangeops"},
		{Name: "howlframe", Found: true, RepoPath: "/src/howlframe"},
		{Name: "howlwriter", Found: true, RepoPath: "/src/howlwriter"},
		{Name: "howlboard", Found: true, RepoPath: "/src/howlboard"},
		{Name: "howlnotes", Found: true, RepoPath: "/src/howlnotes"},
	}

	contractsSource := EvaluateContracts(discoveredSource)
	for _, c := range contractsSource {
		if c.Status != StatusUnversioned {
			t.Errorf("contract %s expected UNVERSIONED when sources found without executables, got %s", c.ID, c.Status)
		}
	}

	// 3. Missing callee -> UNKNOWN
	discoveredPartial := []discovery.DiscoveredComponent{
		{Name: "howlplane", Found: true, ExecutablePath: "/bin/howlplane"},
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
