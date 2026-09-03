package artifact

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// ChecksumMismatchError is returned when a downloaded artifact's SHA-256
// does not match what was expected. Verification always fails closed:
// nothing is extracted or executed after this error.
type ChecksumMismatchError struct {
	Artifact string
	Expected string
	Actual   string
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf("checksum mismatch for %s: expected %s, got %s", e.Artifact, e.Expected, e.Actual)
}

// SHA256File computes the hex-encoded SHA-256 digest of the file at path.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open %s for checksum: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read %s for checksum: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifySHA256 checks that the file at path's SHA-256 matches
// expectedHex, failing closed on mismatch.
func VerifySHA256(path, expectedHex string) error {
	if strings.TrimSpace(expectedHex) == "" {
		return fmt.Errorf("no checksum provided for %s: refusing to trust unverified artifact", path)
	}
	actual, err := SHA256File(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expectedHex) {
		return &ChecksumMismatchError{Artifact: path, Expected: expectedHex, Actual: actual}
	}
	return nil
}

// ParseChecksumFile parses a `sha256sum`-format checksums file (the format
// `sha256sum FILES > SHA256SUMS` produces, and what howlframe's own
// release workflow publishes): one "<hex>  <filename>" pair per line.
func ParseChecksumFile(data []byte) (map[string]string, error) {
	sums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("malformed checksum file at line %d: %q", lineNo, line)
		}
		hexDigest := fields[0]
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if len(hexDigest) != 64 {
			return nil, fmt.Errorf("malformed checksum file at line %d: %q is not a sha256 digest", lineNo, hexDigest)
		}
		sums[name] = hexDigest
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse checksum file: %w", err)
	}
	return sums, nil
}
