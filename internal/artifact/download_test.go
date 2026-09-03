package artifact

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadToFileSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello artifact"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := DownloadToFile(context.Background(), NewHTTPDownloader(), srv.URL, dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello artifact" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestDownloadToFileHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := DownloadToFile(context.Background(), NewHTTPDownloader(), srv.URL, dest)
	if err == nil {
		t.Fatal("expected error on 404")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("expected no partial file left behind on failed download")
	}
}

func TestDownloadToFileNeverLeavesPartialFileOnMidStreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("only a little data"))
		// Hijack and close the connection abruptly to simulate a
		// mid-stream failure, if supported.
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				conn.Close()
			}
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	_ = DownloadToFile(context.Background(), NewHTTPDownloader(), srv.URL, dest)
	// Whether or not io.Copy detected the truncation as an error, destPath
	// must never exist as a partial file -- DownloadToFile only renames
	// into place after a fully successful copy.
	if _, statErr := os.Stat(dest); statErr == nil {
		data, _ := os.ReadFile(dest)
		if len(data) < 1000000 {
			t.Errorf("expected either no file or a complete file, got a %d-byte partial file", len(data))
		}
	}
}
