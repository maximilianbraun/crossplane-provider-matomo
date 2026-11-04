# Spec 001 — Resource controllers (Site, Segment, Token)

## Intent

Implement `ExternalClient` reconciliation for the three managed resources
against the Matomo Reporting API, so that applying a CR creates/updates/deletes
the corresponding Matomo object and reflects drift.

## Design

All controllers share one shape: a `connector` resolves the `ProviderConfig`,
builds a cached `*clients.Client`, and returns an `external` holding a `caller`
interface (`Call(ctx, method, params)`) — the interface keeps controllers
unit-testable with a fake. The integer/string ID returned by Matomo is stored
as the Crossplane external name.

### Site — `SitesManager`

| Op      | Matomo method              |
|---------|----------------------------|
| Observe | `getSiteFromId`            |
| Create  | `addSite` → external-name = idSite |
| Update  | `updateSite`               |
| Delete  | `deleteSite`               |

Drift compares spec against observed `name`, main URL, ecommerce, site search,
timezone, currency, type, excluded IPs, excluded query parameters (only fields
set in spec are compared).

### Segment — `SegmentEditor`

`get` / `add` / `update` / `delete`. Drift compares name, definition, target
site, auto-archive, enabled-all-users.

### Token — `UsersManager`

- Create: `createAppSpecificTokenAuth`; the returned token is written to the
  connection secret (`token`, `userLogin`). External name = description.
- Observe: existence inferred from external name; always up-to-date (the value
  cannot be read back).
- Update: unreachable.
- **Delete: no-op** — Matomo offers no single-token delete (`deleteAllTokensForUser`
  would revoke all of the user's tokens). The token stays valid until expiry.
  See README "Token caveat".

## Acceptance

1. `make build` + `make generate` clean, no drift.
2. Unit tests per controller green (Observe exists/not-found/drift, Create sets
   external name, Delete behaviour).
3. Applying a `Site`/`Segment` CR creates the object; manual edit → reconciled;
   delete → removed.
4. Applying a `Token` CR mints a token into the connection secret.

## Out of scope

e2e against a live Matomo (docker-compose) — follow-up.
