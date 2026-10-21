package interop

import "testing"

func TestMinimumPrereleaseFailsClosed(t *testing.T) {
	for _, v := range []string{"1.0.0-rc.1", "1.0.0rc1", "999999999999999999999999.0.0"} {
		producer := &Descriptor{Version: v, Commands: []Command{{Path: []string{"run"}}}}
		severity, _ := check(Consume{MinVersion: "1.0.0", Command: []string{"run"}}, producer, nil, "producer")
		if severity != SeverityFail {
			t.Errorf("%q accepted: %s", v, severity)
		}
	}
}

func TestInvalidOverrideDoesNotFallBack(t *testing.T) {
	locator := Locator{Getenv: func(string) string { return "/nonexistent/explicit-override" }, LookPath: func(string) (string, error) { return "/otherwise/valid", nil }}
	if _, found := locator.Locate("howlplane"); found {
		t.Fatal("invalid explicit override accepted")
	}
}

func TestOverflowingMinimumFailsClosed(t *testing.T) {
	producer := &Descriptor{Version: "0.0.1", Commands: []Command{{Path: []string{"run"}}}}
	severity, _ := check(Consume{MinVersion: "999999999999999999999999999999999999999999999999999.0.0", Command: []string{"run"}}, producer, nil, "producer")
	if severity != SeverityFail {
		t.Fatal("overflowing minimum was accepted")
	}
}
