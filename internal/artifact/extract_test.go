package artifact

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Mode:     0o644,
			Size:     int64(len(e.content)),
		}
		if e.typeflag == tar.TypeDir {
			hdr.Mode = 0o755
			hdr.Size = 0
		}
		if e.typeflag == tar.TypeSymlink {
			hdr.Linkname = e.linkname
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	tw.Close()
	gz.Close()

	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

type tarEntry struct {
	name     string
	content  string
	typeflag byte
	linkname string
}

func TestExtractTarGzValid(t *testing.T) {
	src := writeTarGz(t, []tarEntry{
		{name: "bin/", typeflag: tar.TypeDir},
		{name: "bin/howlframe", content: "binary contents", typeflag: tar.TypeReg},
	})
	destDir := t.TempDir()

	if err := ExtractTarGz(src, destDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "bin", "howlframe"))
	if err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
	if string(data) != "binary contents" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	src := writeTarGz(t, []tarEntry{
		{name: "../../etc/passwd", content: "pwned", typeflag: tar.TypeReg},
	})
	destDir := t.TempDir()

	err := ExtractTarGz(src, destDir)
	if err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(destDir), "etc", "passwd")); !os.IsNotExist(statErr) {
		t.Fatal("path traversal entry must never be written to disk")
	}
}

func TestExtractTarGzRejectsAbsolutePath(t *testing.T) {
	src := writeTarGz(t, []tarEntry{
		{name: "/etc/passwd", content: "pwned", typeflag: tar.TypeReg},
	})
	destDir := t.TempDir()

	if err := ExtractTarGz(src, destDir); err == nil {
		t.Fatal("expected absolute path entry to be rejected")
	}
}

func TestExtractTarGzRejectsSymlinkEscape(t *testing.T) {
	src := writeTarGz(t, []tarEntry{
		{name: "evil-link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
	})
	destDir := t.TempDir()

	if err := ExtractTarGz(src, destDir); err == nil {
		t.Fatal("expected symlink entry to be rejected")
	}
	if _, statErr := os.Lstat(filepath.Join(destDir, "evil-link")); !os.IsNotExist(statErr) {
		t.Fatal("symlink entry must never be created")
	}
}

func writeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	zw.Close()

	path := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractZipValid(t *testing.T) {
	src := writeZip(t, map[string]string{"howl.exe": "windows binary"})
	destDir := t.TempDir()

	if err := ExtractZip(src, destDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "howl.exe"))
	if err != nil || string(data) != "windows binary" {
		t.Fatalf("expected extracted file, err=%v data=%q", err, data)
	}
}

func TestExtractZipRejectsPathTraversal(t *testing.T) {
	src := writeZip(t, map[string]string{"../../evil.txt": "pwned"})
	destDir := t.TempDir()

	if err := ExtractZip(src, destDir); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}
