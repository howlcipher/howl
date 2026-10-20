package interop

import "github.com/howlcipher/howl/internal/version"

// Self describes the commands of other components that howl itself runs:
// the three narrow HowlPlane process adapters (docs/SCOPE.md) and the
// HowlFrame build steps of a developer install. Howl publishes no
// `interop` command of its own; this descriptor exists only so those edges
// are checked like any other.
func Self() *Descriptor {
	return &Descriptor{
		Schema:    DescriptorSchema,
		Component: "howl",
		Version:   version.GetInfo().Version,
		Commands:  []Command{},
		ExitCodes: map[string]string{
			"0": "ok", "1": "error", "2": "cancelled", "3": "validation", "4": "missing dependency",
			"5": "integrity", "6": "install", "7": "verify", "8": "rollback", "9": "locked",
		},
		Consumes: []Consume{
			{Component: "howlplane", Command: []string{"orchestrate"}},
			{Component: "howlplane", Command: []string{"agents", "doctor"}},
			{Component: "howlplane", Command: []string{"factory", "prepare"}},
			{Component: "howlframe", Command: []string{"check"}},
			{Component: "howlframe", Command: []string{"build"}},
		},
	}
}
