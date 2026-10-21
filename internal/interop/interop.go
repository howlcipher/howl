// Package interop reads the howl.interop/v1 descriptors that ecosystem
// components print from `<component> interop --json` and checks that every
// command one component runs of another still exists there, with the flags
// and schemas the caller relies on. See docs/INTEROP.md.
//
// Howl only reads descriptors to judge whether installed components are
// compatible with each other; it never calls the commands they describe.
package interop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/howlcipher/howl/internal/manifest"
)

// DescriptorSchema identifies an interop descriptor.
const DescriptorSchema = "howl.interop/v1"

// ResultSchema identifies the command result envelope.
const ResultSchema = "howl.result/v1"

// Command is one command a component offers. An empty Path describes the
// global flags that come before any subcommand.
type Command struct {
	Path    []string `json:"path"`
	Flags   []string `json:"flags"`
	JSON    bool     `json:"json"`
	Emits   []string `json:"emits,omitempty"`
	Accepts []string `json:"accepts,omitempty"`
}

// Consume is one command of another component that this one runs.
type Consume struct {
	Component  string   `json:"component"`
	Command    []string `json:"command"`
	Flags      []string `json:"flags,omitempty"`
	Schemas    []string `json:"schemas,omitempty"`
	MinVersion string   `json:"min_version,omitempty"`
}

// Descriptor is a component's howl.interop/v1 document.
type Descriptor struct {
	Schema    string            `json:"schema"`
	Component string            `json:"component"`
	Version   string            `json:"version"`
	Commands  []Command         `json:"commands"`
	ExitCodes map[string]string `json:"exit_codes"`
	Env       []string          `json:"env,omitempty"`
	Consumes  []Consume         `json:"consumes,omitempty"`
}

