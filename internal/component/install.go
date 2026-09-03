package component

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/howlcipher/howl/internal/artifact"
	"github.com/howlcipher/howl/internal/devlocate"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/platform"
	"github.com/howlcipher/howl/internal/pyruntime"
)

// Installer implements engine.Installer for every install method a
// component might declare. It's the one place ecosystem-repository-
// specific installation mechanics live, kept narrow and typed rather than
// scattered through CLI code.
type Installer struct {
	Downloader           artifact.Downloader
	Runner               pyruntime.Runner // also used to run `go build` and other components' binaries
	Getenv               func(string) string
	BaseDir              string // devlocate sibling-checkout fallback base
	ExplicitCheckoutDirs map[string]string
	// GithubBaseURL overrides the GitHub Releases host, defaulting to
	// https://github.com. Only ever changed in tests, against an
	// httptest server standing in for github.com.
	GithubBaseURL string
}

func (i *Installer) githubBaseURL() string {
	if i.GithubBaseURL != "" {
		return i.GithubBaseURL
	}
	return "https://github.com"
}

// Install fetches/builds c at its target version and activates it.
func (i *Installer) Install(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	switch c.Install.Method {
	case manifest.MethodGithubRelease:
		return i.installGithubRelease(ctx, c, paths)
	case manifest.MethodSourceBuild:
		if c.Install.SourceBuild == nil {
			return fmt.Errorf("component %q has no source_build configuration", c.Name)
		}
		switch c.Install.SourceBuild.Language {
		case "go":
			return i.installGoSource(ctx, c, paths)
		case "python":
			return i.installPythonSource(ctx, c, paths)
		default:
			return fmt.Errorf("component %q declares unsupported source_build language %q", c.Name, c.Install.SourceBuild.Language)
		}
	default:
		return fmt.Errorf("component %q declares unsupported install method %q", c.Name, c.Install.Method)
	}
}

func (i *Installer) locateCheckout(checkoutName string) (string, error) {
	dir, ok := devlocate.Locate(checkoutName, devlocate.Options{
		ExplicitDirs: i.ExplicitCheckoutDirs,
		BaseDir:      i.BaseDir,
	}, i.getenv())
	if !ok {
		envKey := "HOWL_" + strings.ToUpper(strings.ReplaceAll(checkoutName, "-", "_")) + "_DIR"
		return "", fmt.Errorf("could not locate source checkout %q: set %s or HOWL_DEV_WORKSPACE", checkoutName, envKey)
	}
	return dir, nil
}

func (i *Installer) getenv() func(string) string {
	if i.Getenv != nil {
		return i.Getenv
	}
	return os.Getenv
}

func (i *Installer) installGithubRelease(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	gr := c.Install.GithubRelease
	if gr == nil {
		return fmt.Errorf("component %q has no github_release configuration", c.Name)
	}

	tag := "v" + c.Version
	osName, archName := runtime.GOOS, runtime.GOARCH
	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}
	artifactName := strings.NewReplacer(
		"{version}", tag,
		"{os}", osName,
		"{arch}", archName,
		"{ext}", ext,
	).Replace(gr.ArtifactPattern)

	base := fmt.Sprintf("%s/%s/releases/download/%s", i.githubBaseURL(), gr.Repository, tag)

	cacheDir := paths.DownloadCacheDir()
	artifactPath := filepath.Join(cacheDir, artifactName)
	checksumPath := filepath.Join(cacheDir, c.Name+"-"+tag+"-"+gr.ChecksumFile)

	if err := artifact.DownloadToFile(ctx, i.Downloader, base+"/"+artifactName, artifactPath); err != nil {
		return fmt.Errorf("failed to download %s: %w", c.Name, err)
	}
	if err := artifact.DownloadToFile(ctx, i.Downloader, base+"/"+gr.ChecksumFile, checksumPath); err != nil {
		return fmt.Errorf("failed to download checksums for %s: %w", c.Name, err)
	}

	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("failed to read downloaded checksum file: %w", err)
	}
	sums, err := artifact.ParseChecksumFile(checksumData)
	if err != nil {
		return err
	}
	expected, ok := sums[artifactName]
	if !ok {
		return fmt.Errorf("no checksum entry for %s in %s", artifactName, gr.ChecksumFile)
	}
	if err := artifact.VerifySHA256(artifactPath, expected); err != nil {
		return err
	}

	releaseDir := paths.ComponentReleaseDir(c.Name, c.Version)
	if err := os.RemoveAll(releaseDir); err != nil {
		return fmt.Errorf("failed to clear staging directory: %w", err)
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	if ext == "zip" {
		err = artifact.ExtractZip(artifactPath, releaseDir)
	} else {
		err = artifact.ExtractTarGz(artifactPath, releaseDir)
	}
	if err != nil {
		return fmt.Errorf("failed to extract %s: %w", c.Name, err)
	}

	binPath := filepath.Join(releaseDir, platform.ExeName(c.Name))
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("expected binary %s not found after extracting %s: %w", binPath, artifactName, err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(binPath, 0o755)
	}

	return ActivateRelease(paths, c.Name, c.Version)
}

