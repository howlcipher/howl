package plane

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOSExecRunner(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "fake_howlplane")
	scriptContent := `#!/bin/sh
if [ "$1" = "exit7" ]; then
    echo "failing with 7" >&2
    exit 7
fi
echo "hello from $1"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	runner := &OSExecRunner{}
	var stdout, stderr bytes.Buffer

	// Test success
	code, err := runner.Run(context.Background(), scriptPath, []string{"world"}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "hello from world\n" {
		t.Errorf("expected stdout 'hello from world\\n', got %q", stdout.String())
	}

	// Test non-zero exit code preservation
	stdout.Reset()
	stderr.Reset()
	code, err = runner.Run(context.Background(), scriptPath, []string{"exit7"}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}
	if code != 7 {
		t.Errorf("expected exit code 7, got %d", code)
	}
	if stderr.String() != "failing with 7\n" {
		t.Errorf("expected stderr 'failing with 7\\n', got %q", stderr.String())
	}
}
