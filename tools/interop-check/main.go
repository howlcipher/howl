// Command interop-check resolves every cross-component edge declared in the
// howl.interop/v1 descriptors of the Howl components it can find, using the
// discovery order in docs/INTEROP.md. It is a developer and CI tool, not a
// `howl` subcommand: `howl doctor` runs the same check over installed
// components.
//
//	go run ./tools/interop-check [--json] [--strict] [--bin name=path ...] [--components a,b]
//
// It exits 1 when any edge fails (or, with --strict, when any edge or
// descriptor only warns).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/howlcipher/howl/internal/interop"
)

// defaultComponents are the ecosystem command lines other components run.
var defaultComponents = []string{
	"howlframe", "howlchangeops", "howlplane", "howlwriter", "howldream", "howlcreate",
	"howlproof", "howlinstinct", "howlforge", "howlrelay",
}

type binFlags map[string]string

func (b binFlags) String() string { return "" }
func (b binFlags) Set(v string) error {
	name, path, ok := strings.Cut(v, "=")
	if !ok || name == "" || path == "" {
		return fmt.Errorf("want name=path, got %q", v)
	}
	b[name] = path
	return nil
}

type descriptorReport struct {
	Component string `json:"component"`
	Path      string `json:"path,omitempty"`
	Status    string `json:"status"`
	Version   string `json:"version,omitempty"`
	Message   string `json:"message,omitempty"`
}

type report struct {
	Schema      string             `json:"schema"`
	Descriptors []descriptorReport `json:"descriptors"`
	Edges       []interop.Finding  `json:"edges"`
	Failed      bool               `json:"failed"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("interop-check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print a machine-readable report")
	strict := fs.Bool("strict", false, "treat warnings (missing descriptors, unverifiable edges) as failures")
	components := fs.String("components", strings.Join(defaultComponents, ","), "comma-separated components to look for")
	bins := binFlags{}
	fs.Var(bins, "bin", "name=path: use this executable for a component (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	locator, err := interop.DefaultLocator()
	if err != nil {
		fmt.Fprintln(stderr, "interop-check:", err)
		return 2
	}
	rep := check(context.Background(), strings.Split(*components, ","), bins, locator.Locate, interop.ExecRunner{}, *strict)
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
	} else {
		render(stdout, rep)
	}
	if rep.Failed {
		return 1
	}
	return 0
}

func check(ctx context.Context, names []string, bins map[string]string, locate func(string) (string, bool), runner interop.Runner, strict bool) report {
	rep := report{Schema: "howl.interop_check/v1"}
	descs := map[string]*interop.Descriptor{"howl": interop.Self()}
	unavailable := map[string]string{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path, ok := bins[name]
		if !ok {
			path, ok = locate(name)
		}
		dr := descriptorReport{Component: name, Path: path}
		if !ok {
			dr.Status, dr.Message = string(interop.SeveritySkip), "not found"
			rep.Descriptors = append(rep.Descriptors, dr)
			continue
		}
		d, err := interop.Describe(ctx, runner, path)
		var none *interop.ErrNoDescriptor
		switch {
		case errors.As(err, &none):
			unavailable[name] = name + " " + none.Reason
			dr.Status, dr.Message = string(interop.SeverityWarn), none.Reason
		case err != nil:
			unavailable[name] = name + " published an invalid descriptor"
			dr.Status, dr.Message = string(interop.SeverityFail), err.Error()
		case d.Component != name:
			unavailable[name] = name + " published a descriptor for " + d.Component
			dr.Status, dr.Message = string(interop.SeverityFail), fmt.Sprintf("descriptor names component %q", d.Component)
		default:
			descs[name] = d
			dr.Status, dr.Version = string(interop.SeverityOK), d.Version
			dr.Message = fmt.Sprintf("%d commands, runs %d of other components'", len(d.Commands), len(d.Consumes))
		}
		rep.Descriptors = append(rep.Descriptors, dr)
	}
	rep.Edges = interop.Resolve(descs, unavailable)

	bad := func(s string) bool {
		return s == string(interop.SeverityFail) || (strict && s == string(interop.SeverityWarn))
	}
	for _, d := range rep.Descriptors {
		rep.Failed = rep.Failed || bad(d.Status)
	}
	for _, e := range rep.Edges {
		rep.Failed = rep.Failed || bad(string(e.Severity))
	}
	return rep
}

func render(w io.Writer, rep report) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMPONENT\tSTATUS\tVERSION\tDETAIL")
	for _, d := range rep.Descriptors {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", d.Component, strings.ToUpper(d.Status), d.Version, d.Message)
	}
	tw.Flush()
	fmt.Fprintln(w)

	edges := append([]interop.Finding(nil), rep.Edges...)
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].Severity < edges[j].Severity })
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "EDGE\tSTATUS\tDETAIL")
	for _, e := range edges {
		edge := e.Consumer + " -> " + strings.TrimSpace(e.Producer+" "+strings.Join(e.Command, " "))
		fmt.Fprintf(tw, "%s\t%s\t%s\n", edge, strings.ToUpper(string(e.Severity)), e.Message)
	}
	tw.Flush()
	if rep.Failed {
		fmt.Fprintln(w, "\nResult: FAILED")
	} else {
		fmt.Fprintln(w, "\nResult: OK")
	}
}
