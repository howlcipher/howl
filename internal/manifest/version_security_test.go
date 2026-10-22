package manifest

import "testing"

func TestPrereleasePrecedenceRegression(t *testing.T) {
	for _, pair := range [][2]string{{"1.0.0-rc.1", "1.0.0"}, {"1.0.0rc1", "1.0.0"}, {"1.0.0-alpha.2", "1.0.0-alpha.10"}, {"1.0.0.dev2", "1.0.0a1"}, {"1.0.0a1", "1.0.0b1"}, {"1.0.0b1", "1.0.0rc1"}, {"1.0.0", "1.0.0.post1"}, {"1.0.0.post1.dev1", "1.0.0.post1"}} {
		got, err := CompareVersions(pair[0], pair[1])
		if err != nil || got >= 0 {
			t.Errorf("%s < %s: got %d, %v", pair[0], pair[1], got, err)
		}
	}
}
func TestVersionInvalidFailsClosed(t *testing.T) {
	for _, v := range []string{"1.0.0garbage", "1.0.0-01", "1.0.0-", "1.0.0+", "01.0.0", "999999999999999999999999.0.0", "1.0.0rc999999999999999999999999", "1.0.0-999999999999999999999999"} {
		if _, err := CompareVersions(v, "1.0.0"); err == nil {
			t.Errorf("accepted %q", v)
		}
	}
}

func TestSemverBuildMetadataDoesNotAffectPrecedence(t *testing.T) {
	for _, pair := range [][2]string{{"1.0.0+build.1", "1.0.0+build.2"}, {"1.0.0-rc.1+build.1", "1.0.0-rc.1+build.2"}} {
		if got, err := CompareVersions(pair[0], pair[1]); err != nil || got != 0 {
			t.Fatalf("metadata changed precedence: %d %v", got, err)
		}
	}
}

func TestPEPLocalNumericNormalizationKeepsTailOrdering(t *testing.T) {
	for _, pair := range [][2]string{{"1.0.0rc1+01.a", "1.0.0rc1+1.b"}, {"1.0.0.post1+01.a", "1.0.0.post1+1.b"}, {"1.0.0.dev1+01.a", "1.0.0.dev1+1.b"}} {
		if got, err := CompareVersions(pair[0], pair[1]); err != nil || got >= 0 {
			t.Fatalf("local tail lost: %d %v", got, err)
		}
	}
}
func TestMetadataDialectValidation(t *testing.T) {
	for _, v := range []string{"1.0.0+build_name", "1.0.0-rc1+build_name"} {
		if _, err := ParseVersion(v); err == nil {
			t.Fatalf("invalid SemVer metadata accepted: %s", v)
		}
	}
	for _, v := range []string{"1.0.0+build--name", "1.0.0rc1+build_name"} {
		if _, err := ParseVersion(v); err != nil {
			t.Fatalf("valid metadata rejected: %s %v", v, err)
		}
	}
}
