# Howl ecosystem interoperability audit

## M0 checkpoint, 2026-11-26 UTC

[Baseline inventory, compatibility observations and graphs](baseline.json) pins GitHub default revisions and distinguishes them from local checkouts. Source anchors identify an initial inspection surface. The inventory was reconciled against the Howl site, installer manifest, interop CI, owner repository metadata, and downstream README and source references. A missing release value means the latest-release lookup returned no record. A Pages flag records GitHub configuration. It does not demonstrate an accessible discovery resource.

The inventory covers the public portion of the audit. Internal components and checkout-only work are tracked in the restricted handoff. The installer manifest is an installation profile and the ecosystem site is a curated catalog. Neither is a complete repository inventory. Several local branches differ from default branch heads, including HowlFrame, HowlPlane, HowlProof, HowlSpeaker, HowlForge and HowlShield. The baseline uses archived source for the three pilot default revisions and leaves the working checkouts alone.

## What the baseline proves

The canonical Howl contracts are [interop v1](../../contracts/interop/howl.interop.v1.schema.json) and [result v1](../../contracts/interop/howl.result.v1.schema.json). [INTEROP.md](../../docs/INTEROP.md) owns the descriptor, result status, discovery precedence, version rules and handoff layer definitions. Consumers vendor exact bytes with a source revision. Howl also owns installation compatibility through its manifest and managed releases. Component-specific evidence and runtime contracts stay with their producers.

The current checker runs bounded descriptor processes and checks declared version, command, flag and schema relationships. It uses override, PATH, managed release, and local-bin discovery in that order. Strict mode promotes warnings to failures and still permits a missing component to be skipped. A release or pilot gate therefore needs an explicit required-producer list and must verify each binary was exercised. Descriptor matching does not execute the advertised operation. A scheduled green run uses moving component branches. The isolated M0 run built the nine descriptor producers from exact default revisions and checked all 29 declared CLI edges successfully. The installed-tool matrix still failed and is listed separately. Build metadata identifies the older Frame, ChangeOps and Instinct builds. Create and Relay fail before descriptor generation because their checkout environments have incompatible or missing dependencies. Release asset checksum identity remains open.

HowlFrame's command table exposes agent capabilities, describe, check, explain, snapshot, context and status. Its runtime enforces capabilities and exposes execution receipts. HowlPlane exposes a parser-derived CLI descriptor and has optional REST/MCP dependencies. The existence of those dependencies supports investigation of an existing surface. It does not establish that any endpoint is deployed or authorized. The source also includes optional Python adapters, which stay distinct from CLI compatibility evidence.

HowlProof emits evidence and handoff artifacts and consumes Plane's reconciliation interface. Its existing contract test demonstrated the real boundary with a rejected artifact, a missing tool and a repaired artifact. Plane separately exercised a real Frame toolchain. These runs establish two runtime legs. A single correlated run connecting those legs is pending.

Howl's interop guide calls the Plane/Create assessment-schema failure open. Plane's current source records a fix, and the combined-environment baseline passed at the inventory revisions. It produced two assessments and one actual Create scaffold with zero execution authority using deterministic exploration input. This is a candidate for correcting the central guide. Forge's default branch differs from the main branch used by interop CI. Its checkpoint ownership is separate from the unsupported v1 descriptor. The pinned CLI baseline reproduced a published example that Forge accepted and Relay skipped because embedded verification lacked its schema version. The Forge correction passed a real Relay handoff check with advisory authority and merged to the default branch at `c2ee71140490219424210d63ce08f793b162012f`. The merged tree matches the verified PR head. See the [Forge issue](https://github.com/howlcipher/howlforge/issues/7) and [fix](https://github.com/howlcipher/howlforge/pull/8). Provider implementations exist in both the shared library and component packages. API, policy and fallback equivalence need runtime evidence before consolidation. The existing improvement pipeline already collects observations and supports advisory review. Its adoption path must remain governed by the component owning the action.

## Architecture direction for review

