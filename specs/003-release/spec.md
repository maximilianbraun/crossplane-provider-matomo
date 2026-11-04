# Spec 003 — Release / publish workflow

## Intent

On a git tag `v*`, publish the provider so it is installable from a registry:
build and push the controller image and the Crossplane provider package (xpkg)
to ghcr.io.

## Artifacts

Two distinct OCI references (a package and its runtime image cannot share a tag):

| Artifact | Reference |
|----------|-----------|
| Provider package (install target) | `ghcr.io/maximilianbraun/provider-matomo:<tag>` |
| Controller runtime image | `ghcr.io/maximilianbraun/provider-matomo-controller:<tag>` (+ `:latest`) |

`package/crossplane.yaml` references the controller image; the committed value
is `:latest` and the release job rewrites it to the exact tag before building
the package, so a released package pins its controller image.

## Workflow (`.github/workflows/release.yaml`)

Trigger: tag push `v*`, or manual `workflow_dispatch` with a `tag` input.
`permissions: packages: write`.

1. buildx multi-arch (`linux/amd64,linux/arm64`) controller image → push `<tag>` + `latest`.
2. `sed` the controller image tag into `package/crossplane.yaml`.
3. Install crossplane CLI, `xpkg build --package-root=package`.
4. `xpkg push -f provider-matomo.xpkg ghcr.io/maximilianbraun/provider-matomo:<tag>`.

Auth is `GITHUB_TOKEN` via `docker/login-action`; `crossplane xpkg push` reuses
the docker credential file.

## Acceptance

1. Tag `vX.Y.Z` → controller image (multi-arch) and xpkg both pushed at `vX.Y.Z`.
2. `xpkg build` succeeds (validated locally with crossplane CLI v2.5.0).
3. The published package pins the matching controller image tag, not `latest`.
4. `make xpkg` builds the package locally.

## Install (result)

```sh
crossplane xpkg install provider ghcr.io/maximilianbraun/provider-matomo:vX.Y.Z
```

## Out of scope

Image signing (cosign), SBOM, Upbound Marketplace publishing.
