package manifest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// The supported dialects have exactly three release components: SemVer 2.0
// (optionally prefixed by v), or PEP 440 a/b/rc, post, dev and local releases.
// Epochs and shortened release tuples are deliberately unsupported. Ambiguous
// final +metadata versions follow SemVer (ignored build metadata); PEP local
// ordering applies when an explicit a/b/rc, post or dev marker identifies PEP.
var versionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:(a|b|rc)([0-9]+))?(?:\.post([0-9]+))?(?:\.dev([0-9]+))?(?:\+([0-9A-Za-z_-]+(?:\.[0-9A-Za-z_-]+)*))?$`)

type parsedVersion struct {
	core            [3]int
	sem             []string
	pre             string
	preN, post, dev int
	local           []string
}

func number(s string) (int, error) {
	n, e := strconv.Atoi(s)
	if e != nil || n < 0 {
		return 0, fmt.Errorf("invalid or overflowing version number")
	}
	return n, nil
}
func numeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func parseVersion(v string) (parsedVersion, error) {
	p := parsedVersion{post: -1, dev: -1}
	m := versionPattern.FindStringSubmatch(v)
	if m == nil {
		return p, fmt.Errorf("unsupported version %q", v)
	}
	for i := range p.core {
		n, e := number(m[i+1])
		if e != nil {
			return p, e
		}
		p.core[i] = n
	}
	if m[4] != "" {
		if m[5] != "" || m[7] != "" || m[8] != "" {
			return p, fmt.Errorf("mixed version dialect")
		}
		p.sem = strings.Split(m[4], ".")
		for _, part := range p.sem {
			if numeric(part) {
				if len(part) > 1 && part[0] == '0' {
					return p, fmt.Errorf("leading zero in prerelease")
				}
				if _, e := number(part); e != nil {
					return p, e
				}
			}
		}
	}
	p.pre = m[5]
	for _, entry := range []struct {
		s    string
		dest *int
	}{{m[6], &p.preN}, {m[7], &p.post}, {m[8], &p.dev}} {
		if entry.s != "" {
			n, e := number(entry.s)
			if e != nil {
				return p, e
			}
			*entry.dest = n
		}
	}
	if m[9] != "" {
		pep := p.pre != "" || p.post >= 0 || p.dev >= 0
		metadataPattern := `^[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*$`
		if pep {
			metadataPattern = `^[0-9A-Za-z]+(?:[._-][0-9A-Za-z]+)*$`
		}
		if !regexp.MustCompile(metadataPattern).MatchString(m[9]) {
			return p, fmt.Errorf("invalid version metadata")
		}

		p.local = strings.FieldsFunc(strings.ToLower(m[9]), func(r rune) bool { return r == '.' || r == '_' || r == '-' })
		if p.sem == nil {
			for _, part := range p.local {
				if numeric(part) {
					if _, e := number(part); e != nil {
						return p, e
					}
				}
			}
		}
	}
	return p, nil
}

// ParseVersion validates a supported version and returns its release tuple.
func ParseVersion(v string) ([3]int, error) { p, e := parseVersion(v); return p.core, e }
func cmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
func compareIdentifiers(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		an, bn := numeric(a[i]), numeric(b[i])
		if an && bn {
			av, _ := number(a[i])
			bv, _ := number(b[i])
			return cmp(av, bv)
		}
		if an {
			return -1
		}
		if bn {
			return 1
		}
		return strings.Compare(a[i], b[i])
	}
	return cmp(len(a), len(b))
}
func pepStage(p parsedVersion) int {
	if p.pre != "" {
		return map[string]int{"a": 0, "b": 1, "rc": 2}[p.pre]
	}
	if p.dev >= 0 && p.post < 0 {
		return -1
	}
	return 3
}

// CompareVersions compares supported versions, preserving prerelease precedence.
// Incomparable mixed prerelease dialects fail closed instead of guessing.
func CompareVersions(a, b string) (int, error) {
	av, e := parseVersion(a)
	if e != nil {
		return 0, e
	}
	bv, e := parseVersion(b)
	if e != nil {
		return 0, e
	}
	for i := range av.core {
		if c := cmp(av.core[i], bv.core[i]); c != 0 {
			return c, nil
		}
	}
	if av.sem != nil || bv.sem != nil {
		if (av.sem != nil && (bv.pre != "" || bv.dev >= 0 || bv.post >= 0)) || (bv.sem != nil && (av.pre != "" || av.dev >= 0 || av.post >= 0)) {
			return 0, fmt.Errorf("incomparable prerelease dialects")
		}
		if av.sem == nil {
			return 1, nil
		}
		if bv.sem == nil {
			return -1, nil
		}
		return compareIdentifiers(av.sem, bv.sem), nil
	}
	if av.pre == "" && bv.pre == "" && av.post < 0 && bv.post < 0 && av.dev < 0 && bv.dev < 0 {
		return 0, nil
	}
	if c := cmp(pepStage(av), pepStage(bv)); c != 0 {
		return c, nil
	}
	if c := cmp(av.preN, bv.preN); c != 0 {
		return c, nil
	}
	if c := cmp(av.post, bv.post); c != 0 {
		return c, nil
	}
	// An absent dev suffix sorts after all dev releases.
	if av.dev != bv.dev {
		if av.dev < 0 {
			return 1, nil
		}
		if bv.dev < 0 {
			return -1, nil
		}
		return cmp(av.dev, bv.dev), nil
	}
	if len(av.local) == 0 || len(bv.local) == 0 {
		return cmp(len(av.local), len(bv.local)), nil
	}
	// PEP 440 local numeric segments sort after alphabetic ones.
	for i := 0; i < len(av.local) && i < len(bv.local); i++ {
		x, y := av.local[i], bv.local[i]
		if x == y {
			continue
		}
		xn, yn := numeric(x), numeric(y)
		if xn && yn {
			xx, _ := number(x)
			yy, _ := number(y)
			if c := cmp(xx, yy); c != 0 {
				return c, nil
			}
			continue
		}
		if xn {
			return 1, nil
		}
		if yn {
			return -1, nil
		}
		return strings.Compare(x, y), nil
	}
	return cmp(len(av.local), len(bv.local)), nil
}
