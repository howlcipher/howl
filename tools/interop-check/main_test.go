package main

import (
	"context"
	"errors"
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
		stubRunner{"/bin/howlplane": plane}, false)

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
	if !check(context.Background(), []string{"howlplane", "howlchangeops"}, nil, locate, stubRunner{"/bin/howlplane": plane}, true).Failed {
		t.Fatal("--strict must fail on warnings")
	}
}

func TestBinOverrideWins(t *testing.T) {
	rep := check(context.Background(), []string{"howlplane"}, map[string]string{"howlplane": "/custom/plane"},
		func(string) (string, bool) { return "", false }, stubRunner{"/custom/plane": plane}, false)
	if rep.Descriptors[0].Path != "/custom/plane" || rep.Descriptors[0].Status != "ok" {
		t.Fatalf("got %+v", rep.Descriptors[0])
	}
}
