# provider-matomo

Crossplane provider for [Matomo](https://matomo.org) analytics. Manages Matomo
resources — tracked **Sites**, saved **Segments**, and app-specific auth
**Tokens** — through the Matomo Reporting API.

## Development Workflow

**Spec + implementation together by default. Spec and code MUST stay in sync.**

1. **Default — one PR:** spec and implementation ship in the same PR.
2. **Small changes — no spec:** a small, self-contained change (one field, one
   bug fix) needs no spec. Just open the PR.
3. **Spec-first (opt-in):** for large or risky work, open a spec-only PR first,
   then stack the implementation PR on it.
4. Any code change that affects documented behavior requires a simultaneous
   spec update in the same commit.
5. PRs that diverge from the spec are blocked until the spec is updated.

## Conventions

- **API group:** `matomo.m.crossplane.io`, version `v1alpha1`. All kinds
  (ProviderConfig, Site, Segment, Token) live in `apis/v1alpha1`.
- **Native provider:** hand-written managed resources + controllers calling the
  Matomo Reporting API over HTTP. No Upjet/Terraform.
- **Generated code:** `make generate` runs controller-gen (deepcopy + CRDs) and
  angryjet (managed-resource method sets). Commit the results; CI `check-diff`
  fails if they drift.
- **Image:** `ghcr.io/maximilianbraun/provider-matomo`.

## Build

```sh
make generate   # deepcopy, CRDs, managed-resource methods
make build      # binary → bin/provider-matomo
make test       # unit tests
```
