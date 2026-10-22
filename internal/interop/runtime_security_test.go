package interop

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func init() {
	mode := os.Getenv("HOWL_DESCRIPTOR_TEST_HELPER")
	if mode == "" {
		return
	}
	if mode == "child" {
		time.Sleep(300 * time.Millisecond)
		if path := os.Getenv("HOWL_DESCRIPTOR_MARKER"); path != "" {
			_ = os.WriteFile(path, []byte("descendant survived"), 0600)
		}
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}
	if mode == "descendant" || mode == "orphan" {
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(), "HOWL_DESCRIPTOR_TEST_HELPER=child")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			os.Exit(1)
		}
		if mode == "descendant" {
			time.Sleep(2 * time.Second)
		}
		os.Exit(0)
	}
	for i := 0; i < 300000; i++ {
		stream := os.Stdout
		if mode == "stderr" {
			stream = os.Stderr
		}
		_, _ = stream.Write([]byte("synthetic descriptor output\n"))
	}
	os.Exit(0)
}
func TestDescriptorDescendantDeadline(t *testing.T) {
	for _, mode := range []string{"descendant", "orphan"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("HOWL_DESCRIPTOR_TEST_HELPER", mode)
			marker := t.TempDir() + "/marker"
			t.Setenv("HOWL_DESCRIPTOR_MARKER", marker)
			start := time.Now()
			_, err := (ExecRunner{Timeout: 100 * time.Millisecond}).Describe(context.Background(), os.Args[0])
			if err == nil {
				t.Fatal("expected deadline or inherited-pipe bound")
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Fatalf("descriptor exceeded bounded wait: %v", elapsed)
			}
			time.Sleep(400 * time.Millisecond)
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("descendant survived descriptor cleanup")
			}
		})
	}

}
func TestDescriptorOutputBound(t *testing.T) {
	for _, mode := range []string{"output", "stderr"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("HOWL_DESCRIPTOR_TEST_HELPER", mode)
			out, err := (ExecRunner{Timeout: time.Second}).Describe(context.Background(), os.Args[0])
			if err == nil || len(out) > 1<<20 {
				t.Fatalf("oversized descriptor accepted (bytes=%d, error=%v)", len(out), err)
			}
		})
	}
}
