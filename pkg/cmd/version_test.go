package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/version"
)

func TestVersionCommand(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "howl version "+version.Version) {
		t.Errorf("expected version output to contain 'howl version %s', got:\n%s", version.Version, out)
	}
}

func TestVersionJSONCommand(t *testing.T) {
	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"version", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version --json command failed: %v", err)
	}

	var info version.Info
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("failed to unmarshal version json: %v", err)
	}

	if info.Version != version.Version {
		t.Errorf("expected version %s, got %s", version.Version, info.Version)
	}
}
