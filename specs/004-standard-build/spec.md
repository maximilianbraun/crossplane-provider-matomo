# Spec 004 — Standard Crossplane build system

## Intent

Replace the hand-rolled Makefile and the bespoke release workflow (spec 003)
with the upstream Crossplane build system — the `build/` git submodule and
`build/makelib` — matching `crossplane/provider-template`.

## Changes

- **`build/` submodule** → `github.com/crossplane/build` (pinned).
- **`Makefile`** includes `common.mk` / `output.mk` / `golang.mk` /
  `k8s_tools.mk` / `imagelight.mk` / `xpkg.mk`. Project vars:
  `PROJECT_REPO=github.com/maximilianbraun/crossplane-provider-matomo`,
  `PLATFORMS=linux_amd64 linux_arm64`, `IMAGES=provider-matomo`,
  `XPKG_REG_ORGS=ghcr.io/maximilianbraun`, `XPKGS=provider-matomo`,
  `CROSSPLANE_CLI_VERSION=v2.3.4`. Custom e2e convenience targets
  (`e2e-up`/`e2e-down`/`test-e2e`) are appended; they wrap the existing
  dockerized-Matomo e2e flow and are independent of makelib.
- **`apis/generate.go`** — `//go:generate` directives (controller-gen +
  angryjet) driven by `make generate`. `controller-gen` and `angryjet` are
  tracked as `tool` dependencies in `go.mod`.
- **`internal/version/version.go`** — `Version` var injected via `GO_LDFLAGS`.
- **`cluster/images/provider-matomo/{Dockerfile,Makefile}`** — standard runtime
  image that `ADD`s the pre-built binary; distroless static base.
- **`package/crossplane.yaml`** — standard metadata + `capabilities: [safe-start]`,
  **no** `spec.controller.image`: makelib embeds the runtime image into the xpkg
  (`--embed-runtime-image`), so the package is self-contained.
- Root `Dockerfile` removed (superseded by `cluster/images/...`).
- **CI** (`ci.yaml`): `make lint` / `make test` / `make generate` + check-diff /
  `make -j2 build` (buildx). **Release** (`release.yaml`): `make build` +
  `make publish VERSION=<tag>` → pushes the xpkg to ghcr. The e2e workflow is
  unchanged.

## Supersedes

Spec 003's hand-rolled release and the separate `provider-matomo-controller`
image: the standard build embeds the runtime in the package, so there is a
single published artifact (`ghcr.io/maximilianbraun/provider-matomo:<tag>`).

## Acceptance

1. `build/` submodule present and pinned; `make submodules` works.
2. `make generate` reproduces committed `zz_*` + CRDs with no diff.
3. `make go.build` / `make build` compile the binary (binary validated locally;
   image + xpkg validated in CI — `docker buildx` unavailable on the dev box).
4. `make test` → unit tests green; e2e still gated behind the `e2e` tag.
5. CI + release workflows use makelib targets; e2e workflow unchanged.

## Out of scope

cosign signing, SBOM, Upbound Marketplace, makelib integration-test (`make e2e`)
machinery (we keep the lighter dockerized-Matomo e2e).
