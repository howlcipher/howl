package component

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/howlcipher/howl/internal/artifact"
	"github.com/howlcipher/howl/internal/manifest"
	"github.com/howlcipher/howl/internal/platform"
)

type call struct {
	dir  string
	name string
	args []string
}

type fakeRunner struct {
	calls []call
	fail  bool
}

func (f *fakeRunner) Run(ctx context.Context, dir, name string, args []string) ([]byte, error) {
	f.calls = append(f.calls, call{dir: dir, name: name, args: args})
	if f.fail {
		return nil, fmt.Errorf("simulated failure")
	}
	return []byte("3.11.6\n"), nil
}

// touchingRunner simulates `go build -o <path>` by creating an empty file
// at the -o argument, so activation has something real to point at.
type touchingRunner struct {
	fakeRunner
}

func (r *touchingRunner) Run(ctx context.Context, dir, name string, args []string) ([]byte, error) {
	r.calls = append(r.calls, call{dir: dir, name: name, args: args})
	for i, a := range args {
		if a == "-o" && i+1 < len(args) {
			os.WriteFile(args[i+1], []byte("built"), 0o755)
		}
	}
	return []byte(""), nil
}

func testPaths(t *testing.T) platform.Paths {
	t.Helper()
	home := t.TempDir()
	p := platform.ResolvePaths(func(string) string { return "" }, home)
	if err := p.EnsureOwnedDirs(); err != nil {
		t.Fatal(err)
	}
	return p
}

func buildTarGzWithFile(t *testing.T, name string, content []byte) string {
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

func TestInstallGithubReleaseDownloadsVerifiesExtractsActivates(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho fake howlframe\n")
	tarPath := buildTarGzWithFile(t, "howlframe", binaryContent)
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
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := manifest.Component{
		Name:    "howlframe",
		Version: "0.1.1",
		Install: manifest.Install{
			Method: manifest.MethodGithubRelease,
			GithubRelease: &manifest.GithubReleaseSource{
				Repository:      "x/howlframe",
				ArtifactPattern: "howlframe_{version}_{os}_{arch}.{ext}",
				ChecksumFile:    "SHA256SUMS",
			},
		},
	}

	paths := testPaths(t)
	inst := &Installer{
		Downloader:    artifact.HTTPDownloader{Client: srv.Client()},
		GithubBaseURL: srv.URL,
	}

	if err := inst.Install(context.Background(), c, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binPath, ok := CurrentBinaryPath(paths, "howlframe")
	if !ok {
		t.Fatal("expected activated binary path")
	}
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("expected extracted binary to exist: %v", err)
	}
	if string(data) != string(binaryContent) {
		t.Errorf("unexpected binary content")
	}
}

func TestInstallGithubReleaseRejectsChecksumMismatch(t *testing.T) {
	binaryContent := []byte("content")
	tarPath := buildTarGzWithFile(t, "howlframe", binaryContent)

	mux := http.NewServeMux()
	mux.HandleFunc("/x/howlframe/releases/download/v0.1.1/howlframe_v0.1.1_linux_amd64.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tarPath)
	})
	mux.HandleFunc("/x/howlframe/releases/download/v0.1.1/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000  howlframe_v0.1.1_linux_amd64.tar.gz\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := manifest.Component{
		Name:    "howlframe",
		Version: "0.1.1",
		Install: manifest.Install{
			Method: manifest.MethodGithubRelease,
			GithubRelease: &manifest.GithubReleaseSource{
				Repository:      "x/howlframe",
				ArtifactPattern: "howlframe_{version}_{os}_{arch}.{ext}",
				ChecksumFile:    "SHA256SUMS",
			},
		},
	}

	paths := testPaths(t)
	inst := &Installer{
		Downloader:    artifact.HTTPDownloader{Client: srv.Client()},
		GithubBaseURL: srv.URL,
	}

	if err := inst.Install(context.Background(), c, paths); err == nil {
		t.Fatal("expected checksum mismatch to be rejected")
	}
	if _, ok := CurrentBinaryPath(paths, "howlframe"); ok {
		if _, statErr := os.Stat(paths.ComponentCurrentLink("howlframe")); statErr == nil {
			t.Error("must not activate a release that failed checksum verification")
		}
	}
}

