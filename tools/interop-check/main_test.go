package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type stubRunner map[string]string

func (s stubRunner) Describe(_ context.Context, bin string) ([]byte, error) {
	out, ok := s[bin]
	if !ok {
		return nil, errors.New("exit status 2")
	}
	return []byte(out), nil
}

const plane = `{"schema":"howl.interop/v1","component":"howlplane","version":"0.2.0",
  "commands":[{"path":["orchestrate"],"flags":[],"json":false},{"path":["agents","doctor"],"flags":["--json"],"json":true},
  {"path":["factory","prepare"],"flags":[],"json":false}],"exit_codes":{"0":"ok"},
  "consumes":[{"component":"howlchangeops","command":["evaluate"],"flags":["--json"]}]}`

func TestCheckReportsEachDescriptorAndEdge(t *testing.T) {
	locate := func(name string) (string, bool) {
		if name == "howlnotes" {
			return "", false
		}
		return "/bin/" + name, true
	}
	rep := check(context.Background(), []string{"howlplane", "howlchangeops", "howlnotes"}, nil, locate,
		stubRunner{"/bin/howlplane": plane}, false, nil)

	status := map[string]string{}
	for _, d := range rep.Descriptors {
		status[d.Component] = d.Status
	}
	if status["howlplane"] != "ok" || status["howlchangeops"] != "warn" || status["howlnotes"] != "skip" {
		t.Fatalf("descriptor statuses: %v", status)
	}
	edges := map[string]string{}
	for _, e := range rep.Edges {
		edges[e.Consumer+">"+e.Producer] = string(e.Severity)
	}
	// howl -> howlplane adapters resolve; howlplane -> howlchangeops cannot be
	// verified without a descriptor; howl -> howlframe is absent here.
	if edges["howl>howlplane"] != "ok" || edges["howlplane>howlchangeops"] != "warn" {
		t.Fatalf("edges: %v", edges)
	}
	if rep.Failed {
		t.Fatal("warnings alone must not fail without --strict")
	}
	if !check(context.Background(), []string{"howlplane", "howlchangeops"}, nil, locate, stubRunner{"/bin/howlplane": plane}, true, nil).Failed {
		t.Fatal("--strict must fail on warnings")
	}
}

func TestBinOverrideWins(t *testing.T) {
	rep := check(context.Background(), []string{"howlplane"}, map[string]string{"howlplane": "/custom/plane"},
		func(string) (string, bool) { return "", false }, stubRunner{"/custom/plane": plane}, false, nil)
	if rep.Descriptors[0].Path != "/custom/plane" || rep.Descriptors[0].Status != "ok" {
		t.Fatalf("got %+v", rep.Descriptors[0])
	}
}

func TestRequiredDescriptorCLI(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	const missing = "howl-checker-test-missing"
	for _, tc := range []struct {
		name, required, status string
		exit                   int
	}{
		{"optional is skipped", "", "skip", 0},
		{"required is rejected", missing, "fail", 1},
		{"unselected is a usage error", "not-selected", "", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := []string{"--json", "--strict", "--components", missing}
			if tc.required != "" {
				args = append(args, "--require-descriptors", tc.required)
			}
			if got := run(args, &stdout, &stderr); got != tc.exit {
				t.Fatalf("exit %d, want %d, stderr %s", got, tc.exit, &stderr)
			}
			if tc.exit == 2 {
				if !strings.Contains(stderr.String(), tc.required) {
					t.Fatalf("usage error does not name component: %s", &stderr)
				}
				return
			}
			var result report
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Descriptors) != 1 || result.Descriptors[0].Status != tc.status || result.Failed != (tc.exit == 1) {
				t.Fatalf("unexpected report: %+v", result)
			}
		})
	}
}

func TestRequiredDescriptorWithOptionalMissingProducer(t *testing.T) {
	names := []string{"howlplane", "howlnotes"}
	locate := func(name string) (string, bool) { return name, name == "howlplane" }
	required := map[string]bool{"howlplane": true}
	for _, tc := range []struct {
		runner stubRunner
		status string
		failed bool
	}{
		{stubRunner{"howlplane": plane}, "ok", false},
		{stubRunner{}, "fail", true},
	} {
		result := check(context.Background(), names, nil, locate, tc.runner, false, required)
		if result.Descriptors[0].Status != tc.status || result.Descriptors[1].Status != "skip" || result.Failed != tc.failed {
			t.Fatalf("unexpected result: %+v", result)
		}
	}
}
