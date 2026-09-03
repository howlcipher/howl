package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func envFrom(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func TestDetectDistroboxOnBazziteHost(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "etc", "os-release"), "ID=ubuntu\nID_LIKE=debian\nPRETTY_NAME=\"Ubuntu 26.04 LTS\"\n")
	writeFile(t, filepath.Join(root, "run", ".containerenv"), "engine=\"podman-5.8.4\"\nname=\"devbox\"\nimage=\"docker.io/library/ubuntu:26.04\"\n")
	writeFile(t, filepath.Join(root, "run", "host", "etc", "os-release"), "ID=bazzite\nID_LIKE=\"fedora\"\nNAME=\"Bazzite\"\n")
	writeFile(t, filepath.Join(root, "run", "host", "run", "ostree-booted"), "")

	info := DetectAt(root, envFrom(map[string]string{
		"DISTROBOX_ENTER_PATH": "/usr/bin/distrobox-enter",
		"container":            "podman",
	}))

	if info.OS != "linux" {
		t.Fatalf("expected OS override to be irrelevant to fixture-driven checks; got runtime OS %q", info.OS)
	}
	if info.Container != ContainerDistrobox {
		t.Errorf("expected Distrobox container, got %q", info.Container)
	}
	if info.ContainerName != "devbox" {
		t.Errorf("expected container name 'devbox', got %q", info.ContainerName)
	}
	if info.IsBazzite {
		t.Errorf("execution environment (Ubuntu container) must not be reported as Bazzite")
	}
	if !info.IsDebianLike {
		t.Errorf("expected execution environment to be detected as Debian-like")
	}
	if info.IsAtomic {
		t.Errorf("execution environment (container) must not be reported as atomic")
	}
	if info.HostDistro == nil || !info.HostIsBazzite {
		t.Fatalf("expected host to be detected as Bazzite, got %+v", info.HostDistro)
	}
	if !info.HostIsAtomic {
		t.Errorf("expected host to be detected as atomic (ostree-booted marker present)")
	}
}

func TestDetectGenericDebian(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "etc", "os-release"), "ID=debian\nPRETTY_NAME=\"Debian GNU/Linux 12 (bookworm)\"\n")

	info := DetectAt(root, envFrom(nil))

	if info.Container != ContainerNone {
		t.Errorf("expected no container detected, got %q", info.Container)
	}
	if !info.IsDebianLike {
		t.Errorf("expected debian to be detected as debian-like")
	}
	if info.IsBazzite {
		t.Errorf("debian must not be detected as bazzite")
	}
}

func TestDetectBareBazziteHost(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "etc", "os-release"), "ID=bazzite\nID_LIKE=\"fedora\"\nPRETTY_NAME=\"Bazzite\"\n")
	writeFile(t, filepath.Join(root, "run", "ostree-booted"), "")

	info := DetectAt(root, envFrom(nil))

	if !info.IsBazzite {
		t.Errorf("expected Bazzite detection")
	}
	if !info.IsAtomic {
		t.Errorf("expected atomic detection via ostree-booted marker")
	}
	if info.Container != ContainerNone {
		t.Errorf("expected no container on a bare host, got %q", info.Container)
	}
}

func TestDetectToolbox(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "etc", "os-release"), "ID=fedora\nPRETTY_NAME=\"Fedora Linux 40\"\n")
	writeFile(t, filepath.Join(root, "run", ".containerenv"), "name=\"fedora-toolbox-40\"\n")
	writeFile(t, filepath.Join(root, "run", ".toolboxenv"), "")

	info := DetectAt(root, envFrom(nil))

	if info.Container != ContainerToolbox {
		t.Errorf("expected Toolbox container, got %q", info.Container)
	}
}

func TestDetectUnknownOSRelease(t *testing.T) {
	root := t.TempDir() // no /etc/os-release at all
	info := DetectAt(root, envFrom(nil))

	if info.Distro.ID != "" {
		t.Errorf("expected empty distro on missing os-release, got %+v", info.Distro)
	}
	if info.IsBazzite || info.IsDebianLike {
		t.Errorf("must not guess distro identity when os-release is absent")
	}
}

func TestSupportedPlatforms(t *testing.T) {
	root := t.TempDir()
	info := DetectAt(root, envFrom(nil))
	// The running test binary's GOOS/GOARCH must be in the supported set
	// for this repository's own CI targets (linux/amd64 or linux/arm64).
	if !info.Supported {
		t.Fatalf("expected the current test platform to be supported, got unsupported_reason=%q", info.UnsupportedReason)
	}
}

func TestExeName(t *testing.T) {
	name := ExeName("howl")
	if name == "" {
		t.Fatal("expected non-empty exe name")
	}
}
