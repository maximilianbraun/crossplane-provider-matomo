# provider-matomo

A [Crossplane](https://crossplane.io) provider for [Matomo](https://matomo.org)
analytics. It reconciles Matomo resources through the Matomo Reporting API:

| Kind      | Matomo API      | Operations        |
|-----------|-----------------|-------------------|
| `Site`    | SitesManager    | create/update/delete + drift |
| `Segment` | SegmentEditor   | create/update/delete + drift |
| `Token`   | UsersManager    | create/delete (token written to a connection secret) |

API group: `matomo.m.crossplane.io/v1alpha1`.

## Quick start

Install the provider (published on tag via the `release` workflow):

```sh
crossplane xpkg install provider ghcr.io/maximilianbraun/provider-matomo:<tag>
```

Then configure it and create resources:

```sh
kubectl apply -f examples/providerconfig.yaml
kubectl apply -f examples/site.yaml
```

See [`examples/`](examples/) for `ProviderConfig`, `Site`, `Segment`, and
`Token` manifests, and [`CLAUDE.md`](CLAUDE.md) for the development workflow.

## Build

Uses the standard Crossplane build system (the `build/` git submodule):

```sh
git clone --recurse-submodules https://github.com/maximilianbraun/crossplane-provider-matomo
# or, if already cloned: make submodules

make generate     # CRDs + deepcopy + managed-resource methods
make build        # binaries, provider image, and xpkg (needs docker buildx)
make reviewable   # lint + generate + tidy, before opening a PR
```

The provider is packaged as a single self-contained xpkg with the controller
runtime embedded; `make publish VERSION=vX.Y.Z` pushes it to
`ghcr.io/maximilianbraun/provider-matomo`.

## Testing

Unit tests (no Matomo needed):

```sh
make test
```

End-to-end against a real, dockerized Matomo (client-level — drives the
controllers' external clients against a live instance, no cluster):

```sh
make test-e2e    # compose up + bootstrap + e2e tests
make e2e-down    # clean up
```

The `e2e` GitHub Actions workflow runs the same flow on every push/PR to `main`.
e2e tests are gated behind the `e2e` build tag, so they never run in the normal
unit suite.

## Status

Site, Segment, and Token controllers implemented (Observe/Create/Update/Delete)
with unit tests. Drift is detected for Site and Segment on every reconcile. An
e2e test against a live Matomo is still outstanding.

### Token caveat

Matomo stores auth tokens hashed and returns the value only once, at creation.
Consequences:

- The minted token is written to the resource's connection secret on create and
  cannot be read back.
- A `Token` is always reported up-to-date once created; changing a field mints a
  **new** token rather than reconciling in place.
- **Delete is a no-op.** Matomo has no API to delete a single token by value or
  ID (only `deleteAllTokensForUser`, which revokes every token for the user).
  Deleting a `Token` resource leaves the token valid until its expiry; rotate
  out of band if you need immediate revocation.