func TestInstallGoSourceBuildsAndActivates(t *testing.T) {
	checkoutDir := t.TempDir()
	paths := testPaths(t)

	c := manifest.Component{
		Name:    "howlchangeops",
		Version: "0.1.0",
		Install: manifest.Install{
			Method: manifest.MethodSourceBuild,
			SourceBuild: &manifest.SourceBuild{
				CheckoutName: "howlchangeops",
				Language:     "go",
				Go: &manifest.GoSourceBuild{
					Package:     "./adapter",
					BuildOutput: "howlchangeops",
				},
			},
		},
	}

	runner := &touchingRunner{}
	inst := &Installer{
		Runner:               runner,
		ExplicitCheckoutDirs: map[string]string{"howlchangeops": checkoutDir},
	}

	if err := inst.Install(context.Background(), c, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binPath, ok := CurrentBinaryPath(paths, "howlchangeops")
	if !ok {
		t.Fatal("expected activated binary")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected built binary to exist: %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "go" {
		t.Errorf("expected a single `go build` invocation, got %+v", runner.calls)
	}
	if runner.calls[0].dir != checkoutDir {
		t.Errorf("expected go build to run in checkout dir %s, got %s", checkoutDir, runner.calls[0].dir)
	}
}

func TestInstallGoSourceMissingCheckoutFails(t *testing.T) {
	paths := testPaths(t)
	c := manifest.Component{
		Name:    "howlchangeops",
		Version: "0.1.0",
		Install: manifest.Install{
			Method: manifest.MethodSourceBuild,
			SourceBuild: &manifest.SourceBuild{
				CheckoutName: "howlchangeops",
				Language:     "go",
				Go:           &manifest.GoSourceBuild{Package: "./adapter", BuildOutput: "howlchangeops"},
			},
		},
	}
	inst := &Installer{Runner: &fakeRunner{}, Getenv: func(string) string { return "" }}
	if err := inst.Install(context.Background(), c, paths); err == nil {
		t.Fatal("expected error when the source checkout cannot be located")
	}
}

func TestInstallGoSourceExtraStepsRunAgainstDependency(t *testing.T) {
	checkoutDir := t.TempDir()
	paths := testPaths(t)

	// Pre-activate a fake "howlframe" so the extra_steps resolver can find
	// its binary, the way it would after a real dependency-ordered install.
	frameDir := paths.ComponentReleaseDir("howlframe", "0.1.1")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frameDir, platform.ExeName("howlframe")), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ActivateRelease(paths, "howlframe", "0.1.1"); err != nil {
		t.Fatal(err)
	}

	c := manifest.Component{
		Name:    "howlchangeops",
		Version: "0.1.0",
		Install: manifest.Install{
			Method: manifest.MethodSourceBuild,
			SourceBuild: &manifest.SourceBuild{
				CheckoutName: "howlchangeops",
				Language:     "go",
				Go: &manifest.GoSourceBuild{
					Package:     "./adapter",
					BuildOutput: "howlchangeops",
					ExtraSteps: []manifest.BuildStep{
						{RunComponent: "howlframe", Args: []string{"check", "src/howlchangeops.howl"}},
					},
				},
			},
		},
	}

	runner := &touchingRunner{}
	inst := &Installer{Runner: runner, ExplicitCheckoutDirs: map[string]string{"howlchangeops": checkoutDir}}

	if err := inst.Install(context.Background(), c, paths); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("expected go build + 1 extra step, got %d calls: %+v", len(runner.calls), runner.calls)
	}
	extra := runner.calls[1]
	if extra.args[0] != "check" || extra.args[1] != "src/howlchangeops.howl" {
		t.Errorf("unexpected extra step invocation: %+v", extra)
	}
}
