package cmd

import (
	"fmt"

	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/state"
)

// appContext bundles the paths, release manifest, and installer state
// almost every command needs, loaded once with consistent error handling.
type appContext struct {
	Paths          platform.Paths
	Manifest       *manifest.Manifest
	ManifestSource string
	State          *state.State
	StateExisted   bool
}

func loadAppContext(manifestPathOverride string) (*appContext, error) {
	paths, err := platform.DefaultPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Howl's data paths: %w", err)
	}
	if err := paths.EnsureOwnedDirs(); err != nil {
		return nil, err
	}

	var m *manifest.Manifest
	var src string
	if manifestPathOverride != "" {
		m, err = manifest.Load(manifestPathOverride)
		src = manifestPathOverride
	} else {
		m, src, err = manifest.LoadDefault(".")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load ecosystem manifest: %w", err)
	}

	st, existed, err := state.Load(paths.StateFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load installer state: %w", err)
	}
	if st.Channel == "" {
		st.Channel = m.Ecosystem.Channel
	}

	return &appContext{
		Paths:          paths,
		Manifest:       m,
		ManifestSource: src,
		State:          st,
		StateExisted:   existed,
	}, nil
}
