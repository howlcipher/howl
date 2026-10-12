package plan

import (
	"reflect"
	"testing"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/state"
)

func componentNames(p *Plan) []string {
	names := make([]string, len(p.Components))
	for i, c := range p.Components {
		names[i] = c.Name
	}
	return names
}

func TestScopedChangeOpsIncludesFrameAcrossProfiles(t *testing.T) {
	m, _, err := manifest.LoadDefault(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"standard", "developer", "local-ai"} {
		t.Run(profile, func(t *testing.T) {
			p, err := Build(m, state.New("stable"), BuildOptions{Profile: profile, Components: []string{"howlchangeops"}})
			if err != nil {
				t.Fatal(err)
			}
			if got := componentNames(p); !reflect.DeepEqual(got, []string{"howlframe", "howlchangeops"}) {
				t.Fatalf("scoped dependency plan = %v", got)
			}
			for _, cp := range p.Components {
				expected := manifest.MethodGithubRelease
				if profile == "developer" {
					expected = manifest.MethodSourceBuild
				}
				if cp.Component.Install.Method != expected || cp.Action != ActionInstall {
					t.Errorf("profile/action changed for %s: %+v", cp.Name, cp)
				}
			}
		})
	}
}

func TestScopedTransitiveSharedDependencies(t *testing.T) {
	m := testManifest(t)
	// The selected leaves share ChangeOps, which itself requires Frame.
	template := m.Components[1]
	for _, name := range []string{"leaf-a", "leaf-b"} {
		c := template
		c.Name = name
		c.DependsOn = []manifest.Dependency{{Component: "howlchangeops"}}
		m.Components = append(m.Components, c)
	}
	unrelated := template
	unrelated.Name = "unrelated"
	unrelated.DependsOn = nil
	m.Components = append(m.Components, unrelated)
	st := state.New("stable")
	st.Components["howlframe"] = state.ComponentState{Version: "0.1.1"}
	st.Components["howlchangeops"] = state.ComponentState{Version: "older"}
	p, err := Build(m, st, BuildOptions{Profile: "standard", Components: []string{"leaf-b", "leaf-a", "leaf-b"}, Detector: fakeDetector{found: map[string]string{"go": "go1.22"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := componentNames(p); !reflect.DeepEqual(got, []string{"howlframe", "howlchangeops", "leaf-a", "leaf-b"}) {
		t.Fatalf("transitive shared dependency plan = %v", got)
	}
	if p.Components[0].Action != ActionSkip || p.Components[1].Action != ActionUpdate || p.Components[2].Action != ActionInstall || p.Components[3].Action != ActionInstall {
		t.Fatalf("wrong installed-version actions: %+v", p.Components)
	}
	if len(p.Dependencies) != 1 || p.Dependencies[0].Name != "go" {
		t.Fatalf("shared external dependency duplicated: %+v", p.Dependencies)
	}
}

func TestScopedDependencyAlreadyInstalledIsNoOp(t *testing.T) {
	m := testManifest(t)
	st := state.New("stable")
	for _, c := range m.Components {
		st.Components[c.Name] = state.ComponentState{Version: c.Version}
	}
	p, err := Build(m, st, BuildOptions{Components: []string{"howlchangeops"}, Profile: "standard"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Components) != 2 || p.HasWork() || len(p.Dependencies) != 0 {
		t.Fatalf("expected complete no-op scope: %+v", p)
	}
}

func TestScopedUnknownMixedWithKnownComponentErrors(t *testing.T) {
	_, err := Build(testManifest(t), state.New("stable"), BuildOptions{Components: []string{"howlchangeops", "unknown"}})
	if err == nil {
		t.Fatal("unknown scoped component accepted")
	}
}
