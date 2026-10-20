package interop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/platform"
)

const frameDescriptor = `{
  "schema": "howl.interop/v1",
  "component": "howlframe",
  "version": "0.2.0",
  "commands": [
    {"path": [], "flags": ["--version"], "json": false},
    {"path": ["run"], "flags": ["-allow-caps", "--max-instructions", "--receipt"], "json": false, "emits": ["howlframe.receipt/v0"]}
  ],
  "exit_codes": {"0": "ok", "2": "bad resource flag"},
  "env": ["HOWLFRAME_HOME"]
}`

func consumer(uses ...Consume) *Descriptor {
	return &Descriptor{
		Schema: DescriptorSchema, Component: "howlchangeops", Version: "0.2.0",
		Commands: []Command{}, ExitCodes: map[string]string{"0": "ok"}, Consumes: uses,
	}
}

func mustParse(t *testing.T, s string) *Descriptor {
	t.Helper()
	d, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func TestParseAcceptsAValidDescriptor(t *testing.T) {
	d := mustParse(t, frameDescriptor)
	if d.Component != "howlframe" || len(d.Commands) != 2 {
		t.Fatalf("unexpected descriptor %+v", d)
	}
	if _, ok := d.Lookup([]string{"run"}); !ok {
		t.Fatal("run command not found")
	}
	if _, ok := d.Lookup(nil); !ok {
		t.Fatal("global flags (empty path) not found")
	}
}

func TestParseAcceptsSemverAndPEP440Versions(t *testing.T) {
	for _, v := range []string{"0.2.0", "v1.0.0", "0.2.0-rc1", "0.2.0b1", "0.2.0.dev3", "0.0.0-dev", "1.2.3+build.7"} {
		doc := `{"schema":"howl.interop/v1","component":"x","version":"` + v + `","commands":[],"exit_codes":{"0":"ok"}}`
		if _, err := Parse([]byte(doc)); err != nil {
			t.Errorf("version %q rejected: %v", v, err)
		}
	}
	producer := mustParse(t, strings.Replace(frameDescriptor, `"0.2.0"`, `"0.2.0b1"`, 1))
	f := Resolve(map[string]*Descriptor{"howlframe": producer, "howlchangeops": consumer(
		Consume{Component: "howlframe", Command: []string{"run"}, MinVersion: "0.2.0"},
		Consume{Component: "howlframe", Command: []string{"run"}, MinVersion: "0.2.1"},
	)}, nil)
	if f[0].Severity != SeverityOK || f[1].Severity != SeverityFail {
		t.Fatalf("pre-release comparison: %+v", f)
	}
}

func TestParseRejectsMalformedDescriptors(t *testing.T) {
	cases := map[string]string{
		"wrong schema":       `{"schema":"howl.interop/v2","component":"x","version":"1.0.0","commands":[],"exit_codes":{"0":"ok"}}`,
		"bad version":        `{"schema":"howl.interop/v1","component":"x","version":"one","commands":[],"exit_codes":{"0":"ok"}}`,
		"no exit codes":      `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[]}`,
		"missing commands":   `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","exit_codes":{"0":"ok"}}`,
		"bad flag":           `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[{"path":["a"],"flags":["a b"],"json":false}],"exit_codes":{"0":"ok"}}`,
		"bad schema id":      `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[{"path":["a"],"flags":[],"json":true,"emits":["Result"]}],"exit_codes":{"0":"ok"}}`,
		"duplicate command":  `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[{"path":["a"],"flags":[],"json":false},{"path":["a"],"flags":[],"json":false}],"exit_codes":{"0":"ok"}}`,
		"non-numeric code":   `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[],"exit_codes":{"ok":"ok"}}`,
		"uppercase name":     `{"schema":"howl.interop/v1","component":"HowlX","version":"1.0.0","commands":[],"exit_codes":{"0":"ok"}}`,
		"bad consume flag":   `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[],"exit_codes":{"0":"ok"},"consumes":[{"component":"y","command":["a"],"flags":["!"]}]}`,
		"bad consume minver": `{"schema":"howl.interop/v1","component":"x","version":"1.0.0","commands":[],"exit_codes":{"0":"ok"},"consumes":[{"component":"y","command":["a"],"min_version":"latest"}]}`,
		"not json":           `usage: x [-h]`,
	}
	for name, doc := range cases {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("%s: Parse accepted %s", name, doc)
		}
	}
}

