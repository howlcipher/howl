# Howl Interop Contract

Howl ecosystem components talk to each other through **command-line
processes and JSON files** for their public interoperability boundary: one component runs another's executable with an
argument list (never a shell string), reads its JSON output and exit status,
and hands larger artifacts over as JSON files on disk. There is no shared
daemon, HTTP API, or message bus between components, and this contract does
not add one.

This page is the single definition of that contract. It exists so a
consumer can depend on a producer's command line without scraping prose,
and so `howl doctor` can tell when two installed components no longer fit
together. The machine-readable schemas live in
[`contracts/interop/`](../contracts/interop/).

## Why command lines and files

| Option | For | Against |
| :--- | :--- | :--- |
| **CLI + JSON (chosen)** | Works offline and without a daemon; every component already has a CLI; cooperative confirmation prompts keep working (they do not authenticate owner identity against same-user processes); a process boundary isolates crashes and dependencies. | Process start-up cost; the contract must be written down, which this page does. |
| MCP servers | Agents could call components as tools. | A second surface to secure; owner-confirmation rules would have to be re-implemented per server. Can be generated later from the descriptors below. |
| Local HTTP services | Rich request/response. | Needs daemons, ports, authentication and lifecycle management; SCOPE.md forbids a server in the installer. |
| In-process imports | No serialization. | Couples dependency trees and release cadence. Verified cases include HowlWriter → HowlPlane (manifest `python_imports`) and optional HowlPlane → HowlDream (`dream-api` extra, pinned API revision; see HowlPlane `documentation/HOWLDREAM_API.md`). Plane also optionally imports Create for advisory scaffold promotion; its combined-environment assessment-schema failure remains open. CLI descriptors describe process calls, not these Python APIs. |

## 1. The interop descriptor

Every component that another component calls publishes a descriptor:

```bash
<component> interop --json
```

It prints one JSON object with `"schema": "howl.interop/v1"`
([schema](../contracts/interop/howl.interop.v1.schema.json)) and exits 0.
The command takes no other arguments, needs no configuration, touches no
network, and never prompts.

```json
{
  "schema": "howl.interop/v1",
  "component": "howlchangeops",
  "version": "0.2.0",
  "commands": [
    {
      "path": ["evaluate"],
      "flags": ["--json"],
      "json": true,
      "emits": ["howl.result/v1"],
      "accepts": ["howlchangeops.proposal/v1"]
    }
  ],
  "exit_codes": {"0": "ok (inspect status for the verdict)", "1": "error"},
  "env": ["HOWLCHANGEOPS_BASE", "HOWLFRAME_BIN"],
  "consumes": [
    {
      "component": "howlframe",
      "command": ["run"],
      "flags": ["--allow-caps"],
      "min_version": "0.1.0"
    }
  ]
}
```

- **`commands[].path`** is the subcommand path; `[]` describes the global
  flags that come before any subcommand.
- **`commands[].flags`** lists every flag the command accepts, spelled
  with the dashes users type. The checker compares flag names without
  their leading dashes, so `-allow-caps` and `--allow-caps` match (Go's
  `flag` package accepts both).
- **`commands[].json`** is true when the command can print machine-readable
  JSON on stdout (by default or with `--json`).
- **`emits` / `accepts`** name the JSON schemas the command writes or
  reads, as `<namespace>.<kind>/v<N>`.
- **`exit_codes`** is the component's own table. Codes are **not**
  renumbered across the ecosystem (see section 3).
- **`consumes`** lists every command of another component this one runs,
  with the flags it passes and the oldest producer version it supports.
  This is what makes an edge checkable.

Generate the descriptor from the real parser where the framework allows it
(walk argparse subparsers, click groups, or the cobra command tree) so it
cannot drift from the CLI, and pin it with a test. Hand-write it only for
CLIs built on Go's standard `flag` package, and test it against the
dispatch table.

## 2. Result envelope

New machine-readable output uses `"schema": "howl.result/v1"`
([schema](../contracts/interop/howl.result.v1.schema.json)):

```json
{
  "schema": "howl.result/v1",
  "component": "howlchangeops",
  "command": ["evaluate"],
  "status": "awaiting_human",
  "data": {"decision_id": "dec-...", "verdict": "REQUIRE_APPROVAL"},
  "error": null
}
```

`status` is the field a consumer branches on, never prose and never an exit
code alone:

| `status` | Meaning |
| :--- | :--- |
| `ok` | Completed; `data` holds the result. |
| `partial` | Completed with part of the work missing; `data` says which. |
| `needs_input` | The caller must supply something (a choice, a file, a flag). |
| `awaiting_human` | Waiting on an owner decision; not an error. |
| `blocked` | A precondition or policy gate stopped the work. |
| `denied` | A policy evaluated the request and refused it. |
| `unavailable` | A dependency (binary, model, package) is missing or unreachable. |
| `failed` | An error; `error.code` and `error.message` explain it. |

Existing JSON artifacts with their own schemas (`howl.exploration/v1`,
`howlinstinct.decide_output/v1`, `howlwriter.copy_package/v1`, and so on)
are unchanged. A command's descriptor names what it emits.

## 3. Exit codes

Each component keeps its documented exit codes; the same number can mean
different things in different components (2 is "cancelled" in `howl`,
"awaiting human" in `howlplane`, and "no model chosen" in `howlwriter`).
Renumbering would break every existing caller, so instead:

1. every component declares its table in `exit_codes`;
2. a consumer reads `status` from JSON output where there is any, and
   otherwise interprets an exit code only through the producer's table;
