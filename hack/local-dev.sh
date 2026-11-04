#!/usr/bin/env bash
# Local out-of-cluster dev loop for provider-matomo.
#
#   hack/local-dev.sh up     # Matomo (docker) + kind cluster + CRDs + a
#                            # ProviderConfig/secret (token from matomo-setup.sh)
#                            # + a sample Site CR
#   hack/local-dev.sh run    # run the controller on the host against the kind
#                            # cluster, with --debug (foreground; Ctrl-C to stop)
#   hack/local-dev.sh down   # delete the kind cluster + tear down Matomo
#
# The provider runs on the host, talks to the kind API via an isolated
# kubeconfig, and to Matomo at http://localhost:8080.
set -euo pipefail

CLUSTER="${KIND_CLUSTER:-provider-matomo-dev}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
WORK="$ROOT/.work"
KCFG="$WORK/dev-kubeconfig"
mkdir -p "$WORK"

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing required tool: $1" >&2; exit 1; }; }

up() {
  need docker; need kind; need kubectl
  echo "== Matomo (docker compose) =="
  docker compose -f docker-compose.e2e.yml up -d
  echo "== bootstrap Matomo + token =="
  bash hack/matomo-setup.sh | grep -E '^MATOMO_(URL|TOKEN)=' > "$WORK/matomo.env"
  # shellcheck disable=SC1091
  . "$WORK/matomo.env"

  echo "== kind cluster '$CLUSTER' =="
  kind get clusters 2>/dev/null | grep -qx "$CLUSTER" || kind create cluster --name "$CLUSTER"
  kind get kubeconfig --name "$CLUSTER" > "$KCFG"
  export KUBECONFIG="$KCFG"

  echo "== install CRDs =="
  kubectl apply -R -f package/crds

  echo "== ProviderConfig + credentials =="
  kubectl get ns crossplane-system >/dev/null 2>&1 || kubectl create ns crossplane-system
  kubectl create secret generic matomo-creds -n crossplane-system \
    --from-literal=token_auth="$MATOMO_TOKEN" --dry-run=client -o yaml | kubectl apply -f -
  cat <<EOF | kubectl apply -f -
apiVersion: matomo.m.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  url: ${MATOMO_URL}
  credentials:
    tokenAuthSecretRef:
      namespace: crossplane-system
      name: matomo-creds
      key: token_auth
EOF

  echo "== sample Site =="
  kubectl apply -f examples/site.yaml

  echo
  echo "ready. Run the controller with:  make dev-run"
}

run() {
  need go
  [ -f "$KCFG" ] || { echo "no dev kubeconfig; run 'make dev-up' first" >&2; exit 1; }
  export KUBECONFIG="$KCFG"
  echo "running provider against kind cluster '$CLUSTER' (Ctrl-C to stop)"
  # Metrics off :8080 to avoid colliding with the dockerized Matomo port.
  exec go run ./cmd/provider --debug --metrics-bind-address=:8081
}

down() {
  kind delete cluster --name "$CLUSTER" 2>/dev/null || true
  docker compose -f docker-compose.e2e.yml down -v 2>/dev/null || true
  rm -f "$KCFG" "$WORK/matomo.env"
  echo "torn down."
}

case "${1:-up}" in
  up) up ;;
  run) run ;;
  down) down ;;
  *) echo "usage: $0 {up|run|down}" >&2; exit 1 ;;
esac
