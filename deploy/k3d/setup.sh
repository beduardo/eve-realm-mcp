#!/usr/bin/env bash
# setup.sh — Bootstrap k3d cluster with registry for eve-realm-mcp
#
# Usage: ./deploy/k3d/setup.sh
#
# Idempotent: safe to run multiple times. Creates only what is missing.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

REGISTRY_NAME="k3d-eve-realm-registry.localhost"
REGISTRY_PORT="5100"
CLUSTER_NAME="eve-realm"
CLUSTER_CONFIG="${SCRIPT_DIR}/cluster.yaml"
INFRA_MANIFEST="${SCRIPT_DIR}/infra.yaml"

echo "=== eve-realm k3d setup ==="
echo ""

# --- Registry (must exist before cluster creation) -----------------------
echo "--- Registry ---"

if ! docker inspect "${REGISTRY_NAME}" > /dev/null 2>&1; then
    echo "Creating registry ${REGISTRY_NAME} on port ${REGISTRY_PORT}..."
    k3d registry create eve-realm-registry.localhost --port "${REGISTRY_PORT}"
    echo "Registry created."
else
    REGISTRY_RUNNING="$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}")"
    if [ "$REGISTRY_RUNNING" = "false" ]; then
        echo "Registry exists but stopped — starting..."
        docker start "${REGISTRY_NAME}"
        echo "Registry started."
    else
        echo "Registry already running."
    fi
fi

# Verify registry is reachable before proceeding
echo "Verifying registry is reachable..."
for i in 1 2 3 4 5; do
    if curl -sf "http://localhost:${REGISTRY_PORT}/v2/" > /dev/null 2>&1; then
        echo "Registry is reachable."
        break
    fi
    if [ "$i" = "5" ]; then
        echo "ERROR: Registry not reachable at localhost:${REGISTRY_PORT} after 5 attempts."
        exit 1
    fi
    sleep 1
done

# --- Cluster -------------------------------------------------------------
echo ""
echo "--- Cluster ---"

CLUSTER_EXISTS="$(k3d cluster list -o json 2>/dev/null | python3 -c "
import sys, json
clusters = json.load(sys.stdin)
print('yes' if any(c['name'] == '${CLUSTER_NAME}' for c in clusters) else 'no')
" 2>/dev/null || echo "no")"

if [ "$CLUSTER_EXISTS" = "no" ]; then
    echo "Creating cluster ${CLUSTER_NAME}..."
    k3d cluster create --config "${CLUSTER_CONFIG}"
    echo "Cluster created."
else
    SERVERS_RUNNING="$(k3d cluster list -o json | python3 -c "
import sys, json
clusters = json.load(sys.stdin)
c = next(x for x in clusters if x['name'] == '${CLUSTER_NAME}')
print(c.get('serversRunning', 0))
")"
    if [ "$SERVERS_RUNNING" = "0" ]; then
        echo "Cluster exists but stopped — starting..."
        k3d cluster start "${CLUSTER_NAME}"
        echo "Cluster started."
    else
        echo "Cluster already running."
    fi
fi

# --- Registry-to-cluster network ----------------------------------------
echo ""
echo "--- Registry network ---"

CONNECTED="$(docker network inspect "k3d-${CLUSTER_NAME}" 2>/dev/null \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
containers = data[0].get('Containers', {})
print('yes' if any('${REGISTRY_NAME}' in c.get('Name', '') for c in containers.values()) else 'no')
" 2>/dev/null || echo "no")"

if [ "$CONNECTED" = "no" ]; then
    echo "Connecting registry to cluster network..."
    docker network connect "k3d-${CLUSTER_NAME}" "${REGISTRY_NAME}" 2>/dev/null || true
    echo "Connected."
else
    echo "Registry already connected to cluster network."
fi

# --- kubeconfig ----------------------------------------------------------
echo ""
echo "--- kubeconfig ---"

k3d kubeconfig merge "${CLUSTER_NAME}" --kubeconfig-switch-context 2>/dev/null
echo "kubectl context set to k3d-${CLUSTER_NAME}."

# --- Infrastructure manifests -------------------------------------------
echo ""
echo "--- Infrastructure ---"

echo "Applying namespace and configmap..."
kubectl apply -f "${INFRA_MANIFEST}"

# --- Wait for node ready -------------------------------------------------
echo ""
echo "--- Readiness ---"

echo "Waiting for node to be ready..."
kubectl wait --for=condition=Ready node --all --timeout=60s

echo ""
echo "=== eve-realm k3d cluster is ready ==="
echo ""
echo "Next steps:"
echo "  make docker-build docker-push deploy-local wait-rollout"
echo ""
