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

Early scaffold. ProviderConfig plumbing and CRD types are in place; the Site,
Segment, and Token controllers are the next milestone.
