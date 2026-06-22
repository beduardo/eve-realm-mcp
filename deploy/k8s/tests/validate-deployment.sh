#!/usr/bin/env bash
# validate-deployment.sh -- TDD validation harness for deploy/k8s/deployment.yaml
#
# Usage: ./deploy/k8s/tests/validate-deployment.sh
# Exits 0 only when all assertions pass.
# Requires: kubectl (with --dry-run=client support), python3

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="${SCRIPT_DIR}/../deployment.yaml"

PASS=0
FAIL=0

pass() {
    echo "  PASS: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo "  FAIL: $1"
    FAIL=$((FAIL + 1))
}

assert_eq() {
    local label="$1"
    local expected="$2"
    local actual="$3"
    if [ "$actual" = "$expected" ]; then
        pass "$label"
    else
        fail "$label — expected='$expected' got='$actual'"
    fi
}

assert_contains() {
    local label="$1"
    local needle="$2"
    local haystack="$3"
    if echo "$haystack" | grep -qF "$needle"; then
        pass "$label"
    else
        fail "$label — expected to contain '$needle'"
    fi
}

echo "=== validate-deployment.sh ==="
echo ""

# -----------------------------------------------------------------------
# Guard: manifest must exist before any other check
# -----------------------------------------------------------------------
echo "--- Prerequisite ---"
if [ ! -f "$MANIFEST" ]; then
    fail "deploy/k8s/deployment.yaml must exist"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "deploy/k8s/deployment.yaml exists"

# -----------------------------------------------------------------------
# Structural validation via kubectl dry-run
# -----------------------------------------------------------------------
echo ""
echo "--- kubectl dry-run ---"

# kubectl --dry-run=client needs a server-side API version to be recognised;
# for offline validation we pipe through stdin with --validate=false so it
# only checks basic YAML structure and kind/apiVersion fields.
if kubectl apply --dry-run=client --validate=false -f "$MANIFEST" > /dev/null 2>&1; then
    pass "kubectl apply --dry-run=client exits 0"
else
    fail "kubectl apply --dry-run=client exits non-zero"
fi

# -----------------------------------------------------------------------
# YAML field assertions via python3
# -----------------------------------------------------------------------
echo ""
echo "--- YAML field assertions ---"

YAML_SCRIPT='
import sys, json

try:
    # Try PyYAML first (usually available)
    import yaml
    with open(sys.argv[1]) as f:
        doc = yaml.safe_load(f)
except ImportError:
    # Fallback: use the stdlib json path after converting — not available for YAML
    # This path should not be reached on a standard Python 3 install with PyYAML
    sys.stderr.write("PyYAML not available; cannot parse YAML\n")
    sys.exit(2)

print(json.dumps(doc))
'

RAW_JSON=$(python3 -c "$YAML_SCRIPT" "$MANIFEST" 2>&1)
PY_EXIT=$?

if [ $PY_EXIT -ne 0 ]; then
    fail "YAML parse failed: $RAW_JSON"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi

# Helper: extract JSON path value using python3 json module
# Usage: json_get '<dot-path>' '<json>'
json_get() {
    local path="$1"
    local json="$2"
    python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
parts = sys.argv[2].split('.')
val = doc
for p in parts:
    if p.startswith('[') and p.endswith(']'):
        val = val[int(p[1:-1])]
    else:
        val = val.get(p, '')
    if val == '' or val is None:
        break
if isinstance(val, (dict, list)):
    print(json.dumps(val))
else:
    print(val)
" "$json" "$path"
}

# -- Metadata --
assert_eq "metadata.name is eve-realm-mcp" \
    "eve-realm-mcp" \
    "$(json_get "metadata.name" "$RAW_JSON")"

assert_eq "metadata.namespace is eve-realm" \
    "eve-realm" \
    "$(json_get "metadata.namespace" "$RAW_JSON")"

assert_eq "metadata.labels.app is eve-realm-mcp" \
    "eve-realm-mcp" \
    "$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
print(doc.get('metadata',{}).get('labels',{}).get('app',''))
" "$RAW_JSON")"

assert_eq "metadata.labels app.kubernetes.io/part-of is eve-realm" \
    "eve-realm" \
    "$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
print(doc.get('metadata',{}).get('labels',{}).get('app.kubernetes.io/part-of',''))
" "$RAW_JSON")"

# -- Container image --
CONTAINER_IMAGE="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
containers = doc.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
print(containers[0].get('image','') if containers else '')
" "$RAW_JSON")"

assert_contains "container image contains VERSION_PLACEHOLDER" \
    "VERSION_PLACEHOLDER" \
    "$CONTAINER_IMAGE"

