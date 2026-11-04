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

```sh
kubectl apply -f examples/providerconfig.yaml
kubectl apply -f examples/site.yaml
```

See [`examples/`](examples/) for `ProviderConfig`, `Site`, `Segment`, and
`Token` manifests, and [`CLAUDE.md`](CLAUDE.md) for the development workflow.

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
