// Package selfupdate updates the running howl binary itself: checking a
// fixed GitHub repository for the latest release, downloading and
// verifying the platform-appropriate artifact the same way any
// github_release component is installed, and atomically replacing the
// current executable while preserving the previous one for manual
// recovery.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/howlcipher/howl/internal/artifact"
)

// Repository is the fixed GitHub repository Howl updates itself from.
const Repository = "howlcipher/howl"

// ReleaseInfo describes an available howl release for the current
// platform.
type ReleaseInfo struct {
	Version      string // e.g. "v1.1.0"
	ArtifactURL  string
	ChecksumURL  string
	ArtifactName string
}

// GithubAPI abstracts the GitHub releases API call so it's testable
// against an httptest server instead of the real api.github.com.
type GithubAPI interface {
	LatestRelease(ctx context.Context, repo string) (*githubRelease, error)
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// HTTPGithubAPI is the real GithubAPI implementation.
type HTTPGithubAPI struct {
	Downloader artifact.Downloader
	BaseURL    string // defaults to https://api.github.com
}

func (a HTTPGithubAPI) baseURL() string {
	if a.BaseURL != "" {
		return a.BaseURL
	}
	return "https://api.github.com"
}

// LatestRelease implements GithubAPI.
func (a HTTPGithubAPI) LatestRelease(ctx context.Context, repo string) (*githubRelease, error) {
	body, err := a.Downloader.Download(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", a.baseURL(), repo))
	if err != nil {
		return nil, fmt.Errorf("failed to query latest release for %s: %w", repo, err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read release metadata: %w", err)
	}
	var rel githubRelease
	if err := json.Unmarshal(data, &rel); err != nil {
		return nil, fmt.Errorf("failed to parse release metadata: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("no release found for %s", repo)
	}
	return &rel, nil
}

// artifactNameFor returns the expected asset name for the current
// platform, matching the naming convention howlframe's release workflow
// (and howl's own, see .github/workflows/release.yml) uses:
// howl_<tag>_<os>_<arch>.<ext>.
func artifactNameFor(tag string) (name, ext string) {
	ext = "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("howl_%s_%s_%s.%s", tag, runtime.GOOS, runtime.GOARCH, ext), ext
}

// Check queries repo for the latest release and, if a matching artifact
// exists for the current platform, returns its download info.
func Check(ctx context.Context, api GithubAPI, repo string) (*ReleaseInfo, error) {
	rel, err := api.LatestRelease(ctx, repo)
	if err != nil {
		return nil, err
	}

	artifactName, _ := artifactNameFor(rel.TagName)
	var artifactURL, checksumURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case artifactName:
			artifactURL = a.BrowserDownloadURL
		case "SHA256SUMS":
			checksumURL = a.BrowserDownloadURL
		}
	}
	if artifactURL == "" || checksumURL == "" {
		return nil, fmt.Errorf("release %s has no published artifact for %s/%s", rel.TagName, runtime.GOOS, runtime.GOARCH)
	}

	return &ReleaseInfo{
		Version:      strings.TrimPrefix(rel.TagName, "v"),
		ArtifactURL:  artifactURL,
		ChecksumURL:  checksumURL,
		ArtifactName: artifactName,
	}, nil
}

// Apply downloads, verifies, and atomically installs the given release
// over currentExePath, keeping the previous binary alongside it (suffixed
// ".prev") for manual recovery. It never leaves currentExePath missing:
// if activation fails partway, it restores the previous binary.
func Apply(ctx context.Context, dl artifact.Downloader, rel *ReleaseInfo, currentExePath, cacheDir string) error {
	artifactPath := filepath.Join(cacheDir, rel.ArtifactName)
	checksumPath := filepath.Join(cacheDir, "howl-"+rel.Version+"-SHA256SUMS")

	if err := artifact.DownloadToFile(ctx, dl, rel.ArtifactURL, artifactPath); err != nil {
		return fmt.Errorf("failed to download howl %s: %w", rel.Version, err)
	}
	if err := artifact.DownloadToFile(ctx, dl, rel.ChecksumURL, checksumPath); err != nil {
		return fmt.Errorf("failed to download checksums for howl %s: %w", rel.Version, err)
	}

	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	sums, err := artifact.ParseChecksumFile(checksumData)
	if err != nil {
		return err
	}
	expected, ok := sums[rel.ArtifactName]
	if !ok {
		return fmt.Errorf("no checksum entry for %s", rel.ArtifactName)
	}
	if err := artifact.VerifySHA256(artifactPath, expected); err != nil {
		return err
	}

	stageDir, err := os.MkdirTemp(cacheDir, ".howl-selfupdate-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	if strings.HasSuffix(rel.ArtifactName, ".zip") {
		err = artifact.ExtractZip(artifactPath, stageDir)
	} else {
		err = artifact.ExtractTarGz(artifactPath, stageDir)
	}
	if err != nil {
		return fmt.Errorf("failed to extract howl %s: %w", rel.Version, err)
	}

	newBinaryName := "howl"
	if runtime.GOOS == "windows" {
		newBinaryName = "howl.exe"
	}
	newBinaryPath := filepath.Join(stageDir, newBinaryName)
	if _, err := os.Stat(newBinaryPath); err != nil {
		return fmt.Errorf("expected binary %s not found in downloaded release: %w", newBinaryName, err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(newBinaryPath, 0o755)
	}

	return activate(newBinaryPath, currentExePath)
}

// activate moves the previous binary aside (to <currentExePath>.prev) and
// the new one into place. If moving the new binary in fails, it restores
// the previous one so currentExePath is never left missing.
func activate(newBinaryPath, currentExePath string) error {
	prevPath := currentExePath + ".prev"
	_ = os.Remove(prevPath)

	hadPrevious := false
	if err := os.Rename(currentExePath, prevPath); err == nil {
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to preserve current howl binary: %w", err)
	}

	if err := os.Rename(newBinaryPath, currentExePath); err != nil {
		if hadPrevious {
			_ = os.Rename(prevPath, currentExePath)
		}
		return fmt.Errorf("failed to activate new howl binary: %w", err)
	}
	return nil
}
