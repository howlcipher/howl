# Howl Ecosystem

> **Website & Documentation:** https://howlcipher.github.io/howl/

**Howl** is a unified open-source ecosystem designed to bring mathematical rigor, capability boundaries, adversarial verification, and cryptographic human authority to AI-assisted software engineering.

The `howl` CLI is the canonical user-facing front door for the ecosystem.

---

## Axiom

> **Intent is not authority.**
> Probabilistic intelligence proposes; deterministic machinery controls execution; humans retain sovereign authority over consequential risk.

---

## Ecosystem Responsibility Model

| Repository | Role & Description | Documentation Site | Status |
| :--- | :--- | :--- | :--- |
| **[`howl`](https://github.com/howlcipher/howl)** | **Ecosystem Front Door & CLI:** Unified CLI entry point, component discovery, diagnostics, health, and cross-repo routing. | [howlcipher.github.io/howl](https://howlcipher.github.io/howl/) | Active |
| **[`howlframe`](https://github.com/howlcipher/howlframe)** | **Intelligence & Reasoning Framework:** AI DSL compiler, typed HFIR verification gate, and capability-bounded VM. | [howlcipher.github.io/howlframe](https://howlcipher.github.io/howlframe/) | Active |
| **[`howlplane`](https://github.com/howlcipher/howlplane)** | **AI Engineering Control Plane:** Deterministic multi-agent task routing, adversarial falsification, and evidence ledgers. | [howlcipher.github.io/howlplane](https://howlcipher.github.io/howlplane/) | Active |
| **[`howlnotes`](https://github.com/howlcipher/howlnotes)** | **Knowledge Notebook & Dogfood Consumer:** Full-stack notes application proving browser compilation and native store persistence. | [howlcipher.github.io/howlnotes](https://howlcipher.github.io/howlnotes/) | Active |
| **[`howlchangeops`](https://github.com/howlcipher/howlchangeops)** | **Authority Boundary & Release Controller:** Enforces HMAC cryptographic human approvals and bounded Git mutations. | [howlcipher.github.io/howlchangeops](https://howlcipher.github.io/howlchangeops/) | Active |
| **[`howlboard`](https://github.com/howlcipher/howlboard)** | **Evaluation Surface & Telemetry Console:** Deterministic task state machine proving compiler maturity through dogfooding. | [howlcipher.github.io/howlboard](https://howlcipher.github.io/howlboard/) | Active |
| **[`howlwriter`](https://github.com/howlcipher/howlwriter)** | **Writing Control & Review System:** Voice preservation, deterministic style linting, humanization, and adversarial review. | [howlcipher.github.io/howlwriter](https://howlcipher.github.io/howlwriter/) | Active |

---

## Architecture

### 1. CLI Routing Architecture
The `howl` CLI is the unprivileged, fast entry point and front door for ecosystem commands, inspection, and delegation.

```text
                               ┌────────────────────────┐
                               │  HUMAN OPERATOR / LEAD │
                               └───────────┬────────────┘
                                           │
                                           │ Invokes Commands / Inspection
                                           ▼
                               ┌────────────────────────┐
                               │   HOWL CLI (ENTRY)     │
                               │ Ecosystem Front Door   │
                               └─────┬────────────┬─────┘
                                     │            │
                 ┌───────────────────┘            └───────────────────┐
                 ▼                                                    ▼
┌─────────────────────────┐                                ┌─────────────────────────┐
│     HOWLFRAME (VM)      │                                │    HOWLPLANE (CONTROL)  │
│ Language, HFIR & Parser │                                │ Task Routing & Evidence │
└────────────┬────────────┘                                └──────┬────────────┬─────┘
             │                                                    │            │
             │ Compiles DSLs & Serves VM            Orchestrates  │            │ Controls Model
             ▼                                                    ▼            ▼ Execution
┌─────────────────────────┐                          ┌────────────┴┐   ┌───────┴─────────────┐
│  HOWLNOTES (KNOWLEDGE)  │                          │  HOWLBOARD  │   │  HOWLWRITER (PROSE) │
│ Field Notebook & Store  │                          │  Telemetry  │   │ Voice, Lint & Review│
└─────────────────────────┘                          └─────────────┘   └─────────────────────┘
```

### 2. Consequential Authority Flow
Consequential mutations (releases, merges, destructive mutations) require sovereign human approval through cryptographic HMAC envelopes managed by HowlChangeOps.

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CONSEQUENTIAL AUTHORITY FLOW                        │
└─────────────────────────────────────────────────────────────────────────────┘

       HowlPlane / Autonomous Workflows ──► Proposed Consequential Action
                                                      │
                                                      ▼
                                        HowlChangeOps (Release Gate)
                                                      ▲
                                                      │ [Cryptographic HMAC Signature]
                                        Human Operator (Sovereign Authority)
                                                      │
                                                      ▼
                                        Bounded Mutation / Execution
```

---

## Howl CLI (`howl`)

`howl` is the canonical command-line entry point for developers and automation across the Howl ecosystem.

### Installation & Build

```bash
# Build binary locally
go build -o bin/howl ./cmd/howl

# Run ecosystem diagnostics
./bin/howl doctor
```

### Command Reference

#### `howl version`
Report root CLI version, build timestamp, commit SHA, Go version, and platform.

```bash
howl version
howl version --json
howl version --components  # probe discovered component versions
```

#### `howl doctor`
Run non-destructive diagnostics verifying platform capabilities, toolchain dependencies, manifest syntax, component discovery, and cross-component integration contracts.

```bash
howl doctor
howl doctor -v             # show diagnostic details
howl doctor --json         # machine-readable diagnostic report
howl doctor --strict       # fail on warnings or unverified optional components
```

#### `howl status`
Fast, lightweight inspection of discovered ecosystem components, sources, and executables.

```bash
howl status
howl status --json
```

#### `howl graph`
Render the architectural hierarchy and responsibility boundaries.

```bash
howl graph
howl graph --mermaid       # export GitHub-flavored Mermaid graph
howl graph --json          # machine-readable nodes and edges
```

#### `howl project validate`
Authoritative ecosystem entry point for validating project integration manifests (`.ai-project.toml`). Directly delegates to HowlPlane's validator without duplicating logic.

```bash
howl project validate
howl project validate /path/to/project
```

#### `howl plane ...`
Direct access to HowlPlane AI engineering control plane commands. Go subcommands run in-process; extended subcommands forward to the canonical `howlplane` executable with stream and exit-code fidelity.

```bash
howl plane --help
howl plane project validate [path]
howl plane status --repo [path]
howl plane providers
```

---

## Component Discovery & Precedence

Howl uses a deterministic, bounded discovery engine with documented strict precedence:

1. **Explicit flags / options** (e.g. `--manifest`, `--config`)
2. **Environment variables** (`HOWLPLANE_HOME`, `HOWLFRAME_HOME`, `HOWL_<NAME>_DIR`, `HOWL_COMPONENTS_DIR`)
3. **User configuration** (`~/.config/howl/config.toml`)
4. **Sibling repository layout** (`../<component>`)
5. **System PATH executables** (`howlplane`, `howlframe`, etc.)
6. **Current repository** (when running from within a component directory)
7. **Manifest discovery hints** (defined in `ecosystem.toml`)

---

## Feature Roadmap

| Feature / Command | Status | Description |
| :--- | :--- | :--- |
| `howl` | **Available (v0.1.1)** | Canonical root executable |
| `howl version` | **Available (v0.1.1)** | Build and component version inspection |
| `howl doctor` | **Available (v0.1.1)** | Ecosystem health diagnostics & legacy CLI check |
| `howl status` | **Available (v0.1.1)** | Lightweight component status |
| `howl graph` | **Available (v0.1.1)** | Architectural relationship rendering |
| `howl project validate` | **Available (v0.1.1)** | Project manifest validation routing |
| `howl plane ...` | **Available (v0.1.1)** | HowlPlane control plane composition & forwarding |
| `howl install` | Planned (v0.2) | Safe prerequisite setup & component cloning |
| `howl bootstrap` | Planned (v0.2) | Workspace initialization & toolchain validation |
| `howl audit` | Planned (v0.3) | Ecosystem-wide provenance and evidence verification |

---

## Sovereign Authority Boundary

The `howl` ecosystem CLI coordinates and inspects components; it does **not** bypass or duplicate authority:
* Does not bypass HowlChangeOps cryptographic human approval gates.
* Does not bypass HowlPlane execution policies or capability grants.
* Does not mutate Git repositories during diagnostics.
* Does not directly invoke model providers without HowlPlane routing.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
