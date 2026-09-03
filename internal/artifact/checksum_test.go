package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func sha256Hex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func TestVerifySHA256Valid(t *testing.T) {
	path := writeTestFile(t, "howlframe binary contents")
	if err := VerifySHA256(path, sha256Hex("howlframe binary contents")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySHA256Mismatch(t *testing.T) {
	path := writeTestFile(t, "actual content")
	err := VerifySHA256(path, sha256Hex("different content"))
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	var mismatch *ChecksumMismatchError
	if !asChecksumMismatch(err, &mismatch) {
		t.Fatalf("expected ChecksumMismatchError, got %T: %v", err, err)
	}
}

func TestVerifySHA256MissingChecksum(t *testing.T) {
	path := writeTestFile(t, "content")
	if err := VerifySHA256(path, ""); err == nil {
		t.Fatal("expected error when no checksum is provided: must fail closed")
	}
}

func TestParseChecksumFile(t *testing.T) {
	data := []byte(
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  howlframe_v0.1.1_linux_amd64.tar.gz\n" +
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  howlframe_v0.1.1_darwin_amd64.tar.gz\n",
	)
	sums, err := ParseChecksumFile(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sums["howlframe_v0.1.1_linux_amd64.tar.gz"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("expected parsed checksum for linux artifact, got %+v", sums)
	}
	if len(sums) != 2 {
		t.Errorf("expected 2 entries, got %d", len(sums))
	}
}

func TestParseChecksumFileMalformed(t *testing.T) {
	if _, err := ParseChecksumFile([]byte("not-a-valid-line\n")); err == nil {
		t.Fatal("expected error for malformed checksum line")
	}
	if _, err := ParseChecksumFile([]byte("short  file.tar.gz\n")); err == nil {
		t.Fatal("expected error for a digest that isn't 64 hex characters")
	}
}

func asChecksumMismatch(err error, target **ChecksumMismatchError) bool {
	if m, ok := err.(*ChecksumMismatchError); ok {
		*target = m
		return true
	}
	return false
}