var (
	componentRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	schemaIDRe  = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+/v[0-9]+$`)
	flagRe      = regexp.MustCompile(`^--?[A-Za-z0-9][A-Za-z0-9_-]*$`)
	exitCodeRe  = regexp.MustCompile(`^[0-9]+$`)
	envRe       = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	// versionRe accepts semver ("0.2.0", "0.2.0-rc1") and PEP 440
	// ("0.2.0b1", "0.2.0.dev3") release strings; only the leading
	// major.minor.patch is compared.
	versionRe = regexp.MustCompile(`^v?([0-9]+\.[0-9]+\.[0-9]+)([-+.a-z][0-9A-Za-z.+-]*)?$`)
)

// releaseCore returns the major.minor.patch a version string starts with.
func releaseCore(v string) (string, bool) {
	m := versionRe.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// Parse decodes and validates a descriptor.
func Parse(data []byte) (*Descriptor, error) {
	var d Descriptor
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

// Validate checks the descriptor against the howl.interop/v1 rules.
func (d *Descriptor) Validate() error {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if d.Schema != DescriptorSchema {
		add("schema is %q, want %q", d.Schema, DescriptorSchema)
	}
	if !componentRe.MatchString(d.Component) {
		add("component %q is not a lowercase component name", d.Component)
	}
	if _, ok := releaseCore(d.Version); !ok {
		add("version %q is not a major.minor.patch release string", d.Version)
	}
	if d.Commands == nil {
		add("commands is missing")
	}
	if len(d.ExitCodes) == 0 {
		add("exit_codes is missing or empty")
	}
	for code, meaning := range d.ExitCodes {
		if !exitCodeRe.MatchString(code) || strings.TrimSpace(meaning) == "" {
			add("exit_codes entry %q: %q is not a numeric code with a meaning", code, meaning)
		}
	}
	seen := map[string]bool{}
	for _, c := range d.Commands {
		key := strings.Join(c.Path, " ")
		if seen[key] {
			add("command %q is listed twice", key)
		}
		seen[key] = true
		for _, f := range c.Flags {
			if !flagRe.MatchString(f) {
				add("command %q: flag %q is not a flag name", key, f)
			}
		}
		for _, s := range append(append([]string{}, c.Emits...), c.Accepts...) {
			if !schemaIDRe.MatchString(s) {
				add("command %q: %q is not a <namespace>.<kind>/vN schema id", key, s)
			}
		}
	}
	for _, e := range d.Env {
		if !envRe.MatchString(e) {
			add("env %q is not an environment variable name", e)
		}
	}
	for _, c := range d.Consumes {
		if !componentRe.MatchString(c.Component) {
			add("consumes: component %q is not a lowercase component name", c.Component)
		}
		for _, f := range c.Flags {
			if !flagRe.MatchString(f) {
				add("consumes %s %q: flag %q is not a flag name", c.Component, strings.Join(c.Command, " "), f)
			}
		}
		for _, s := range c.Schemas {
			if !schemaIDRe.MatchString(s) {
				add("consumes %s %q: %q is not a schema id", c.Component, strings.Join(c.Command, " "), s)
			}
		}
		if c.MinVersion != "" {
			if _, ok := releaseCore(c.MinVersion); !ok {
				add("consumes %s: min_version %q is not a major.minor.patch release string", c.Component, c.MinVersion)
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("invalid %s descriptor: %s", DescriptorSchema, strings.Join(problems, "; "))
	}
	return nil
}

// Lookup returns the command with the given path.
func (d *Descriptor) Lookup(path []string) (*Command, bool) {
	want := strings.Join(path, " ")
	for i := range d.Commands {
		if strings.Join(d.Commands[i].Path, " ") == want {
			return &d.Commands[i], true
		}
	}
	return nil, false
}

// flagName strips leading dashes, so `-allow-caps` and `--allow-caps` (both
// accepted by Go's flag package) compare equal.
func flagName(f string) string { return strings.TrimLeft(f, "-") }

// Severity grades one edge.
type Severity string

const (
	SeverityOK   Severity = "ok"
	SeverityWarn Severity = "warn"
	SeverityFail Severity = "fail"
	// SeveritySkip marks an edge whose producer is not present at all, so
	// the edge is not in use on this machine.
	SeveritySkip Severity = "skip"
)

// Finding is the outcome of checking one consumes entry.
type Finding struct {
	Consumer string   `json:"consumer"`
	Producer string   `json:"producer"`
	Command  []string `json:"command"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

// Resolve checks every consumes entry in descs against its producer.
// unavailable maps a component that is present but published no usable
// descriptor to the reason; a producer in neither map is absent.
func Resolve(descs map[string]*Descriptor, unavailable map[string]string) []Finding {
	consumers := make([]string, 0, len(descs))
	for name := range descs {
		consumers = append(consumers, name)
	}
	sort.Strings(consumers)

	var findings []Finding
	for _, consumer := range consumers {
		for _, use := range descs[consumer].Consumes {
			f := Finding{Consumer: consumer, Producer: use.Component, Command: use.Command}
			f.Severity, f.Message = check(use, descs[use.Component], unavailable, use.Component)
			findings = append(findings, f)
		}
	}
	return findings
}

func check(use Consume, producer *Descriptor, unavailable map[string]string, name string) (Severity, string) {
	cmd := strings.TrimSpace(name + " " + strings.Join(use.Command, " "))
	if producer == nil {
		if reason, ok := unavailable[name]; ok {
			return SeverityWarn, fmt.Sprintf("cannot check `%s`: %s", cmd, reason)
		}
		return SeveritySkip, fmt.Sprintf("%s is not present; edge unused", name)
	}

	var problems []string
	if use.MinVersion != "" {
		have, _ := releaseCore(producer.Version)
		want, _ := releaseCore(use.MinVersion)
		if c, err := manifest.CompareVersions(have, want); err == nil && c < 0 {
			problems = append(problems, fmt.Sprintf("needs %s >= %s, found %s", name, use.MinVersion, producer.Version))
		}
	}
	command, ok := producer.Lookup(use.Command)
	if !ok {
		problems = append(problems, fmt.Sprintf("%s %s has no command `%s`", name, producer.Version, cmd))
		return SeverityFail, strings.Join(problems, "; ")
	}
	offered := map[string]bool{}
	for _, f := range command.Flags {
		offered[flagName(f)] = true
	}
	var missingFlags []string
	for _, f := range use.Flags {
		if !offered[flagName(f)] {
			missingFlags = append(missingFlags, f)
		}
	}
	if len(missingFlags) > 0 {
		problems = append(problems, fmt.Sprintf("`%s` does not accept %s", cmd, strings.Join(missingFlags, ", ")))
	}
	schemas := map[string]bool{}
	for _, s := range append(append([]string{}, command.Emits...), command.Accepts...) {
		schemas[s] = true
	}
	var missingSchemas []string
	for _, s := range use.Schemas {
		if !schemas[s] {
			missingSchemas = append(missingSchemas, s)
		}
	}
	if len(missingSchemas) > 0 {
		problems = append(problems, fmt.Sprintf("`%s` neither emits nor accepts %s", cmd, strings.Join(missingSchemas, ", ")))
	}
	if len(problems) > 0 {
		return SeverityFail, strings.Join(problems, "; ")
	}
	return SeverityOK, fmt.Sprintf("`%s` matches %s %s", cmd, name, producer.Version)
}

// Runner runs a component's `interop --json` and returns its stdout.
type Runner interface {
	Describe(ctx context.Context, binPath string) ([]byte, error)
}

// ExecRunner runs the descriptor command as a child process with a short
// timeout, keeping stderr out of the JSON it returns.
type ExecRunner struct {
	Timeout time.Duration
}

// Describe implements Runner.
func (r ExecRunner) Describe(ctx context.Context, binPath string) ([]byte, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, "interop", "--json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		// The last stderr line carries the reason (argparse and cobra both
		// print usage first and the error last).
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		msg := strings.TrimSpace(lines[len(lines)-1])
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return out, nil
}

// ErrNoDescriptor reports output that is not a descriptor at all, as from
// a release that predates the interop command.
type ErrNoDescriptor struct{ Reason string }

func (e *ErrNoDescriptor) Error() string { return e.Reason }

// Describe runs binPath's descriptor command. A release without the
// command yields *ErrNoDescriptor; a descriptor that is present but
// malformed yields a plain error.
func Describe(ctx context.Context, r Runner, binPath string) (*Descriptor, error) {
	out, err := r.Describe(ctx, binPath)
	if err != nil {
		return nil, &ErrNoDescriptor{Reason: fmt.Sprintf("publishes no interop descriptor (`interop --json` failed: %v)", err)}
	}
	var probe struct {
		Schema string `json:"schema"`
	}
	if json.Unmarshal(bytes.TrimSpace(out), &probe) != nil || probe.Schema != DescriptorSchema {
		return nil, &ErrNoDescriptor{Reason: "publishes no interop descriptor (`interop --json` printed no " + DescriptorSchema + " document)"}
	}
	return Parse(out)
}
