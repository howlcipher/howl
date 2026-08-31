# Howl Ecosystem

> **Website & Documentation:** https://howlcipher.github.io/howl/

**Howl** is a unified open-source ecosystem designed to bring mathematical rigor, capability boundaries, adversarial verification, and cryptographic human authority to AI-assisted software engineering.

## Axiom

> **Intent is not authority.**
> Probabilistic intelligence proposes; deterministic machinery controls execution; humans retain sovereign authority over consequential risk.

---

## Ecosystem Repositories

| Repository | Role & Description | Documentation Site | Status |
| :--- | :--- | :--- | :--- |
| **[`howlframe`](https://github.com/howlcipher/howlframe)** | **Intelligence & Reasoning Framework:** AI DSL compiler, typed HFIR verification gate, and capability-bounded VM. | [howlcipher.github.io/howlframe](https://howlcipher.github.io/howlframe/) | Active |
| **[`howlplane`](https://github.com/howlcipher/howlplane)** | **AI Engineering Control Plane:** Deterministic multi-agent task routing, adversarial falsification, and evidence ledgers. | [howlcipher.github.io/howlplane](https://howlcipher.github.io/howlplane/) | Active |
| **[`howlnotes`](https://github.com/howlcipher/howlnotes)** | **Knowledge Notebook & Dogfood Consumer:** Full-stack notes application proving browser compilation and native store persistence. | [howlcipher.github.io/howlnotes](https://howlcipher.github.io/howlnotes/) | Active |
| **[`howlchangeops`](https://github.com/howlcipher/howlchangeops)** | **Authority Boundary & Release Controller:** Enforces HMAC cryptographic human approvals and bounded Git mutations. | [howlcipher.github.io/howlchangeops](https://howlcipher.github.io/howlchangeops/) | Active |
| **[`howlboard`](https://github.com/howlcipher/howlboard)** | **Evaluation Surface & Telemetry Console:** Deterministic task state machine proving compiler maturity through dogfooding. | [howlcipher.github.io/howlboard](https://howlcipher.github.io/howlboard/) | Active |
| **[`howlwriter`](https://github.com/howlcipher/howlwriter)** | **Writing Control & Review System:** Voice preservation, deterministic style linting, humanization, and adversarial review. | [howlcipher.github.io/howlwriter](https://howlcipher.github.io/howlwriter/) | Active |

---

## Architecture

```text
                              ┌────────────────────────┐
                              │  HUMAN OPERATOR / LEAD │
                              └───────────┬────────────┘
                                          │
                                   Cryptographic HMAC
                                   Approval Signature
                                          │
                                          ▼
┌─────────────────────────┐   ┌────────────────────────┐   ┌─────────────────────────┐
│     HOWLFRAME (VM)      │◄──┤  HOWLCHANGEOPS (GATE)  ├──►│    HOWLPLANE (CONTROL)  │
│ Language, HFIR & Parser │   │ Bounded Release Action │   │ Task Routing & Evidence │
└────────────┬────────────┘   └────────────────────────┘   └──────┬────────────┬─────┘
             │                                                    │            │
             │ Compiles DSLs & Serves VM            Orchestrates  │            │ Controls Model
             ▼                                                    ▼            ▼ Execution
┌─────────────────────────┐                          ┌────────────┴┐   ┌───────┴─────────────┐
│  HOWLNOTES (KNOWLEDGE)  │                          │  HOWLBOARD  │   │  HOWLWRITER (PROSE) │
│ Field Notebook & Store  │                          │  Telemetry  │   │ Voice, Lint & Review│
└─────────────────────────┘                          └─────────────┘   └─────────────────────┘
```

---

## Getting Started

```bash
# Clone the complete ecosystem
git clone https://github.com/howlcipher/howl.git
git clone https://github.com/howlcipher/howlframe.git
git clone https://github.com/howlcipher/howlplane.git
git clone https://github.com/howlcipher/howlnotes.git
git clone https://github.com/howlcipher/howlchangeops.git
git clone https://github.com/howlcipher/howlboard.git
git clone https://github.com/howlcipher/howlwriter.git
```

## License

MIT License. See [LICENSE](LICENSE) for details.
