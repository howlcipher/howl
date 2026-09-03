# Howl Architecture

## Ecosystem Context

Howl installs and manages the lifecycle of the following components. This
table also records the responsibility each repository owns for itself —
useful for understanding *why* Howl needs to install it, but none of this
is functionality Howl re-implements or routes through.

| Repository | Role | Status |
| :--- | :--- | :--- |
| **[`howlframe`](https://github.com/howlcipher/howlframe)** | Language, HFIR verification gate, and capability-bounded VM | Active |
| **[`howlchangeops`](https://github.com/howlcipher/howlchangeops)** | Authority boundary and release controller (HMAC human approvals, bounded Git mutations) | Active |
| **[`howlplane`](https://github.com/howlcipher/howlplane)** | AI engineering control plane (task routing, evidence ledgers) | Active |
| **[`howlwriter`](https://github.com/howlcipher/howlwriter)** | Writing control and review system | Active |

```text
Howl Installer
      |
      v installs (never imports or requires at build time)
      |
HowlFrame  --->  HowlChangeOps  --->  HowlPlane  --->  HowlWriter
(no deps)        (needs howlframe    (needs howlchangeops   (needs howlplane
                  binary at build     binary at runtime)      source tree —
                  + runtime)                                  see Known
                                                                Limitations)
```

This install order is derived from evidence in each repository, not
invented: HowlChangeOps's own README documents building against a
`howlframe` binary; HowlPlane resolves a `howlchangeops` binary at runtime
(`src/control_plane/executor.py`); HowlWriter's integration bridge inserts
HowlPlane's source tree onto `sys.path` at runtime.

## Language Decision

Howl is implemented in **Go**, evolving the existing Cobra-based CLI rather
than rewriting it. This was a deliberate evaluation, not a default:

- Go already produces standalone, dependency-free cross-platform binaries —
  required for a `curl | sh`-style bootstrap and for self-update.
- HowlFrame (a sibling ecosystem component) already proves the exact
  release pattern Howl needs for itself: cross-compiled linux/darwin/windows
  x86_64/arm64 binaries, `SHA256SUMS`, published via GitHub Releases.
- The existing `howl` codebase was already Go + Cobra; a rewrite would not
  have materially improved distribution, reliability, or maintainability
  over evolving it, so no rewrite was performed.

Go is a **developer-only** dependency for Howl's own build. End users
install a prebuilt `howl` binary and never need a Go toolchain to run
`howl install` — with one caveat, documented under Known v1 Limitations
below: building three of the four *managed components* from source
currently requires Go and/or Python on the machine, because those
components don't yet publish their own release artifacts.

## Package Layout

```text
cmd/howl/              main() — thin entry point
pkg/cmd/                Cobra command definitions (CLI layer only)
internal/
  platform/             OS + architecture + Bazzite/Distrobox detection,
                         XDG-compliant path resolution, executable naming
  manifest/              release manifest schema, parsing, validation,
                         embedded default ecosystem release
  state/                 installer state file + installation lock
  plan/                  typed installation/update operations, dependency
                         ordering, human-readable plan rendering
  engine/                 executes a plan: stage, verify, activate,
                         health-check, commit state, capture rollback data
  artifact/               HTTP download, SHA-256 verification, safe
                         tar.gz/zip extraction
  pyruntime/              Python venv provisioning and pip install for
                         Python-based components
  devlocate/              sibling source-checkout locator, used only by the
                         source_build install method
  component/               per-component installer strategies (GitHub
                         release download vs. build-from-source)
  health/                   declarative component health-check runner
  selfupdate/               Howl's own binary self-update
  doctor/                   installer-owned diagnostics + repair
  status/                   installer-level status reporting
  version/                  build metadata (unchanged from v0.1)
```

Planning is separated from execution throughout: `plan` produces an
inspectable, typed list of operations before anything touches the
filesystem; `engine` is the only package that performs mutation, and always
against a plan that was rendered and (outside `--yes`) approved first.

## Filesystem Layout

Howl is a user-scoped, XDG-compliant installation. It respects
`XDG_DATA_HOME`, `XDG_CONFIG_HOME`, and `XDG_CACHE_HOME` when set.

```text
~/.local/bin/howl                          the howl binary itself
~/.config/howl/config.toml                 channel, default profile
~/.local/share/howl/state/state.json       installer + component state
~/.local/share/howl/state/lock             installation lock
~/.local/share/howl/components/<name>/
    releases/<version>/...                 staged, immutable per version
    current -> releases/<version>          atomic activation pointer
~/.local/share/howl/runtimes/<name>/venv   isolated per-component Python venv
~/.cache/howl/downloads/                   staged downloads pending verification
```

Every path Howl writes to lives under one of these roots. `howl uninstall`
proves data ownership by path prefix before removing anything.

## Known v1 Limitations

- **Only HowlFrame has a real `github_release` install path today.**
  HowlChangeOps, HowlPlane, and HowlWriter have no published release
  artifacts yet, so Howl installs them via `source_build` (compiling or
  `pip install`-ing from the local sibling checkout). This means a system Go
  toolchain and Python 3 interpreter are effectively required in the
  standard profile right now, not developer-only — a known, documented
  deviation from the target end state. See `docs/SCOPE.md` for what would
  resolve this.
- **HowlWriter's dependency on HowlPlane is a hardcoded absolute path** in
  HowlWriter's own source (a `sys.path` insertion pointing at a specific
  dev-machine checkout location). Howl does not patch other repositories'
  source; `howl doctor` surfaces this as a compatibility note instead of
  silently working around it.
- **macOS and Windows** get platform detection, path handling, and
  cross-compiled builds, but are not runtime-verified — see the support
  matrix in the README.
