// Package artifact handles fetching, integrity-verifying, and safely
// extracting release artifacts. Every downloaded byte is treated as
// untrusted until its checksum is verified; nothing is executed or
// extracted before that.
package artifact

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Downloader fetches a URL and returns its body. Implementations are
// injected so download logic is testable against an httptest server
// instead of the real network.
type Downloader interface {
	Download(ctx context.Context, url string) (io.ReadCloser, error)
}

// HTTPDownloader is the real Downloader used outside tests.
type HTTPDownloader struct {
	Client *http.Client
}

// NewHTTPDownloader returns an HTTPDownloader with a sane default timeout.
func NewHTTPDownloader() HTTPDownloader {
	return HTTPDownloader{Client: &http.Client{Timeout: 5 * time.Minute}}
}

// Download implements Downloader.
func (d HTTPDownloader) Download(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request for %s: %w", url, err)
	}
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download %s: unexpected status %s", url, resp.Status)
	}
	return resp.Body, nil
}

// DownloadToFile streams a URL to destPath, writing to a temp file in the
// same directory and renaming into place so a failed or interrupted
// download never leaves a partially-written file at destPath.
func DownloadToFile(ctx context.Context, dl Downloader, url, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	body, err := dl.Download(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".download-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary download file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write downloaded artifact: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close downloaded artifact: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to stage downloaded artifact: %w", err)
	}
	return nil
}
