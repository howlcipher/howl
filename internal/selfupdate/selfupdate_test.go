package selfupdate

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
	"runtime"
	"testing"

	"github.com/howlcipher/howl/internal/artifact"
)

func buildTarGzWithBinary(t *testing.T, name string, content []byte) string {
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
	path := filepath.Join(t.TempDir(), "howl.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckFindsMatchingArtifact(t *testing.T) {
	artifactName, _ := artifactNameFor("v1.1.0")

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/howlcipher/howl/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v1.1.0","assets":[
			{"name":%q,"browser_download_url":"http://example.invalid/%s"},
			{"name":"SHA256SUMS","browser_download_url":"http://example.invalid/SHA256SUMS"}
		]}`, artifactName, artifactName)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := HTTPGithubAPI{Downloader: artifact.HTTPDownloader{Client: srv.Client()}, BaseURL: srv.URL}
	rel, err := Check(context.Background(), api, "howlcipher/howl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.Version != "1.1.0" {
		t.Errorf("expected version 1.1.0, got %s", rel.Version)
	}
	if rel.ArtifactName != artifactName {
		t.Errorf("expected artifact %s, got %s", artifactName, rel.ArtifactName)
	}
}

func TestCheckNoMatchingArtifactForPlatform(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/howlcipher/howl/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v1.1.0","assets":[{"name":"howl_v1.1.0_plan9_amd64.tar.gz","browser_download_url":"http://example.invalid/x"}]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := HTTPGithubAPI{Downloader: artifact.HTTPDownloader{Client: srv.Client()}, BaseURL: srv.URL}
	_, err := Check(context.Background(), api, "howlcipher/howl")
	if err == nil {
		t.Fatal("expected error when no matching platform artifact exists")
	}
}

func TestCheckNoReleaseFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/howlcipher/howl/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := HTTPGithubAPI{Downloader: artifact.HTTPDownloader{Client: srv.Client()}, BaseURL: srv.URL}
	_, err := Check(context.Background(), api, "howlcipher/howl")
	if err == nil {
		t.Fatal("expected error for empty release response")
	}
}

func TestApplyDownloadsVerifiesAndReplacesBinary(t *testing.T) {
	newContent := []byte("#!/bin/sh\necho new howl\n")
	binaryName := "howl"
	if runtime.GOOS == "windows" {
		binaryName = "howl.exe"
	}
	tarPath := buildTarGzWithBinary(t, binaryName, newContent)
	archiveBytes, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	artifactName, _ := artifactNameFor("v1.1.0")
	checksumLine := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), artifactName)

	mux := http.NewServeMux()
	mux.HandleFunc("/artifact", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tarPath)
	})
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumLine))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := &ReleaseInfo{
		Version:      "1.1.0",
		ArtifactURL:  srv.URL + "/artifact",
		ChecksumURL:  srv.URL + "/checksums",
		ArtifactName: artifactName,
	}

	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "howl")
	oldContent := []byte("old howl binary")
	if err := os.WriteFile(exePath, oldContent, 0o755); err != nil {
		t.Fatal(err)
	}
	cacheDir := t.TempDir()

	dl := artifact.HTTPDownloader{Client: srv.Client()}
	if err := Apply(context.Background(), dl, rel, exePath, cacheDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("expected new binary at exePath: %v", err)
	}
	if string(data) != string(newContent) {
		t.Errorf("expected new binary content, got %q", data)
	}

	prevData, err := os.ReadFile(exePath + ".prev")
	if err != nil {
		t.Fatalf("expected previous binary preserved: %v", err)
	}
	if string(prevData) != string(oldContent) {
		t.Errorf("expected preserved previous binary to match old content, got %q", prevData)
	}
}

func TestApplyRejectsChecksumMismatch(t *testing.T) {
	tarPath := buildTarGzWithBinary(t, "howl", []byte("content"))
	artifactName, _ := artifactNameFor("v1.1.0")

	mux := http.NewServeMux()
	mux.HandleFunc("/artifact", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tarPath)
	})
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000  " + artifactName + "\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := &ReleaseInfo{Version: "1.1.0", ArtifactURL: srv.URL + "/artifact", ChecksumURL: srv.URL + "/checksums", ArtifactName: artifactName}

	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "howl")
	oldContent := []byte("old howl binary")
	if err := os.WriteFile(exePath, oldContent, 0o755); err != nil {
		t.Fatal(err)
	}

	dl := artifact.HTTPDownloader{Client: srv.Client()}
	err := Apply(context.Background(), dl, rel, exePath, t.TempDir())
	if err == nil {
		t.Fatal("expected checksum mismatch to be rejected")
	}

	// The current binary must be completely untouched on a failed apply.
	data, err := os.ReadFile(exePath)
	if err != nil || string(data) != string(oldContent) {
		t.Errorf("expected original binary left untouched, err=%v data=%q", err, data)
	}
	if _, err := os.Stat(exePath + ".prev"); !os.IsNotExist(err) {
		t.Errorf("expected no .prev file to be created on a failed apply")
	}
}
