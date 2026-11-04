# Spec 002 — e2e framework + dockerized Matomo

## Intent

Prove the provider's Matomo integration against a real Matomo instance in CI,
without a Kubernetes cluster. Client-level e2e: the real controller `external`
clients are driven directly against a live, freshly-bootstrapped Matomo.

## Components

- **`docker-compose.e2e.yml`** — pinned `matomo:5.3.0-apache` + `mariadb:11`,
  Matomo port-mapped to `localhost:8080`. The `MATOMO_DATABASE_*` env vars
  pre-fill the installer's database step.
- **`hack/matomo-setup.sh`** — headless bootstrap:
  1. drives the web installer with a cookie jar (database → superuser `admin`
     → first website → finish);
  2. reads the salt Matomo generated in `config.ini.php`;
  3. inserts an app-specific token for `admin` straight into
     `matomo_user_token_auth` as `sha512(token . salt)` (the scheme Matomo uses),
     avoiding any UI token flow;
  4. verifies the token via `API.getMatomoVersion` and exports
     `MATOMO_URL` / `MATOMO_TOKEN` / `MATOMO_ADMIN_PASS` (stdout + `$GITHUB_ENV`).
- **`clients.NewClient(url, token)`** — plain constructor used by the e2e tests.
- **e2e tests** — `//go:build e2e` files colocated in each controller package.
  They skip unless `MATOMO_URL`/`MATOMO_TOKEN` are set, so the normal
  `go test ./...` unit run never touches Matomo.
- **`.github/workflows/e2e.yaml`** — compose up → setup script → `go test -tags e2e`
  → compose down. Separate from the unit CI.

## Wire-protocol facts (discovered via e2e, encoded in code + unit tests)

- Method params must travel in the **query string**; only `token_auth` goes in
  the POST body. Sending `idSite` in the body triggers "An unexpected website
  was found in the request".
- `getSiteFromId` returns `ecommerce`/`sitesearch` as JSON **numbers**;
  `SegmentEditor.get` returns the flags as numbers too → decoded as `json.Number`.
- Not-found signals: sites → "An unexpected website was found in the request";
  deleted segments → "This segment is marked as deleted." Both map to
  `ResourceExists:false` via `clients.IsNotFound`.
- Valid segment dimension is `countryCode`, not `visitCountry`.

## Coverage

- `Site`: create → observe up-to-date → rename (drift) → update → observe
  up-to-date → delete → observe gone.
- `Segment`: same lifecycle (definition drift).
- `Token`: create mints a token; the minted token authenticates a live
  `API.getMatomoVersion` call.

## Acceptance

See PR; all three e2e lifecycles pass against `matomo:5.3.0` locally and in the
`e2e` GitHub Actions job. Unit CI stays Matomo-free (build tag `e2e`).

## Out of scope

kind cluster, CR-level (kubectl) assertions.