assert_contains "container image references eve-realm-mcp registry path" \
    "k3d-eve-realm-registry.localhost:5100/eve-realm-mcp" \
    "$CONTAINER_IMAGE"

# -- Ports --
PORTS_JSON="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
containers = doc.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
ports = containers[0].get('ports',[]) if containers else []
print(json.dumps(ports))
" "$RAW_JSON")"

assert_contains "container exposes port 8080" \
    "8080" \
    "$PORTS_JSON"

assert_contains "container exposes port 50051" \
    "50051" \
    "$PORTS_JSON"

# -- envFrom ConfigMap --
ENVFROM_JSON="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
containers = doc.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
envfrom = containers[0].get('envFrom',[]) if containers else []
print(json.dumps(envfrom))
" "$RAW_JSON")"

assert_contains "envFrom references eve-realm-config ConfigMap" \
    "eve-realm-config" \
    "$ENVFROM_JSON"

# -- Liveness probe --
LIVENESS_JSON="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
containers = doc.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
liveness = containers[0].get('livenessProbe',{}) if containers else {}
print(json.dumps(liveness))
" "$RAW_JSON")"

assert_contains "liveness probe path is /healthz" \
    "/healthz" \
    "$LIVENESS_JSON"

assert_contains "liveness probe port is 8080" \
    "8080" \
    "$LIVENESS_JSON"

assert_eq "liveness initialDelaySeconds is 5" \
    "5" \
    "$(python3 -c "
import json, sys
probe = json.loads(sys.argv[1])
print(probe.get('initialDelaySeconds',''))
" "$LIVENESS_JSON")"

assert_eq "liveness periodSeconds is 10" \
    "10" \
    "$(python3 -c "
import json, sys
probe = json.loads(sys.argv[1])
print(probe.get('periodSeconds',''))
" "$LIVENESS_JSON")"

assert_eq "liveness failureThreshold is 3" \
    "3" \
    "$(python3 -c "
import json, sys
probe = json.loads(sys.argv[1])
print(probe.get('failureThreshold',''))
" "$LIVENESS_JSON")"

# -- Readiness probe --
READINESS_JSON="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
containers = doc.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
readiness = containers[0].get('readinessProbe',{}) if containers else {}
print(json.dumps(readiness))
" "$RAW_JSON")"

assert_contains "readiness probe path is /readyz" \
    "/readyz" \
    "$READINESS_JSON"

assert_contains "readiness probe port is 8080" \
    "8080" \
    "$READINESS_JSON"

assert_eq "readiness initialDelaySeconds is 3" \
    "3" \
    "$(python3 -c "
import json, sys
probe = json.loads(sys.argv[1])
print(probe.get('initialDelaySeconds',''))
" "$READINESS_JSON")"

assert_eq "readiness periodSeconds is 5" \
    "5" \
    "$(python3 -c "
import json, sys
probe = json.loads(sys.argv[1])
print(probe.get('periodSeconds',''))
" "$READINESS_JSON")"

assert_eq "readiness failureThreshold is 3" \
    "3" \
    "$(python3 -c "
import json, sys
probe = json.loads(sys.argv[1])
print(probe.get('failureThreshold',''))
" "$READINESS_JSON")"

# -- Resources --
RESOURCES_JSON="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
containers = doc.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
resources = containers[0].get('resources',{}) if containers else {}
print(json.dumps(resources))
" "$RAW_JSON")"

assert_eq "resource requests memory is 128Mi" \
    "128Mi" \
    "$(python3 -c "
import json, sys
res = json.loads(sys.argv[1])
print(res.get('requests',{}).get('memory',''))
" "$RESOURCES_JSON")"

assert_eq "resource requests cpu is 100m" \
    "100m" \
    "$(python3 -c "
import json, sys
res = json.loads(sys.argv[1])
print(res.get('requests',{}).get('cpu',''))
" "$RESOURCES_JSON")"

assert_eq "resource limits memory is 256Mi" \
    "256Mi" \
    "$(python3 -c "
import json, sys
res = json.loads(sys.argv[1])
print(res.get('limits',{}).get('memory',''))
" "$RESOURCES_JSON")"

assert_eq "resource limits cpu is 250m" \
    "250m" \
    "$(python3 -c "
import json, sys
res = json.loads(sys.argv[1])
print(res.get('limits',{}).get('cpu',''))
" "$RESOURCES_JSON")"

# -----------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------
echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ $FAIL -gt 0 ]; then
    exit 1
fi

exit 0
