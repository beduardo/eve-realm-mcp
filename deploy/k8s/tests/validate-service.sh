#!/usr/bin/env bash
# validate-service.sh -- TDD validation harness for deploy/k8s/service.yaml
#
# Usage: ./deploy/k8s/tests/validate-service.sh
# Exits 0 only when all assertions pass.
# Requires: kubectl (with --dry-run=client support), python3

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="${SCRIPT_DIR}/../service.yaml"
DEPLOYMENT_MANIFEST="${SCRIPT_DIR}/../deployment.yaml"

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

echo "=== validate-service.sh ==="
echo ""

# -----------------------------------------------------------------------
# Guard: manifest must exist before any other check
# -----------------------------------------------------------------------
echo "--- Prerequisite ---"
if [ ! -f "$MANIFEST" ]; then
    fail "deploy/k8s/service.yaml must exist"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "deploy/k8s/service.yaml exists"

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

# -- Service type --
assert_eq "spec.type is ClusterIP" \
    "ClusterIP" \
    "$(json_get "spec.type" "$RAW_JSON")"

# -- Selector --
assert_eq "spec.selector.app is eve-realm-mcp" \
    "eve-realm-mcp" \
    "$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
print(doc.get('spec',{}).get('selector',{}).get('app',''))
" "$RAW_JSON")"

# -- Ports --
PORTS_JSON="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
ports = doc.get('spec',{}).get('ports',[])
print(json.dumps(ports))
" "$RAW_JSON")"

# Port 8080 named http
assert_eq "port 8080 is named http" \
    "http" \
    "$(python3 -c "
import json, sys
ports = json.loads(sys.argv[1])
match = next((p for p in ports if p.get('port') == 8080), {})
print(match.get('name',''))
" "$PORTS_JSON")"

assert_eq "port 8080 targetPort is 8080" \
    "8080" \
    "$(python3 -c "
import json, sys
ports = json.loads(sys.argv[1])
match = next((p for p in ports if p.get('port') == 8080), {})
print(str(match.get('targetPort','')))
" "$PORTS_JSON")"

# Port 50051 named grpc
assert_eq "port 50051 is named grpc" \
    "grpc" \
    "$(python3 -c "
import json, sys
ports = json.loads(sys.argv[1])
match = next((p for p in ports if p.get('port') == 50051), {})
print(match.get('name',''))
" "$PORTS_JSON")"

assert_eq "port 50051 targetPort is 50051" \
    "50051" \
    "$(python3 -c "
import json, sys
ports = json.loads(sys.argv[1])
match = next((p for p in ports if p.get('port') == 50051), {})
print(str(match.get('targetPort','')))
" "$PORTS_JSON")"

# -----------------------------------------------------------------------
# Cross-validation: Service selector must match pod template labels
# in deployment.yaml
# -----------------------------------------------------------------------
echo ""
echo "--- Cross-validation: selector vs deployment pod template labels ---"

if [ ! -f "$DEPLOYMENT_MANIFEST" ]; then
    fail "deploy/k8s/deployment.yaml must exist for cross-validation"
else
    pass "deploy/k8s/deployment.yaml exists for cross-validation"

    DEPLOYMENT_JSON=$(python3 -c "$YAML_SCRIPT" "$DEPLOYMENT_MANIFEST" 2>&1)
    DEP_PY_EXIT=$?

    if [ $DEP_PY_EXIT -ne 0 ]; then
        fail "deployment.yaml YAML parse failed: $DEPLOYMENT_JSON"
    else
        # Extract Service selector app label
        SERVICE_SELECTOR_APP="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
print(doc.get('spec',{}).get('selector',{}).get('app',''))
" "$RAW_JSON")"

        # Extract pod template labels app label from deployment
        POD_TEMPLATE_APP="$(python3 -c "
import json, sys
doc = json.loads(sys.argv[1])
print(doc.get('spec',{}).get('template',{}).get('metadata',{}).get('labels',{}).get('app',''))
" "$DEPLOYMENT_JSON")"

        assert_eq "Service selector app matches deployment pod template label app" \
            "$POD_TEMPLATE_APP" \
            "$SERVICE_SELECTOR_APP"
    fi
fi

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