func TestResolveGradesEachEdge(t *testing.T) {
	frame := mustParse(t, frameDescriptor)
	descs := map[string]*Descriptor{
		"howlframe": frame,
		"howlchangeops": consumer(
			Consume{Component: "howlframe", Command: []string{"run"}, Flags: []string{"--allow-caps"}, MinVersion: "0.1.0"},
			Consume{Component: "howlframe", Command: []string{"build"}},
			Consume{Component: "howlframe", Command: []string{"run"}, Flags: []string{"--deadline"}},
			Consume{Component: "howlframe", Command: []string{"run"}, Schemas: []string{"howlframe.receipt/v1"}},
			Consume{Component: "howlframe", Command: []string{"run"}, MinVersion: "0.3.0"},
			Consume{Component: "howldream", Command: []string{"explore"}},
			Consume{Component: "howlplane", Command: []string{"reconcile"}},
		),
	}
	findings := Resolve(descs, map[string]string{"howlplane": "howlplane publishes no interop descriptor"})
	want := []Severity{SeverityOK, SeverityFail, SeverityFail, SeverityFail, SeverityFail, SeveritySkip, SeverityWarn}
	if len(findings) != len(want) {
		t.Fatalf("got %d findings, want %d: %+v", len(findings), len(want), findings)
	}
	for i, f := range findings {
		if f.Severity != want[i] {
			t.Errorf("finding %d (%s %v): severity %s, want %s: %s", i, f.Producer, f.Command, f.Severity, want[i], f.Message)
		}
		if f.Consumer != "howlchangeops" {
			t.Errorf("finding %d consumer = %q", i, f.Consumer)
		}
	}
	for i, fragment := range map[int]string{1: "no command", 2: "--deadline", 3: "howlframe.receipt/v1", 4: ">= 0.3.0"} {
		if !strings.Contains(findings[i].Message, fragment) {
			t.Errorf("finding %d message %q does not mention %q", i, findings[i].Message, fragment)
		}
	}
}

type fakeRunner struct {
	out map[string]string
	err map[string]error
}

func (f fakeRunner) Describe(_ context.Context, bin string) ([]byte, error) {
	if err := f.err[bin]; err != nil {
		return nil, err
	}
	return []byte(f.out[bin]), nil
}

func TestDescribeSeparatesMissingFromInvalid(t *testing.T) {
	r := fakeRunner{
		out: map[string]string{"good": frameDescriptor, "help": "usage: old [-h]", "bad": `{"schema":"howl.interop/v1","component":"x"}`},
		err: map[string]error{"old": errors.New("exit status 2")},
	}
	if d, err := Describe(context.Background(), r, "good"); err != nil || d.Component != "howlframe" {
		t.Fatalf("good: %v %+v", err, d)
	}
	for _, bin := range []string{"old", "help"} {
		var none *ErrNoDescriptor
		if _, err := Describe(context.Background(), r, bin); !errors.As(err, &none) {
			t.Errorf("%s: want ErrNoDescriptor, got %v", bin, err)
		}
	}
	var none *ErrNoDescriptor
	if _, err := Describe(context.Background(), r, "bad"); err == nil || errors.As(err, &none) {
		t.Errorf("bad: want a validation error, got %v", err)
	}
}

func TestLocatorFollowsTheDocumentedOrder(t *testing.T) {
	root := t.TempDir()
	paths := platform.Paths{DataHome: filepath.Join(root, "data"), BinDir: filepath.Join(root, "bin")}
	home := filepath.Join(root, "home")
	write := func(p string) string {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	localBin := write(filepath.Join(home, ".local", "bin", platform.ExeName("howlframe")))
	env := map[string]string{}
	pathHit := ""
	l := Locator{
		Paths:  paths,
		Getenv: func(k string) string { return env[k] },
		LookPath: func(name string) (string, error) {
			if pathHit == "" || name != platform.ExeName("howlframe") {
				return "", errors.New("not found")
			}
			return pathHit, nil
		},
		Home: home,
	}

	if got, _ := l.Locate("howlframe"); got != localBin {
		t.Fatalf("~/.local/bin: got %q", got)
	}
	pathHit = "/usr/bin/howlframe"
	if got, _ := l.Locate("howlframe"); got != pathHit {
		t.Fatalf("PATH: got %q", got)
	}
	release := write(filepath.Join(paths.ComponentReleaseDir("howlframe", "0.1.0"), platform.ExeName("howlframe")))
	if err := os.WriteFile(paths.ComponentCurrentLink("howlframe"), []byte("0.1.0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := l.Locate("howlframe"); got != release {
		t.Fatalf("managed release: got %q", got)
	}
	env["HOWLFRAME_BIN"] = "/opt/frame"
	if got, _ := l.Locate("howlframe"); got != "/opt/frame" {
		t.Fatalf("env override: got %q", got)
	}
	if EnvVar("howlplane-engine") != "HOWLPLANE_ENGINE_BIN" {
		t.Fatalf("EnvVar = %q", EnvVar("howlplane-engine"))
	}
	if _, ok := l.Locate("howlnothing"); ok {
		t.Fatal("found a component that does not exist")
	}
}

// The canonical schema files must keep naming the identifiers this package
// checks for.
func TestContractSchemasMatchThisPackage(t *testing.T) {
	for file, id := range map[string]string{
		"howl.interop.v1.schema.json": DescriptorSchema,
		"howl.result.v1.schema.json":  ResultSchema,
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "contracts", "interop", file))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			ID         string `json:"$id"`
			Properties struct {
				Schema struct {
					Const string `json:"const"`
				} `json:"schema"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if doc.ID != id || doc.Properties.Schema.Const != id {
			t.Errorf("%s: $id %q / const %q, want %q", file, doc.ID, doc.Properties.Schema.Const, id)
		}
	}
}