| Approach | Advantages | Costs and limits |
| --- | --- | --- |
| Existing CLI/JSON plus static public resources | Reuses the resolver, process guardrails, schemas, Pages and offline operation | CLI probes have startup cost. Public metadata needs revision provenance and explicit authority and side-effect descriptions |
| In-process Python APIs for suitable optional edges | Avoids process serialization and can preserve typed objects | Dependency and release coupling. Existing Dream/Create integration behavior needs separate verification |
| REST or MCP for an explicitly authorized use case | Can make an operation accessible through a different client transport | Adds authentication, scope, revocation, supervision and remote trust requirements |
| Shared hosted runtime or new product | Could centralize execution and telemetry | Adds an operational dependency and conflicts with installer scope and component autonomy |

Recommend extending the existing CLI/JSON design. Start with a static JSON index and concise Markdown grounded in an allowlist. Publish versioned schema references and approved evidence links. Generate from pinned producers where practical and use reviewed metadata for scope and side effects. Keep correlation local and first test whether existing receipts and ledgers can carry it. Evaluate HowlFrame on deterministic validation only after proving required parsing, deterministic output, schema coverage and failure behavior against fixtures. The Go validator already covers the central runtime probe and avoids an immediate extra dependency. No contract change has been approved or adopted by this checkpoint.

## Preliminary threat model

| Boundary | Threats | Control to verify before rollout |
| --- | --- | --- |
| Public discovery to AI reader | Prompt injection, malicious generated Markdown, forged claims | Treat text as data, cite immutable evidence, check actor and evidence provenance, exercise retrieval failure |
| Build inputs to public Pages | Schema poisoning, private data or secret leakage, compromised dependencies | Explicit public field and source allowlist, pinned producer and schema provenance, reproducible generation, prohibited-content and negative-fixture checks |
| Local descriptor to consumer | Executable substitution, arbitrary command or destination, capability escalation | Caller-owned executable and operation mapping, argument arrays, schema validation, existing resolver precedence, no authority from metadata |
| Local execution and artifact intake | Path traversal, network fetch abuse, output exhaustion, orphan processes, recursion | Validate paths and trust roots, restrict destinations to policy, enforce deadlines and size budgets, cancel and clean up, bound depth and retries |
| Evidence to change authority | Forged provenance, duplicate mutation, replay, scope escalation | Validate evidence and target identity. Keep Proof's verdict advisory to release authority. Require the existing approval path and verify replay and retry behavior |
| Operational records to consumers | Log leakage and inaccurate metrics | Local storage and retention, minimal fields, explicit metric provenance, unknown usage stays unknown |

Public discovery is read-only metadata. Authorized local calls retain producer policy and existing approval rules. Any remote operation needs a separate use case and authorization design. Protected ChangeOps execution remains unavailable where the protected authority path is undeployed. Discovery must have no mechanism for unlocking it.

## Milestone ledger

| Milestone | State | Next evidence |
| --- | --- | --- |
| M0 Audit and baseline | In progress | Complete source-level coverage, including private sources in restricted evidence, reproduce unresolved dependency edges, review direction and baseline independently |
| M1 Contracts | Pending M0 | Document justified optional extensions and verify both old and new producers and consumers using canonical schemas |
| M2 Pilot | Existing legs verified | Run one pinned Frame, Plane and Proof workflow with shared correlation, blocked and unavailable outcomes, and unauthorized operation rejection |
| M3 Discovery and AI | Pending threat model review | Generate, validate and deploy the minimal Pages surface, include accessible copy and offline text, and retrieve it over public HTTP |
| M4 Reuse | Pending pilot | Fix a demonstrated repeated adapter or discovery problem and exercise its real consumers |
| M5 Observability | Pending pilot | Use existing records to produce one correlated failure, remediation and verification trail |
| M6 Adoption | Pending conformance contract | Roll out by relevance, maturity and risk, with explicit exemptions for libraries and reference apps |
| M7 Dogfood | Pending adoption | Reproduce engineering, evidence and governed handoff workflows at pinned revisions |
| M8 Maintenance | Pending verified adoption | Wire compatibility, schema provenance, publication validation and onboarding into normal development and release gates |

Independent evidence at this checkpoint includes the existing HowlProof evaluator run and historical hosted Howl CI. An independent review of the audit itself is pending. No production source, CI, schemas, release configuration or deployed assets changed in this checkpoint.
