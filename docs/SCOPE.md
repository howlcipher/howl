# Howl Product Scope

Howl is the **installer and lifecycle manager for the Howl ecosystem**.

That is its entire purpose.

Howl is **NOT**:

- an AI application
- a control plane
- an agent framework
- a workflow engine
- a model router
- a dashboard
- an observability platform
- a package manager
- a generic DevOps tool
- a replacement for any other Howl repository

Howl exists to: install the Howl ecosystem, install or validate the
dependencies required to run it, keep components at compatible versions,
update them safely, verify installation health, repair installer-owned
problems, roll back failed upgrades, and remove components cleanly.

Do not expand beyond this responsibility.

## The Constitution

> Every proposed Howl feature must directly answer:
>
> **"Is this required to install, update, verify, repair, roll back, or
> uninstall a registered Howl ecosystem component?"**
>
> If the answer is no, it does not belong in the Howl repository.

Additional rules:

1. If uncertain whether functionality belongs in Howl, choose the smaller
   design.
2. Do not build speculative abstractions.
3. Do not introduce a daemon when a CLI command can perform the task.
4. Do not create an API server merely to support the installer.
5. Do not add AI to the installer.
6. Do not duplicate functionality owned by another Howl repository.
7. Do not turn `howl status` into application monitoring.
8. Do not turn `howl doctor` into a generalized operating-system repair
   tool.
9. Do not turn dependency handling into a generic package manager.
10. Do not add features merely because they would be convenient for
    developers.

Keep Howl boring. That is intentional.

## What Howl Owns

- ecosystem installation
- component installation
- component removal
- component update
- self-update of the Howl installer
- update checking
- ecosystem version resolution
- compatibility validation
- rollback
- installer-level status
- installer diagnostics
- installer-owned repair
- platform detection
- environment detection
- dependency detection
- Howl-managed runtime installation
- artifact download
- artifact integrity verification
- installer state
- installation locking
- release channel selection
- installation plans
- installation logs
- component health contracts needed to determine whether installation
  succeeded

## What Howl Does NOT Own

- AI inference
- LLM routing
- prompts
- model selection
- model lifecycle
- agent execution
- agent orchestration
- workflow execution
- user workflows
- application dashboards
- application observability
- document generation
- writing functionality
- user content
- generalized DevOps automation
- arbitrary system administration
- arbitrary Git repository management
- generic package management
- secrets platforms
- CI/CD orchestration
- cloud provisioning
- Kubernetes administration
- fleet management
- enterprise policy engines
- plugin marketplaces
- remote machine management
- application business logic
- application configuration UIs

## Dependency Direction

```text
Howl Installer
      |
      v installs
      |
HowlPlane
HowlFrame
HowlChangeOps
HowlWriter
```

Howl must never depend on a component in order to install that component.
The `howl` module must not import another Howl repository's source as a Go
package, and must not require a component to already be present in order to
bootstrap that same component.

## Scope Ideas Explicitly Rejected For v1

These were considered during the v1 installer milestone and deliberately
left out — recorded here so they are not silently re-proposed:

- **Automatic Ollama bootstrap.** Detection and reporting only; installing
  or configuring Ollama without explicit per-run approval is deferred.
- **Remote/downloadable release manifests.** v1 ships an embedded manifest
  only; fetching manifests over the network is deferred until the trust and
  validation model is designed.
- **A Howl-managed Go toolchain.** Three of the four ecosystem components
  currently lack prebuilt release artifacts and must be built from source,
  which requires a system Go toolchain in v1. An isolated, Howl-managed Go
  toolchain (so a system Go install isn't required) would resolve this but
  is a meaningfully larger feature than this milestone's budget; tracked as
  remaining work.
- **`howl graph` / architecture visualization.** Explicitly listed as a
  non-goal; graphical or textual dependency visualization is not installer
  lifecycle work.
- **Direct composition of another component's CLI** (the old `howl plane`
  and `howl project validate` commands, which imported and forwarded into
  HowlPlane's command tree). This duplicated functionality HowlPlane already
  owns and created a circular source dependency; removed in this milestone.
