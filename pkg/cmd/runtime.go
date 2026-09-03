package cmd

import (
	"os"

	"github.com/howlcipher/howl/internal/artifact"
	"github.com/howlcipher/howl/internal/component"
	"github.com/howlcipher/howl/internal/engine"
	"github.com/howlcipher/howl/internal/health"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/pyruntime"
)

// newEngine wires the real (non-test) Installer and HealthChecker
// implementations used by install/update/rollback.
func newEngine(paths platform.Paths) *engine.Engine {
	cwd, _ := os.Getwd()
	installer := &component.Installer{
		Downloader:    artifact.NewHTTPDownloader(),
		Runner:        pyruntime.ExecRunner{},
		BaseDir:       cwd,
		GithubBaseURL: os.Getenv("HOWL_GITHUB_BASE_URL"),
	}
	healthChecker := health.Checker{Exec: health.OSExecer{}}
	return engine.New(installer, healthChecker, paths)
}
