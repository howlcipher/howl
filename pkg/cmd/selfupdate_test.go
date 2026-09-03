package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/howlcipher/howl/internal/version"
)

func startFakeSelfUpdateAPI(t *testing.T, newerVersion string) *httptest.Server {
	t.Helper()
	artifactName := fmt.Sprintf("howl_v%s_%s_%s.tar.gz", newerVersion, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		artifactName = fmt.Sprintf("howl_v%s_%s_%s.zip", newerVersion, runtime.GOOS, runtime.GOARCH)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/howlcipher/howl/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v%s","assets":[
			{"name":%q,"browser_download_url":"IGNORED"},
			{"name":"SHA256SUMS","browser_download_url":"IGNORED"}
		]}`, newerVersion, artifactName)
	})
	return httptest.NewServer(mux)
}

func TestUpdateCheckReportsInstallerUpdate(t *testing.T) {
	sandboxHowlPaths(t)
	newer := "99.0.0" // guaranteed different from the built-in dev version
	srv := startFakeSelfUpdateAPI(t, newer)
	defer srv.Close()
	t.Setenv("HOWL_SELFUPDATE_API_BASE_URL", srv.URL)

	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"update", "--check", "--manifest", manifestFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	want := fmt.Sprintf("%s -> %s", version.GetInfo().Version, newer)
	if !strings.Contains(out, want) {
		t.Errorf("expected installer version delta %q in output, got:\n%s", want, out)
	}
}

func TestUpdateCheckNoInstallerUpdateWhenAPIUnreachable(t *testing.T) {
	sandboxHowlPaths(t) // already points HOWL_SELFUPDATE_API_BASE_URL at a dead port

	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "ecosystem.toml")
	if err := os.WriteFile(manifestFile, []byte(testManifestTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := mustResolveTestPaths(t)
	seedInstalledState(t, paths, "howlplane", "1.0.0")

	rootCmd := NewRootCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"update", "--check", "--manifest", manifestFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected self-update check failure to degrade gracefully, got error: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("expected graceful up-to-date report when the self-update API is unreachable, got:\n%s", buf.String())
	}
}
