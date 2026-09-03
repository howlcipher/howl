package artifact

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// safeJoin resolves name against destDir and rejects anything that would
// escape it: absolute paths and ".." traversal both fail closed.
func safeJoin(destDir, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("refusing to extract entry with absolute path: %q", name)
	}
	cleaned := filepath.Clean(filepath.Join(destDir, name))
	rel, err := filepath.Rel(destDir, cleaned)
	if err != nil {
		return "", fmt.Errorf("refusing to extract entry %q: %w", name, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to extract entry outside destination directory: %q", name)
	}
	return cleaned, nil
}

// ExtractTarGz safely extracts a gzip-compressed tar archive into destDir.
// Absolute paths, path traversal, and symlink/hardlink entries are all
// rejected -- extraction only ever writes plain files and directories
// under destDir.
func ExtractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", src, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to read gzip archive %s: %w", src, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry in %s: %w", src, err)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			target, err := safeJoin(destDir, hdr.Name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			target, err := safeJoin(destDir, hdr.Name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", target, err)
			}
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("refusing to extract link entry %q: links are not permitted in Howl release artifacts", hdr.Name)
		default:
			// Ignore anything else (device files, fifos, etc.) rather than
			// silently succeeding at extracting something unexpected.
			return fmt.Errorf("refusing to extract unsupported entry type for %q", hdr.Name)
		}
	}
}

// ExtractZip safely extracts a zip archive into destDir, with the same
// path-traversal and symlink protections as ExtractTarGz.
func ExtractZip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip archive %s: %w", src, err)
	}
	defer r.Close()

	for _, entry := range r.File {
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to extract symlink entry %q: links are not permitted in Howl release artifacts", entry.Name)
		}

		target, err := safeJoin(destDir, entry.Name)
		if err != nil {
			return err
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", target, err)
		}

		rc, err := entry.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry %q: %w", entry.Name, err)
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode(int64(entry.Mode().Perm())))
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file %s: %w", target, err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("failed to write file %s: %w", target, err)
		}
		out.Close()
		rc.Close()
	}
	return nil
}

func fileMode(mode int64) os.FileMode {
	m := os.FileMode(mode) & 0o777
	if m == 0 {
		return 0o644
	}
	return m
}
