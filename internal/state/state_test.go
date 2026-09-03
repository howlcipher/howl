package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingStateReturnsFreshState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, existed, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if existed {
		t.Errorf("expected existed=false for missing state file")
	}
	if len(s.Components) != 0 {
		t.Errorf("expected empty components map")
	}
}

func TestLoadCorruptStateReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, existed, err := Load(path)
	if err == nil {
		t.Fatal("expected error loading corrupt state")
	}
	if !existed {
		t.Errorf("expected existed=true for a present-but-corrupt file")
	}
}

func TestLoadUnsupportedSchemaVersionReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":999,"components":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected error loading unsupported schema version")
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	s := New("stable")
	s.Components["howlframe"] = ComponentState{Version: "0.1.1", InstalledAt: "2026-01-01T00:00:00Z"}
	s.EcosystemVersion = "0.1.0"

	if err := s.Save(path); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	loaded, existed, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if !existed {
		t.Fatal("expected existed=true")
	}
	if loaded.EcosystemVersion != "0.1.0" {
		t.Errorf("expected ecosystem version to round-trip")
	}
	got, ok := loaded.Components["howlframe"]
	if !ok || got.Version != "0.1.1" {
		t.Errorf("expected howlframe component state to round-trip, got %+v", loaded.Components)
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s := New("stable")
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Errorf("expected only state.json in directory after save, found leftover %s", e.Name())
		}
	}
}

func TestBeginAndCompleteOperation(t *testing.T) {
	s := New("stable")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.BeginOperation(OpInstall, "howlframe", now)

	if s.InProgress == nil || s.InProgress.Operation != OpInstall || s.InProgress.Component != "howlframe" {
		t.Fatalf("expected in-progress marker, got %+v", s.InProgress)
	}

	s.CompleteOperation(OpInstall, now)
	if s.InProgress != nil {
		t.Errorf("expected in-progress marker cleared, got %+v", s.InProgress)
	}
	if s.LastOperation != string(OpInstall) {
		t.Errorf("expected last operation recorded")
	}
}

func TestSnapshotForRollbackAndHasTarget(t *testing.T) {
	s := New("stable")
	if s.HasRollbackTarget() {
		t.Fatal("expected no rollback target on a fresh state")
	}

	s.Components["howlframe"] = ComponentState{Version: "0.1.0"}
	s.EcosystemVersion = "1.0.0"
	s.SnapshotForRollback("1.1.0")

	if !s.HasRollbackTarget() {
		t.Fatal("expected rollback target after snapshot")
	}
	if s.PreviousEcosystem != "1.0.0" {
		t.Errorf("expected previous ecosystem version 1.0.0, got %s", s.PreviousEcosystem)
	}
	if s.EcosystemVersion != "1.1.0" {
		t.Errorf("expected new ecosystem version 1.1.0, got %s", s.EcosystemVersion)
	}
	prev, ok := s.PreviousComponents["howlframe"]
	if !ok || prev.Version != "0.1.0" {
		t.Errorf("expected previous component snapshot to retain 0.1.0, got %+v", s.PreviousComponents)
	}
}
