# Ecosystem Release Manifest

The release manifest (`internal/manifest/default_ecosystem.toml`, embedded
into the `howl` binary via `go:embed`) is the single source of truth for
what a given Howl ecosystem release installs. It describes a *tested,
compatible combination* of component versions — not "whatever is newest in
each repository right now."

A local `ecosystem.toml` found by searching upward from the current
directory (or passed via `--manifest`) overrides the embedded default. This
is meant for a pinned or custom manifest; it is not how a normal install
resolves its manifest.

## Top-Level Fields

```toml
schema_version = 1

[ecosystem]
name = "Howl"
version = "0.1.0"      # ecosystem release version
channel = "stable"     # stable | beta | dev
description = "..."

[[optional_capabilities]]
name = "ollama"
required = false
capability = "local-inference"
```

`optional_capabilities` are ecosystem-wide (not tied to a specific
component) and are only checked under the `local-ai` profile.

## Component Fields

```toml
[[components]]
name = "howlframe"                 # required, unique, used as the binary name
display_name = "HowlFrame"         # optional, defaults to name
role = "..."                       # required, human-readable
repository = "https://github.com/..."
version = "0.1.0"                  # required, major.minor.patch
optional = false
platforms = ["linux", "darwin", "windows"]
archs = ["amd64", "arm64"]

[components.install]
method = "github_release"          # or "source_build"

[components.health_check]
type = "exec_version"              # exec_version | binary_exists | python_import

[[components.depends_on]]
component = "howlframe"
min_version = "0.1.0"              # optional compatibility constraint

[[components.external_dependencies]]
name = "go"
required = true
capability = "source-build"
```

Every component's binary is located by the uniform convention
`<release-dir>/<component-name>` (or `.exe` on Windows) regardless of
install method — see `docs/ARCHITECTURE.md`'s activation model.

### `install.method = "github_release"`

```toml
[components.install.github_release]
repository = "howlcipher/howlframe"                       # owner/repo
artifact_pattern = "howlframe_{version}_{os}_{arch}.{ext}" # {version} includes the "v" prefix
checksum_file = "SHA256SUMS"
```

The download URL is
`https://github.com/<repository>/releases/download/v<version>/<artifact>`.
The artifact and its `SHA256SUMS` are both downloaded, the checksum is
verified before extraction, and extraction rejects path traversal,
absolute paths, and symlink/hardlink entries. `{os}`/`{arch}` resolve to Go's
own `GOOS`/`GOARCH` names; `{ext}` is `zip` on Windows and `tar.gz`
everywhere else.

### `install.method = "source_build"`

```toml
[components.install.source_build]
checkout_name = "howlchangeops"   # sibling directory name (see devlocate)
language = "go"                   # or "python"

[components.install.source_build.go]
package = "./adapter"
build_output = "howlchangeops"

[[components.install.source_build.go.extra_steps]]
run_component = "howlframe"       # must already be installed (depends_on enforces this)
args = ["build", "src/howlchangeops.howl"]
```

or

```toml
[components.install.source_build.python]
package_dir = "."
extras = ["dev"]
console_script = "howlwriter"
min_python = "3.11.0"
```

`source_build` is a developer-machine-only install path: it requires a
local sibling checkout (see `internal/devlocate`) and, for Go components, a
system Go toolchain. See `docs/ARCHITECTURE.md`'s Known v1 Limitations for
why three of the four current components use it instead of
`github_release`.

### Health Checks

- `binary_exists`: the activated executable exists and (on non-Windows) is
  executable. The conservative default for components without a
  confirmed-safe self-check invocation.
- `exec_version`: runs the activated binary with `args` (default
  `["--version"]`) and requires non-empty output.
- `python_import`: imports `module` using the component's managed
  virtualenv interpreter.

## Validation

`manifest.Load`/`LoadBytes` validate: required fields present, a parseable
`major.minor.patch` version on the ecosystem and every component, declared
platforms/archs are ones Howl recognizes, the install method's
method-specific block is complete, the health check type is known and
carries its own required fields, every `depends_on` target exists in the
manifest, no dependency is self-referential or part of a cycle
(`Manifest.TopoOrder`), and every `min_version` compatibility constraint is
satisfied by the manifest's own declared versions. Compatibility is
re-checked against *installed* versions at runtime by `howl doctor`.