3. a usage error (unknown flag or subcommand) must not share a code that
   the component documents for a normal outcome.

`howl orchestrate`, `howl agents doctor`, and `howl factory prepare` pass
HowlPlane's exit status through unchanged, so their codes follow
HowlPlane's table, not `howl`'s.

## 4. Schema identifiers

New outputs carry `"schema": "<namespace>.<kind>/v<N>"`. Artifacts that
already use `schema_version` with the same string form (the `howl.*/v1`
envelopes, HowlWriter's native contracts) keep that key; a consumer
accepts the key its producer documents. A breaking change to a document's
shape bumps `vN`; adding an optional field does not.

## 5. Finding another component

A consumer locates a producer's executable in this order and stops at the
first hit:

1. `HOWL<NAME>_BIN` (for example `HOWLCHANGEOPS_BIN`, `HOWLFRAME_BIN`); an
   explicit override that is not an executable is an error, not a reason to
   look elsewhere;
2. `PATH`, which is the user's own choice of executable;
3. the Howl-managed release, for callers whose `PATH` lacks it (services,
   cron): `$XDG_DATA_HOME/howl/components/<name>/current` is a symlink to the
   active `releases/<version>` directory (on platforms without symlinks, a file
   holding `<version>`), and the executable inside it is named `<name>`;
4. `~/.local/bin/<name>`.

Producers must not depend on the caller's working directory to find their
own files (bytecode, policies, configuration): resolve them next to the
executable or under an explicit environment variable or XDG directory.

## 6. Vendored schemas

A consumer that validates a producer's JSON vendors the schema file and a
`SOURCE.md` naming the producer repository and a commit **on its `main`
branch**. CI compares the vendored copy with that commit so drift is caught
when it happens, not when an edge breaks.

## 7. Handoff documents

Three handoff formats exist, one per layer. Use the one for your layer; do
not add a fourth.

| Format | Layer | Producer | Reader |
| :--- | :--- | :--- | :--- |
| `howlforge.checkpoint/v1` (a bundle directory with `checkpoint.json`) | One task passing between AI workers: what is done, what remains, the last verified commit, whether it is safe to resume | HowlForge (`howlforge handoff validate <dir>`) | HowlRelay (`worker_checkpoint` evidence and next action) |
| `howlfutureworks.handoff/v1` | Work passing between organizational positions | HowlFutureWorks | people and positions |
| `ai.handoff/v1` | Superseded by `howlforge.checkpoint/v1`; kept so existing HowlPlane examples validate | none | none |

Field correspondence:

| Meaning | `howlforge.checkpoint/v1` | `howlfutureworks.handoff/v1` | `ai.handoff/v1` |
| :--- | :--- | :--- | :--- |
| Work identifier | `task_id` | `work_item_id` | `run_id` |
| Who hands off | `worker.role`, `runtime_identity` | `from`, `from_employee_id` | none |
| Who receives | the next matched worker | `to`, `to_employee_id` | none |
| Done | `completed` | `summary` | `work_completed` |
| Remaining | `remaining` | `next_action` | `remaining_work`, `recommended_next` |
| Changed files | `changed_files` | `authoritative_pointers` | `files_modified` |
| Commit | `last_verified_commit` | none | `starting_commit` |
| Why it stopped | `handoff_reason` | `blockers` | `failure_or_interruption_reason` |

## 8. Checking the edges

`howl doctor` runs `interop --json` for each installed component and
reports, under **Interop**:

- a component that publishes no descriptor (WARN; older releases);
- an invalid descriptor (FAIL);
- a `consumes` entry whose producer lacks the command or a flag, emits no
  schema the consumer expects, or is older than `min_version` (FAIL).

For a developer workspace, the same resolution runs over every component
that can be found with the discovery order above:

```bash
go run ./tools/interop-check            # human-readable
go run ./tools/interop-check --json     # machine-readable
go run ./tools/interop-check --bin howlplane=/path/to/howlplane
```

It exits 1 when any edge fails. The `Ecosystem interop` workflow
(`.github/workflows/interop.yml`) runs it nightly against these ten components' explicit `main` branches (Forge's current default branch differs): HowlFrame,
HowlChangeOps, HowlInstinct, HowlPlane, HowlWriter, HowlDream, HowlCreate,
HowlProof, HowlForge and HowlRelay. It fails on a missing descriptor as well as a broken edge.

The workflow also runs `scripts/test_interop_runtime.py` against the real built
Howl CLI and installed HowlPlane. It verifies public CLI forwarding, structured
empty-session inspection, missing-session failure propagation, and repeated
read-only inspection without repository mutation. This bounded smoke test does
not establish live model execution, protected approval, checkpoint recovery, or
complete workflow readiness. Other ecosystem repositories are outside this CI
job's declared scope.

Version checks support three-part SemVer (including numeric and lexical
prerelease identifiers) and three-part PEP 440 a/b/rc, post and dev releases.
Epochs and shortened tuples are rejected. Ambiguous final `+metadata` versions
follow SemVer build semantics; local ordering applies to explicitly marked PEP
versions. Mixed nonfinal dialects with no defined common ordering fail closed.
Descriptor discovery limits stdout to 1 MiB, stderr to 64 KiB, execution to 15
seconds by default and pipe cleanup to 100 ms. POSIX process groups and Windows
kill-on-close jobs terminate ordinary descendants. This is lifecycle management,
not an operating-system sandbox against deliberately escaping processes.
