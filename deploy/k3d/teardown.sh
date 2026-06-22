#!/usr/bin/env bash
# teardown.sh — Destroy the eve-realm k3d cluster and registry
#
# Usage: ./deploy/k3d/teardown.sh
#
# Removes cluster, registry, Docker network, and image volume.

set -euo pipefail

CLUSTER_NAME="eve-realm"
REGISTRY_NAME="k3d-eve-realm-registry.localhost"
NETWORK_NAME="k3d-${CLUSTER_NAME}"
VOLUME_NAME="k3d-${CLUSTER_NAME}-images"

echo "=== eve-realm k3d teardown ==="
echo ""

# --- Registry (before cluster to avoid network endpoint conflicts) -------
echo "--- Registry ---"

if docker inspect "${REGISTRY_NAME}" > /dev/null 2>&1; then
    echo "Disconnecting registry from cluster network..."
    docker network disconnect "${NETWORK_NAME}" "${REGISTRY_NAME}" 2>/dev/null || true
    echo "Deleting registry ${REGISTRY_NAME}..."
    k3d registry delete "${REGISTRY_NAME}"
    echo "Registry deleted."
else
    echo "Registry ${REGISTRY_NAME} does not exist — skipping."
fi

# --- Cluster -------------------------------------------------------------
echo ""
echo "--- Cluster ---"

if k3d cluster list -o json 2>/dev/null | python3 -c "
import sys, json
clusters = json.load(sys.stdin)
sys.exit(0 if any(c['name'] == '${CLUSTER_NAME}' for c in clusters) else 1)
" 2>/dev/null; then
    echo "Deleting cluster ${CLUSTER_NAME}..."
    k3d cluster delete "${CLUSTER_NAME}"
    echo "Cluster deleted."
else
    echo "Cluster ${CLUSTER_NAME} does not exist — skipping."
fi

# --- Orphaned network ----------------------------------------------------
echo ""
echo "--- Network cleanup ---"

if docker network inspect "${NETWORK_NAME}" > /dev/null 2>&1; then
    echo "Removing orphaned network ${NETWORK_NAME}..."
    docker network rm "${NETWORK_NAME}" 2>/dev/null || true
    echo "Network removed."
else
    echo "No orphaned network."
fi

# --- Orphaned volume -----------------------------------------------------
if docker volume inspect "${VOLUME_NAME}" > /dev/null 2>&1; then
    echo "Removing orphaned volume ${VOLUME_NAME}..."
    docker volume rm "${VOLUME_NAME}" 2>/dev/null || true
    echo "Volume removed."
else
    echo "No orphaned volume."
fi

echo ""
echo "=== eve-realm k3d teardown complete ==="
