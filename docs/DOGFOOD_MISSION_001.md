# Howl Ecosystem Dogfood Mission 001

**Campaign:** `2026-09-27-continuous-improvement`

## Mission

Evaluate and improve Howl as a coherent autonomous engineering ecosystem capable of supporting a long-running, multi-agent software factory.

Primary question:

> Can the Howl ecosystem safely and coherently PLAN → DECOMPOSE → ASSIGN → EXECUTE → GOVERN → VERIFY → REVIEW → HANDOFF → RESUME → REPAIR → RECONCILE → CONTINUE without requiring excessive manual glue?

This is an execution specification for a future dogfood run. Begin the mission only when explicitly launched with the command in [Launch](#launch). Read this entire document before planning or changing anything.

## Operating principles

### Discover the current ecosystem

Do not assume a static component list or trust documentation without checking implementation. Inventory the repositories and installed capabilities available at run time. Likely projects include HowlPlane, HowlFrame, HowlChangeOps, HowlProof, HowlBoard, HowlWriter, HowlRelay, HowlCreate, HowlInstinct, installers, provider integrations, and shared tooling.

For every discovered component, establish from source, metadata, executable behavior, and current documentation:

- component responsibilities and maturity
- public interfaces and CLI conventions
- lifecycle terminology and status/error semantics
- run records, schemas, and artifact formats
- provider and worker abstractions
- integration points and duplicated responsibilities
- installation, configuration, upgrade, and compatibility expectations

Determine whether canonical ecosystem concepts exist for campaign, run, orchestration, task, execution, parent/child relationship, artifacts, provider, worker, lifecycle states, failures, retries, resumability, evidence, approval, timestamps, and schema versions. Locate incompatible or duplicated definitions across repositories.

### Ownership and concurrent execution

**INSPECT BROADLY. MODIFY NARROWLY.**

This mission may inspect all related repositories, but it owns only:

- cross-component architecture and interoperability
- shared conventions, contracts, schemas, and identifiers
- installers and ecosystem-level integration
- cross-component observability
- integration documentation

The concurrent HowlPlane mission owns HowlPlane implementation, orchestration lifecycle, state, scheduling, task graphs, workers, providers, handoffs, resumability, reconciliation, and orchestration observability.

The concurrent HowlFrame mission owns HowlFrame implementation, plans, policy evaluation, authorization, hashing/integrity, TTL, apply, verify, rollback, replay protection, and execution evidence.

Do not independently redesign or edit HowlPlane- or HowlFrame-owned implementation. Record component-specific problems as structured findings with `recommended_owner` set to `howlplane` or `howlframe`. Likewise, do not edit any other mission's temporary workspace or artifacts. Prefer ecosystem-owned contract or integration changes only when evidence proves they belong here.

All three missions may run simultaneously. Therefore:

- do not assume a globally exclusive process
- use campaign- and run-scoped temporary directories and artifact names
- preserve run identity and avoid shared unscoped files
- avoid log, lockfile, provider-state, and generated-artifact collisions
- do not reuse another mission's workspace
- surface locking and global-state defects as findings rather than bypassing them

### Evidence before changes

For every candidate defect:

1. establish current behavior,
2. reproduce or demonstrate the problem,
3. preserve evidence,
4. determine expected behavior from contracts, safety requirements, or tests,
5. make the smallest justified in-scope change,
6. add regression coverage,
7. rerun the exposing scenario, and
8. record before/after results.

Do not perform speculative architecture rewrites merely because an agent prefers another design. Clearly separate observed defects from speculative future ideas.

### Real dogfooding

This is not a static code review. Use existing Howl capabilities on Howl itself wherever reasonably practical. Exercise actual cross-component workflows and artifact handoffs. When an intended Howl path fails or requires manual intervention, preserve the failure and the manual bridge as dogfood evidence before using an escape hatch. Do not manufacture meaningless failures to increase finding count.

For each manual bridge ask: **Is this an unavoidable safety boundary, or missing ecosystem integration?**

### Severity

Use exactly this shared vocabulary:

- **P0** — safety/integrity failure that prevents trustworthy autonomous operation
- **P1** — major capability is broken or cannot reliably complete
- **P2** — substantial reliability, architecture, integration, recovery, concurrency, or correctness weakness
- **P3** — meaningful UX, diagnostics, observability, or maintainability issue
- **P4** — optional improvement

**Do not inflate severity.**

### Findings and correlation

Create machine-readable findings as well as Markdown reports. Ecosystem finding IDs use `HE-*` (for example, `HE-001`). At minimum each finding must conform to this shape; extensions are allowed when useful:

```json
{
  "id": "HE-001",
  "campaign_id": "2026-09-27-continuous-improvement",
  "component": "ecosystem",
  "severity": "P2",
  "category": "interoperability",
  "summary": "...",
  "evidence": [],
  "reproduction": [],
  "expected_behavior": "...",
  "recommended_owner": "ecosystem",
  "dependencies": [],
  "status": "open"
}
```

Use stable paths or identifiers in evidence entries. Record disposition and remediation evidence without deleting the original observation.

Require generated evidence, where feasible, to preserve:

- `campaign_id`
- `run_id`
- `orchestration_id`
- `task_id`
- `execution_id`
- `parent_id`

Do not invent identifiers the implementation cannot reasonably produce. Record missing correlation capability as a finding.

### Git discipline

Before changing anything, inspect each affected repository's contribution practices, current branch, worktree state, and expected verification. Do not overwrite unrelated changes. Do not silently commit to a protected/default branch when branches or pull requests are expected. Do not merge work merely to make the report appear successful. Keep changes attributable to this campaign and respect human authority boundaries.

## Investigation and dogfood program

### Architecture and contracts

Produce an evidence-backed current architecture. Map responsibility, interfaces, lifecycle vocabulary, schemas, artifact ownership, providers, workers, state persistence, errors, retries, approvals, and evidence flows. Identify responsibility gaps and overlaps. Compare canonical concepts and schema versions across repositories, including timestamp and identifier semantics.

Evaluate:

- interoperability and component boundaries
- contracts and shared schemas
- provider-neutral behavior
- correlation and cross-component auditability
- resumability across component boundaries
- artifact passing and ownership
- error propagation and status translation
- installation, update, rollback, configuration, and compatibility
- observability, governance, and security boundaries
- concurrency and recovery

### Workflow exercises

Select meaningful, harmless workflows that traverse multiple real components. Record the exact commands, inputs, versions/commits, environment, identifiers, artifacts, expected behavior, observed behavior, exit status, and manual intervention. Prefer deterministic fixtures and reversible operations. Do not perform destructive tests merely to demonstrate a theoretical risk.

Exercise, where practical:

- planning or orchestration into governed execution and verification
- provider or worker handoff while preserving task intent
- artifact production, transfer, validation, and attribution
- interruption followed by cross-component resume
- propagated failure and recovery
- installer or configuration compatibility checks
- evidence aggregation sufficient for an independent audit

### Safe concurrency

Perform safe concurrent workflow tests where justified. Look for:

- global mutable state
- temporary-directory, generated-file, log, and lock collisions
- provider state leakage
- artifact ownership ambiguity or crossover
- approvals scoped too broadly
- identifiers absent from logs and evidence
- assumptions that only one Howl operation runs at a time

Do not let concurrency tests interfere with the Plane or Frame mission workspaces.

### Remediation and audit

Only remediate validated ecosystem-owned defects. Every fix requires a regression test or equivalent deterministic contract coverage and rerun of the scenario that exposed it. Preserve negative and success-path evidence. After remediation, perform an independent audit that attempts to falsify the claimed result and reconcile all findings without silently dismissing them.

## Required outputs

Place version-controlled reports under `docs/` or clearly run-scoped artifacts under the repository's established run-artifact location:

- `HOWL_ECOSYSTEM_DOGFOOD_REPORT.md`
- `HOWL_ECOSYSTEM_FINDINGS.json`
- `HOWL_ECOSYSTEM_ARCHITECTURE.md`
- `HOWL_ECOSYSTEM_CONTRACTS.md` when useful

The final report must contain:

1. Executive Summary
2. Current Ecosystem Architecture
3. Component Responsibilities
4. Responsibility Overlap
5. Cross-Component Contracts
6. Dogfood Workflows Performed
7. What Worked
8. What Failed
9. Manual Intervention Required
10. Concurrency Findings
11. Resumability Findings
12. Provider/Handoff Findings
13. Observability Findings
14. Governance/Security Findings
15. Changes Implemented
16. Regression Coverage
17. Remaining Findings
18. Recommended Remediation Sequence
19. Longer-Term Architectural Ideas

Clearly separate observed defects from speculative future ideas. Success means improved coherence, recoverability, interoperability, and ability to operate continuously, not maximum code churn.

## Completion criteria

The mission is complete only when:

- current ecosystem components and responsibilities are evidenced rather than assumed
- meaningful cross-component workflows have been exercised where practical
- failures and manual interventions have been preserved
- all findings are represented in valid machine-readable output
- ownership boundaries and recommended owners are correct
- in-scope fixes are minimal, tested, rerun, and independently audited
- before/after evidence is correlated and reproducible
- remaining risks and remediation order are explicit
- no out-of-scope component implementation was modified

## Successor Mission Contract

After this mission has completed:

1. implementation,
2. deterministic verification,
3. independent audit,
4. remediation of valid in-scope findings,
5. verification after remediation,
6. final acceptance,
7. findings reconciliation,

the agent must decide whether meaningful follow-up work remains.

Do not create another mission merely because iteration is possible, for cosmetic churn, or to keep agents busy. Create `docs/DOGFOOD_MISSION_002.md` only when one or more of the following is true:

- an unresolved P0 finding exists
- an unresolved P1 finding exists
- meaningful P2 work remains
- multiple P3 findings share a root cause worth addressing
- completed remediation exposes another necessary maturity step
- deterministic evidence demonstrates a missing capability
- a blocked capability becomes actionable
- architecture needs a bounded follow-up before the system can progress safely

If no worthwhile successor exists, record in the final completion report:

```
NEXT_MISSION: NOT_REQUIRED
```

If meaningful work remains, create `docs/DOGFOOD_MISSION_002.md`. The successor must be derived from Mission 001 evidence, must not merely duplicate Mission 001, and must explicitly identify:

- predecessor mission (`docs/DOGFOOD_MISSION_001.md`)
- campaign ID (`2026-09-27-continuous-improvement`)
- findings that justify it, citing finding identifiers as evidence whenever possible
- work completed by the predecessor
- work that must not be repeated
- unresolved findings
- newly exposed findings
- dependencies
- repository ownership
- objective
- scope
- explicit non-goals
- deterministic verification
- independent audit requirements
- success criteria
- stop conditions

Mission numbering applies recursively to future missions. A Mission N may generate Mission N+1 when justified (for example, `DOGFOOD_MISSION_002.md` may generate `DOGFOOD_MISSION_003.md`). Each successor must reference its predecessor. Never overwrite previous mission documents; they are historical execution contracts.

Creating a successor mission does not authorize executing it. Do not invoke `howl orchestrate` recursively, `howl factory` recursively, or launch Mission N+1 from Mission N. Return control to the outer Factory supervisor after Mission N completes.

### Factory handoff

The final completion report for this mission must end with a machine-readable final section containing:

```
MISSION_STATUS: <COMPLETE|BLOCKED|IDLE>
NEXT_MISSION: <docs/DOGFOOD_MISSION_002.md or NOT_REQUIRED>
OPEN_P0: <count>
OPEN_P1: <count>
OPEN_P2: <count>
OPEN_P3: <count>
BLOCKED: <true|false>
RECOMMENDED_PRIORITY: <advisory only>
FACTORY_NOTES: <any portfolio context>
```

`RECOMMENDED_PRIORITY` is advisory only; the master Factory makes the final portfolio decision. A project is allowed to report that its own next mission should not run yet.

## Launch

Run from this repository only when ready to begin the mission:

```bash
howl orchestrate \
  "Execute Dogfood Mission 001 exactly as specified in docs/DOGFOOD_MISSION_001.md. Read the entire mission before planning. This repository owns ecosystem-level architecture and integration. Inspect related Howl repositories when useful, but respect the ownership boundaries in the mission. Complete dogfooding, justified remediation, deterministic verification, independent audit, and final reporting." \
  --repo "$PWD" \
  --heartbeat 15
```
