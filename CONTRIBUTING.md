# Contributing to Howl

Howl is the installer and lifecycle manager for the Howl ecosystem — nothing
more. Before proposing or implementing a feature, read
[`docs/SCOPE.md`](docs/SCOPE.md) in full.

Every proposed feature must directly answer:

> "Is this required to install, update, verify, repair, roll back, or
> uninstall a registered Howl ecosystem component?"

If the answer is no, it does not belong in this repository — even if it
would be convenient, even if another AI application in the ecosystem could
use it, and even if it's technically interesting to build. Open an issue in
the repository that actually owns the concern instead.

When in doubt, choose the smaller design. Do not add a daemon, an API
server, a plugin system, or AI-assisted behavior to the installer. Keep
Howl boring — that's deliberate, not an oversight.

## Development

```bash
go build -o bin/howl ./cmd/howl
go test ./...
gofmt -l .
go vet ./...
```

See `docs/ARCHITECTURE.md` for the internal package layout and
`docs/MANIFEST.md` for the release manifest schema.

## Pull Requests

- Keep commits small and logically scoped.
- Add or update tests for any behavior change, especially anything that
  touches installation, update, rollback, or uninstall — these are the
  parts of Howl that can damage a user's machine if they regress.
- Do not mix scope-boundary cleanup with unrelated feature work in the same
  change.
