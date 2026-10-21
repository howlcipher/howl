package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestStrictCIScopeMatchesDefaultComponents(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/interop.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	match := regexp.MustCompile(`--components ([a-z,]+)`).FindStringSubmatch(text)
	if len(match) != 2 {
		t.Fatal("workflow missing explicit component scope")
	}
	selected := map[string]bool{}
	for _, name := range strings.Split(match[1], ",") {
		selected[name] = true
	}
	bins := regexp.MustCompile(`for c in ([a-z ]+); do`).FindStringSubmatch(text)
	if len(bins) != 2 {
		t.Fatal("workflow missing explicit executable overrides")
	}
	for _, name := range defaultComponents {
		if !selected[name] || !strings.Contains(" "+bins[1]+" ", " "+name+" ") {
			t.Errorf("CI does not cover %s", name)
		}
	}
	if len(selected) != len(defaultComponents) {
		t.Fatal("workflow scope differs from CLI scope")
	}
	if !strings.Contains(text, "scripts/test_interop_runtime.py") {
		t.Fatal("workflow lacks behavior smoke check")
	}
}
