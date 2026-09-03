// Package platform detects the operating system, architecture, and execution
// environment Howl is running in, and resolves the standard filesystem
// locations Howl is allowed to write to. It is intentionally narrow: it
// answers "what machine/container am I on" and "where do I write files",
// nothing about what to do with that information.
package platform

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ContainerKind identifies the kind of containerized development
// environment Howl is executing inside, if any.
type ContainerKind string

const (
	ContainerNone      ContainerKind = ""
	ContainerDistrobox ContainerKind = "distrobox"
	ContainerToolbox   ContainerKind = "toolbox"
	ContainerGeneric   ContainerKind = "container"
)

// OSRelease is the subset of /etc/os-release Howl cares about.
type OSRelease struct {
	ID         string   `json:"id"`
	IDLike     []string `json:"id_like,omitempty"`
	PrettyName string   `json:"pretty_name,omitempty"`
}

// Info describes the detected platform and execution environment.
type Info struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`

	Distro       OSRelease `json:"distro"`
	IsBazzite    bool      `json:"is_bazzite"`
	IsAtomic     bool      `json:"is_atomic"`
	IsDebianLike bool      `json:"is_debian_like"`

	Container     ContainerKind `json:"container,omitempty"`
	ContainerName string        `json:"container_name,omitempty"`

	// Host* fields are best-effort and only populated when Howl detects it
	// is running inside a container that exposes host information (e.g. a
	// Distrobox container bind-mounting /run/host).
	HostDistro    *OSRelease `json:"host_distro,omitempty"`
	HostIsBazzite bool       `json:"host_is_bazzite,omitempty"`
	HostIsAtomic  bool       `json:"host_is_atomic,omitempty"`

	Supported         bool   `json:"supported"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

// supportedPlatforms is the set of OS/arch combinations Howl handles at
// all. This is a floor, not the support matrix (see docs/ARCHITECTURE.md
// for which of these are runtime-verified vs. compile-only).
var supportedPlatforms = map[string]map[string]bool{
	"linux":   {"amd64": true, "arm64": true},
	"darwin":  {"amd64": true, "arm64": true},
	"windows": {"amd64": true, "arm64": true},
}

// Detect inspects the real running machine.
func Detect() Info {
	return DetectAt("/", os.Getenv)
}

// DetectAt inspects a machine rooted at root using getenv for environment
// lookups, so tests can point Howl at fixture directories instead of the
// real filesystem.
func DetectAt(root string, getenv func(string) string) Info {
	info := Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	if archs, ok := supportedPlatforms[info.OS]; ok && archs[info.Arch] {
		info.Supported = true
	} else {
		info.Supported = false
		info.UnsupportedReason = "unsupported OS/architecture combination: " + info.OS + "/" + info.Arch
	}

	if info.OS != "linux" {
		return info
	}

	if rel, ok := readOSRelease(filepath.Join(root, "etc", "os-release")); ok {
		info.Distro = rel
		info.IsBazzite = isBazzite(rel)
		info.IsDebianLike = isDebianLike(rel)
	}
	info.IsAtomic = fileExists(filepath.Join(root, "run", "ostree-booted"))

	info.Container, info.ContainerName = detectContainer(root, getenv)
	if info.Container != ContainerNone {
		hostOSRelease := filepath.Join(root, "run", "host", "etc", "os-release")
		if hostRel, ok := readOSRelease(hostOSRelease); ok {
			info.HostDistro = &hostRel
			info.HostIsBazzite = isBazzite(hostRel)
		}
		info.HostIsAtomic = fileExists(filepath.Join(root, "run", "host", "run", "ostree-booted"))
	}

	return info
}

func isBazzite(rel OSRelease) bool {
	if strings.EqualFold(rel.ID, "bazzite") {
		return true
	}
	return strings.Contains(strings.ToLower(rel.PrettyName), "bazzite")
}

func isDebianLike(rel OSRelease) bool {
	if strings.EqualFold(rel.ID, "debian") || strings.EqualFold(rel.ID, "ubuntu") {
		return true
	}
	for _, like := range rel.IDLike {
		if strings.EqualFold(like, "debian") {
			return true
		}
	}
	return false
}

// detectContainer determines whether Howl is running inside a Distrobox,
// Toolbox, or other generic container, using the most specific signal
// available. DISTROBOX_ENTER_PATH is set exclusively by the distrobox-enter
// launcher and is the strongest signal; /run/.containerenv and
// /run/.toolboxenv are marker files left by podman/toolbox respectively.
func detectContainer(root string, getenv func(string) string) (ContainerKind, string) {
	containerEnvPath := filepath.Join(root, "run", ".containerenv")
	toolboxEnvPath := filepath.Join(root, "run", ".toolboxenv")
	dockerEnvPath := filepath.Join(root, ".dockerenv")

	name := ""
	if content, err := os.ReadFile(containerEnvPath); err == nil {
		name = parseContainerenvName(string(content))
	}
	if name == "" {
		name = getenv("CONTAINER_ID")
	}

	switch {
	case getenv("DISTROBOX_ENTER_PATH") != "":
		return ContainerDistrobox, name
	case fileExists(toolboxEnvPath):
		return ContainerToolbox, name
	case fileExists(containerEnvPath) || fileExists(dockerEnvPath):
		return ContainerGeneric, name
	default:
		return ContainerNone, ""
	}
}

func parseContainerenvName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name=") {
			return strings.Trim(strings.TrimPrefix(line, "name="), `"`)
		}
	}
	return ""
}

func readOSRelease(path string) (OSRelease, bool) {
	f, err := os.Open(path)
	if err != nil {
		return OSRelease{}, false
	}
	defer f.Close()

	rel := OSRelease{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, `"`)
		switch key {
		case "ID":
			rel.ID = val
		case "ID_LIKE":
			rel.IDLike = strings.Fields(val)
		case "PRETTY_NAME":
			rel.PrettyName = val
		}
	}
	if rel.ID == "" && rel.PrettyName == "" {
		return OSRelease{}, false
	}
	return rel, true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ExeName returns base with the platform-appropriate executable suffix
// (".exe" on Windows).
func ExeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}
