# Howl

**Howl is the installer and lifecycle manager for the Howl ecosystem.**

That's it. Howl installs the Howl ecosystem, keeps its components at
compatible versions, updates them safely, checks and repairs installation
health, rolls back a bad update, and uninstalls cleanly. It is not an AI
application, a control plane, an agent framework, or a general-purpose
DevOps tool — see [`docs/SCOPE.md`](docs/SCOPE.md) for the full boundary.

```bash
howl install
howl status
howl update --check
howl update
howl doctor
howl rollback
howl uninstall
```

A user on a clean, supported machine should be able to install the `howl`
binary, run `howl install`, and end up with a working Howl ecosystem —
without manually cloning repositories, picking compatible versions,
creating Python virtual environments, or figuring out install order.

---

## What Howl Installs

The default (`standard`) profile installs the current compatible Howl
ecosystem release:

| Component | Role |
| :--- | :--- |
| **HowlFrame** | Language, HFIR verification gate, capability-bounded VM |
| **HowlChangeOps** | Authority boundary and release controller |
| **HowlPlane** | AI engineering control plane |
| **HowlWriter** | Writing control and review system |

These are installed in dependency order (HowlFrame first; HowlWriter last),
resolved from a versioned **release manifest**, not "whatever is newest on
each repository right now." See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
for why that order is required and [`docs/MANIFEST.md`](docs/MANIFEST.md)
for the manifest schema.

## Installation Profiles

- **`standard`** (default) — the full ecosystem and the runtimes/
  dependencies it requires. No Go toolchain, no Ollama, no local models
  required by Howl itself.
- **`local-ai`** — `standard`, plus detection of local-inference
  prerequisites (currently: Ollama). Missing optional capabilities are
  reported, not installed automatically, and never block the rest of the
  install.
- **`developer`** — `standard`, plus the toolchains needed to build and test
  the ecosystem's own repositories (Go, Python dev tooling).

```bash
howl install --profile local-ai
howl install --profile developer
```

## Dependency Handling

Howl divides dependencies into three categories and treats each
differently — see `docs/SCOPE.md` for the full policy:

1. **Howl-managed components** — the four repositories above. Howl owns
   their install, update, rollback, and removal completely.
2. **Howl-managed runtimes** — isolated Python virtual environments Howl
   creates per component under `~/.local/share/howl/runtimes/`. Howl never
   installs into or modifies your system Python.
3. **External/system dependencies** — Git, Go, Python 3, and (only under
   `local-ai`) Ollama. Howl detects these and reports what's missing; it
   does not silently modify your operating system or run `sudo`.

A missing **required** dependency stops installation with a clear,
actionable message. A missing **optional** dependency (Ollama) is reported
as an unavailable capability — the rest of the ecosystem still installs.

## Bazzite and Immutable/Atomic Linux

Bazzite is a first-class target. Howl detects it explicitly (rather than
treating it as generic Fedora) and never runs `rpm-ostree install`, layers
packages onto the base image, or otherwise modifies the immutable system.
Everything Howl does lives under your user-scoped `~/.local` and
`~/.config` directories. Howl also detects Distrobox/devbox-style
containerized dev environments and targets dependency checks at the
environment a component will actually run in, not blindly at the host OS.

## Commands

```bash
howl install [component] [--profile standard|local-ai|developer] [--yes] [--channel <name>]
howl status [--json]
howl update --check [--json]
howl update [--yes]
howl rollback [component] [--yes]
howl doctor [-v] [--json] [--strict] [--fix]
howl uninstall [component] [--purge] [--yes]
howl channel [stable|beta|dev]
howl version [--json]
```

`howl install` and `howl update` always show a plan before touching your
machine and require `--yes` to run non-interactively. Running `howl
install` again after a successful install is a safe no-op for anything
already at the target version.

## Filesystem Layout

Howl is entirely user-scoped and XDG-compliant (respects `XDG_DATA_HOME`,
`XDG_CONFIG_HOME`, `XDG_CACHE_HOME`). Full layout in
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md#filesystem-layout).

## Security Model

- Every downloaded artifact is SHA-256 verified before use; a mismatch
  fails closed and nothing is extracted or executed.
- Archive extraction rejects path traversal, absolute paths, and symlink
  escapes, and only ever writes into Howl's own staging directories.
- No installer operation shells out through `sh -c`; external commands run
  with explicit argument lists.
- Howl never requests elevated privileges by default; nothing in the
  current dependency set requires root.

## Developer Workflow

```bash
go build -o bin/howl ./cmd/howl
go test ./...
gofmt -l .
go vet ./...
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) before proposing new
functionality — Howl's scope is intentionally narrow.

## Support Matrix

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md#known-v1-limitations) and
the implementation report in `docs/` for exactly what has been runtime
verified versus compile-only per platform.

## License

MIT License. See [LICENSE](LICENSE) for details.