func (i *Installer) installGoSource(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	sb := c.Install.SourceBuild
	checkoutDir, err := i.locateCheckout(sb.CheckoutName)
	if err != nil {
		return err
	}

	releaseDir := paths.ComponentReleaseDir(c.Name, c.Version)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	outputPath := filepath.Join(releaseDir, platform.ExeName(sb.Go.BuildOutput))
	if _, err := i.Runner.Run(ctx, checkoutDir, "go", []string{"build", "-o", outputPath, sb.Go.Package}); err != nil {
		return fmt.Errorf("failed to build %s from source: %w", c.Name, err)
	}

	canonical := filepath.Join(releaseDir, platform.ExeName(c.Name))
	if canonical != outputPath {
		if err := copyFile(outputPath, canonical); err != nil {
			return err
		}
	}

	for _, step := range sb.Go.ExtraSteps {
		runnerBin, ok := CurrentBinaryPath(paths, step.RunComponent)
		if !ok {
			return fmt.Errorf("build step for %s requires %s to already be installed", c.Name, step.RunComponent)
		}
		if _, err := i.Runner.Run(ctx, checkoutDir, runnerBin, step.Args); err != nil {
			return fmt.Errorf("build step `%s %v` failed for %s: %w", step.RunComponent, step.Args, c.Name, err)
		}
	}

	return ActivateRelease(paths, c.Name, c.Version)
}

func (i *Installer) installPythonSource(ctx context.Context, c manifest.Component, paths platform.Paths) error {
	sb := c.Install.SourceBuild
	checkoutDir, err := i.locateCheckout(sb.CheckoutName)
	if err != nil {
		return err
	}

	pythonPath, _, err := pyruntime.DetectPython(ctx, i.Runner, sb.Python.MinPython)
	if err != nil {
		return fmt.Errorf("failed to provision python runtime for %s: %w", c.Name, err)
	}

	runtimeRoot := paths.RuntimeDir(c.Name)
	venvDir := pyruntime.VenvDir(runtimeRoot)
	if err := pyruntime.CreateVenv(ctx, i.Runner, pythonPath, venvDir); err != nil {
		return fmt.Errorf("failed to create python runtime for %s: %w", c.Name, err)
	}
	if err := pyruntime.PipInstallEditable(ctx, i.Runner, venvDir, checkoutDir, sb.Python.PackageDir, sb.Python.Extras); err != nil {
		return fmt.Errorf("failed to install %s into its python runtime: %w", c.Name, err)
	}

	releaseDir := paths.ComponentReleaseDir(c.Name, c.Version)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	wrapperPath := filepath.Join(releaseDir, platform.ExeName(c.Name))
	if err := pyruntime.WriteWrapperScript(wrapperPath, venvDir, sb.Python.ConsoleScript); err != nil {
		return err
	}

	return ActivateRelease(paths, c.Name, c.Version)
}
