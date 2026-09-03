package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// githubReleaseManifestTOML declares a single component installable via a
// real (httptest-served) github_release download, so install tests don't
// need a Go/Python toolchain or a real sibling checkout. The download host
// itself is redirected via the HOWL_GITHUB_BASE_URL test-only override.
func githubReleaseManifestTOML() string {
	return `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[components]]
name = "howlframe"
role = "language"
version = "0.1.1"
platforms = ["linux", "darwin", "windows"]
archs = ["amd64", "arm64"]
  [components.install]
  method = "github_release"
    [components.install.github_release]
    repository = "x/howlframe"
    artifact_pattern = "howlframe_{version}_{os}_{arch}.{ext}"
    checksum_file = "SHA256SUMS"
  [components.health_check]
  type = "binary_exists"
`
}

func buildTarGzFixture(t *testing.T, name string, content []byte) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func startFakeGithubRelease(t *testing.T) *httptest.Server {
	t.Helper()
	binaryContent := []byte("fake howlframe binary")
	tarPath := buildTarGzFixture(t, "howlframe", binaryContent)
	archiveBytes, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	checksumLine := fmt.Sprintf("%s  howlframe_v0.1.1_linux_amd64.tar.gz\n", hex.EncodeToString(sum[:]))

	mux := http.NewServeMux()
	mux.HandleFunc("/x/howlframe/releases/download/v0.1.1/howlframe_v0.1.1_linux_amd64.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tarPath)
	})
	mux.HandleFunc("/x/howlframe/releases/download/v0.1.1/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumLine))
	})
	return httptest.NewServer(mux)
}

func TestInstallEndToEndAgainstFakeGithubRelease(t *testing.T) {
	sandboxHowlPaths(t)
	srv := startFakeGithubRelease(t)
	defer srv.Close()
	t.Setenv("HOWL_GITHUB_BASE_URL", srv.URL)

	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(githubReleaseManifestTOML()), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"install", "--manifest", manifestFile, "--yes"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected install error: %v\noutput:\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "Install complete") {
		t.Errorf("expected success message, got:\n%s", buf.String())
	}

	// Running install again must be a safe no-op (idempotency).
	rootCmd2 := NewRootCommand()
	var buf2 bytes.Buffer
	rootCmd2.SetOut(&buf2)
	rootCmd2.SetArgs([]string{"install", "--manifest", manifestFile, "--yes"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("unexpected error on repeat install: %v", err)
	}
	if !strings.Contains(buf2.String(), "Already installed") {
		t.Errorf("expected idempotent re-install to report already installed, got:\n%s", buf2.String())
	}

	// howl status must now show it installed and up to date.
	rootCmd3 := NewRootCommand()
	var buf3 bytes.Buffer
	rootCmd3.SetOut(&buf3)
	rootCmd3.SetArgs([]string{"status", "--manifest", manifestFile})
	if err := rootCmd3.Execute(); err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if !strings.Contains(buf3.String(), "up to date") {
		t.Errorf("expected status to report up to date, got:\n%s", buf3.String())
	}
}

func TestInstallUnknownProfileRejected(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"install", "--manifest", manifestFile, "--profile", "bogus", "--yes"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitValidationFailure {
		t.Errorf("expected ExitValidationFailure, got %v", err)
	}
}

func TestInstallMissingRequiredDependency(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	toml := `
schema_version = 1
[ecosystem]
name = "Howl"
version = "0.1.0"
channel = "stable"

[[components]]
name = "howlchangeops"
role = "gate"
version = "0.1.0"
platforms = ["linux"]
archs = ["amd64"]
  [components.install]
  method = "source_build"
    [components.install.source_build]
    checkout_name = "howlchangeops-nonexistent-checkout"
    language = "go"
      [components.install.source_build.go]
      package = "./adapter"
      build_output = "howlchangeops"
  [components.health_check]
  type = "binary_exists"
  [[components.external_dependencies]]
  name = "definitely-not-a-real-binary-xyz"
  required = true
  capability = "source-build"
`
	if err := os.WriteFile(manifestFile, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"install", "--manifest", manifestFile, "--yes"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required dependency")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitMissingDependency {
		t.Errorf("expected ExitMissingDependency, got %v", err)
	}
	if !strings.Contains(buf.String(), "Missing required dependency") {
		t.Errorf("expected missing-dependency message in output, got:\n%s", buf.String())
	}
}

func TestInstallCancelledWithoutYes(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetIn(strings.NewReader("n\n"))
	rootCmd.SetArgs([]string{"install", "--manifest", manifestFile})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if ec, ok := err.(ExitCoder); !ok || ec.ExitCode() != ExitCancelled {
		t.Errorf("expected ExitCancelled, got %v", err)
	}
}

func TestInstallAlreadyUpToDateIsNoop(t *testing.T) {
	sandboxHowlPaths(t)
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-seed state so the single declared component (howlplane@1.0.0)
	// already matches the manifest target -- install should be a no-op
	// and must not even prompt for confirmation.
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlplane", "1.0.0")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"install", "--manifest", manifestFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Already installed") {
		t.Errorf("expected already-installed message, got:\n%s", buf.String())
	}
}
