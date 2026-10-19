# Changelog

## 2026-10-19

HowlWriter's runtime now receives HowlPlane. HowlWriter runs HowlPlane's role
dispatcher in its own process, but Howl installed only the `howlplane` binary,
so the HowlWriter launcher found no HowlPlane (or, with `HOWLPLANE_ROOT` set,
lacked jsonschema and HowlPlane's other dependencies) and every model step
failed while `howl doctor` reported HEALTHY. A component can now declare
`python_imports`; each named component's package is installed into its runtime
the way that component installs itself (a verified release wheel, or an
editable checkout under the developer profile), its dependency declarations are
tracked for drift, the health check imports it, and rollback restores it at its
own version. The default manifest declares `python_imports =
["howlplane-engine"]` for HowlWriter.

A developer runtime's repair now reinstalls from the checkout its dependency
marker records. With a second editable checkout in the runtime, the previous
lookup took the first in sorted order, which would have reinstalled HowlWriter
from HowlPlane's checkout.

## 2026-10-12

Scoped installation now includes transitive component dependencies in manifest
dependency order. Installing HowlChangeOps on a fresh machine includes HowlFrame;
shared dependencies appear once, profile selection is preserved, and components
already at their target versions remain unchanged.

Fix UX-005: developer source installation copies declared runtime sidecar files
after build steps and before activation. The default ChangeOps manifest declares
its compiled policy; missing or unsafe artifacts now fail installation.

## 2026-10-08

The first-run example quotes its verification command: `--verify python3 -m
unittest` failed with `unrecognized arguments: -m unittest` because HowlPlane
read `-m` as an orchestrate option (DOG-035, fixed in HowlPlane). Documentation
only; installer behavior is unchanged.

## 2026-09-11

Added HowlProof to the ecosystem as a first-class component and recut the hub's
lifecycle to include verification. The topology diagram previously ran creation
through governance with no challenge step and omitted HowlDream entirely; it now
shows imagine, orchestrate, build, challenge, govern, learn, with remediation
routed back through the control plane rather than performed by the evaluator.
The interoperability grid is ordered by that lifecycle instead of by tier.

The drawer footer read "9 CORE TOOLS" while listing ten nodes; it now states the
count it shows. "STATUS: 100% GREEN" was removed: HowlProof's first evaluation of
HowlBoard returned CONDITIONALLY_PROVEN with open findings, so the hub should not
assert that everything is green. The ecosystem page's meta description, component
directory, responsibility arrows, companion list and audit date were updated, and
the HowlProof to HowlChangeOps boundary is stated explicitly because the two are
adjacent and easily confused.

Installer behavior is unchanged. HowlProof is not in the default installer
manifest and holds no authority to release anything.

Corrected the quick start in the same pass. It advertised `howl graph`,
`howl project validate` and `howl plane --help`, none of which the CLI registers;
`docs/SCOPE.md` records that they were removed. The block now shows doctor,
status, install, update, rollback, channel and uninstall. The HowlProof cards
state that it is experimental and at milestone one, so the directory does not
imply a maturity the repository does not claim.

## 2026-09-09

Added a first-class Ecosystem page with accurate runtime and authority boundaries,
component maturity, newcomer paths, and HowlDream discovery. Updated current
architecture documentation and site navigation. HowlDream remains experimental
and outside the default installer manifest; installer behavior is unchanged.

Live link verification found HowlBot Pages unavailable (HTTP 404). Its ecosystem
card now links to the repository and explicitly records site unavailability.
