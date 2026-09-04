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

internal = false                   # true hides it from PATH (an engine another component drives)

[components.install]
method = "github_release"          # github_release | github_release_python_wheel | source_build

[components.developer_install]     # optional; only consulted under --profile developer
method = "source_build"

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
install method — see `docs/ARCHITECTURE.md`'s activation model. A component
marked `internal = true` (e.g. a Python engine another component's CLI
drives) is still installed, health-checked, and staged at that path, but
Howl never symlinks it onto the user's PATH.

`developer_install`, if set, is an entire second `Install` stanza used
instead of `install` when the active profile is `developer` (e.g. a
`source_build` fallback for a component whose standard path is
`github_release`). Standard-profile planning never reads this field, so a
missing or broken release artifact never silently falls back to it —
`--profile developer` must be requested explicitly.

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

### `install.method = "github_release_python_wheel"`

```toml
[components.install.github_release_python_wheel]
repository = "howlcipher/howlwriter"
artifact_pattern = "howlwriter-{pep440_version}-py3-none-any.whl" # no "v" prefix -- wheel filenames must be valid PEP 440
checksum_file = "SHA256SUMS"
console_script = "howlwriter"      # the entry point pyproject.toml's [project.scripts] declares
extras = ["web"]                   # baked in at install time, not into the wheel itself
min_python = "3.11.0"
```

The wheel and its checksum file are downloaded and verified exactly like a
`github_release` archive, but nothing is extracted: the wheel itself is
`pip install`ed (non-editable) into a Howl-managed, isolated virtualenv
under `<DataHome>/runtimes/<name>/venv`, the same venv layout
`source_build`'s Python variant uses. A thin wrapper script at
`<release-dir>/<component-name>` execs the venv's `console_script` entry
point, giving this method the same uniform binary location as every other.

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
system Go toolchain. As of the Release Artifacts v1 milestone, every
component in the default manifest uses `github_release` or
`github_release_python_wheel` under the standard profile; `source_build`
only remains reachable via each component's `developer_install`.

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
