#!/usr/bin/env bash
# validate-deploy-local.sh -- TDD validation harness for the deploy-local Makefile target
#
# Usage: ./deploy/k8s/tests/validate-deploy-local.sh
# Exits 0 only when all assertions pass.
#
# Strategy: Uses a mock kubectl wrapper that captures invocations and stdin/file content.
# Does NOT require a live k3d cluster.
#
# RED phase: exits non-zero because the deploy-local target does not yet exist in the
# Makefile. Once the target is implemented, this script is expected to exit 0.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MAKEFILE="${PROJECT_ROOT}/Makefile"
VERSION_FILE="${PROJECT_ROOT}/VERSION"
K8S_DIR="${SCRIPT_DIR}/.."

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

assert_not_contains() {
    local label="$1"
    local needle="$2"
    local haystack="$3"
    if echo "$haystack" | grep -qF "$needle"; then
        fail "$label — expected NOT to contain '$needle'"
    else
        pass "$label"
    fi
}

echo "=== validate-deploy-local.sh ==="
echo ""

# -----------------------------------------------------------------------
# Guard: deploy-local target must exist in the Makefile
# This is the primary RED-phase gate. The test exits non-zero here until
# the target is added.
# -----------------------------------------------------------------------
echo "--- Prerequisite: Makefile target ---"

if ! grep -q '^deploy-local:' "$MAKEFILE"; then
    fail "deploy-local target must be defined in Makefile"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    echo ""
    echo "RED: deploy-local target not found in Makefile — implement the target to make this test pass."
    exit 1
fi
pass "deploy-local target exists in Makefile"

# -----------------------------------------------------------------------
# Guard: VERSION file and manifest files must exist
# -----------------------------------------------------------------------
echo ""
echo "--- Prerequisites: VERSION file and manifest files ---"

if [ ! -f "$VERSION_FILE" ]; then
    fail "VERSION file must exist at project root"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "VERSION file exists"

if [ ! -f "${K8S_DIR}/deployment.yaml" ]; then
    fail "deploy/k8s/deployment.yaml must exist"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "deploy/k8s/deployment.yaml exists"

if [ ! -f "${K8S_DIR}/service.yaml" ]; then
    fail "deploy/k8s/service.yaml must exist"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "deploy/k8s/service.yaml exists"

# -----------------------------------------------------------------------
# Read actual version from VERSION file
# -----------------------------------------------------------------------
ACTUAL_VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
if [ -z "$ACTUAL_VERSION" ]; then
    fail "VERSION file must not be empty"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "VERSION file is non-empty (version='${ACTUAL_VERSION}')"

# -----------------------------------------------------------------------
# sed portability assertion
# Verify that the POSIX-portable `sed -e 's/VERSION_PLACEHOLDER/<ver>/g'` form
# succeeds on this platform (BSD macOS and GNU Linux both support -e without
# an extension argument). This validates the sed invocation pattern that the
# Makefile target is expected to use.
# -----------------------------------------------------------------------
echo ""
echo "--- sed portability assertion ---"

SED_INPUT="image: k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER"
SED_OUTPUT="$(echo "$SED_INPUT" | sed -e "s/VERSION_PLACEHOLDER/${ACTUAL_VERSION}/g")"
EXPECTED_SED_OUTPUT="image: k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:${ACTUAL_VERSION}"

assert_eq "portable sed -e replaces VERSION_PLACEHOLDER correctly" \
    "$EXPECTED_SED_OUTPUT" \
    "$SED_OUTPUT"

assert_not_contains "sed output does not contain VERSION_PLACEHOLDER" \
    "VERSION_PLACEHOLDER" \
    "$SED_OUTPUT"

assert_contains "sed output contains actual version string" \
    "$ACTUAL_VERSION" \
    "$SED_OUTPUT"

# -----------------------------------------------------------------------
# Mock kubectl harness
# Create a temporary directory with:
#   - A mock kubectl script that records every invocation (args + stdin/file content)
#   - An override PATH so make uses the mock kubectl
# -----------------------------------------------------------------------
echo ""
echo "--- Mock kubectl invocation harness ---"

TMP_DIR="$(mktemp -d)"
MOCK_KUBECTL="${TMP_DIR}/kubectl"
INVOCATIONS_DIR="${TMP_DIR}/invocations"
mkdir -p "$INVOCATIONS_DIR"

# Write the mock kubectl script.
# It records: the argument list and any content piped through stdin or read from -f <file>.
cat > "$MOCK_KUBECTL" << 'MOCK_EOF'
#!/usr/bin/env bash
# Mock kubectl -- records invocations for test assertions.
set -euo pipefail

INVOCATIONS_DIR="${KUBECTL_INVOCATIONS_DIR:?}"

# Count existing invocations to assign a sequential ID
INVOCATION_ID="$(ls -1 "${INVOCATIONS_DIR}"/ 2>/dev/null | wc -l | tr -d ' ')"
RECORD_DIR="${INVOCATIONS_DIR}/${INVOCATION_ID}"
mkdir -p "$RECORD_DIR"

# Save all arguments
printf '%s\n' "$@" > "${RECORD_DIR}/args"

# If -f <file> is in args, copy the referenced file as manifest content.
# If no -f, capture stdin.
MANIFEST_FILE=""
NEXT_IS_FILE=false
for arg in "$@"; do
    if $NEXT_IS_FILE; then
        MANIFEST_FILE="$arg"
        NEXT_IS_FILE=false
    elif [ "$arg" = "-f" ]; then
        NEXT_IS_FILE=true
    fi
done

if [ -n "$MANIFEST_FILE" ]; then
    if [ "$MANIFEST_FILE" = "-" ]; then
        cat > "${RECORD_DIR}/manifest"
    else
        cat "$MANIFEST_FILE" > "${RECORD_DIR}/manifest" 2>/dev/null || true
    fi
else
    # Check stdin
    if [ -p /dev/stdin ] || [ ! -t 0 ]; then
        cat > "${RECORD_DIR}/manifest"
    else
        touch "${RECORD_DIR}/manifest"
    fi
fi

# Always succeed (we are mocking a happy-path apply)
exit 0
MOCK_EOF

chmod +x "$MOCK_KUBECTL"

# Cleanup on exit
trap 'rm -rf "$TMP_DIR"' EXIT

# -----------------------------------------------------------------------
# Run make deploy-local with the mock kubectl injected via PATH
# -----------------------------------------------------------------------
echo ""
echo "--- Running: make deploy-local (with mock kubectl) ---"

export KUBECTL_INVOCATIONS_DIR="$INVOCATIONS_DIR"

MAKE_EXIT=0
make -C "$PROJECT_ROOT" deploy-local \
    PATH="${TMP_DIR}:${PATH}" \
    > "${TMP_DIR}/make-stdout.txt" 2> "${TMP_DIR}/make-stderr.txt" \
    || MAKE_EXIT=$?

if [ $MAKE_EXIT -ne 0 ]; then
    fail "make deploy-local exited with code ${MAKE_EXIT}"
    echo "  stdout: $(cat "${TMP_DIR}/make-stdout.txt")"
    echo "  stderr: $(cat "${TMP_DIR}/make-stderr.txt")"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "make deploy-local exits 0"

# -----------------------------------------------------------------------
# Assert: mock kubectl was invoked at least twice (one per manifest)
# -----------------------------------------------------------------------
echo ""
echo "--- Assertions: kubectl invocation count ---"

INVOCATION_COUNT="$(ls -1 "$INVOCATIONS_DIR" | wc -l | tr -d ' ')"

if [ "$INVOCATION_COUNT" -ge 2 ]; then
    pass "kubectl was invoked at least twice (deployment.yaml + service.yaml)"
else
    fail "kubectl must be invoked at least twice — got ${INVOCATION_COUNT} invocation(s)"
fi

# -----------------------------------------------------------------------
# Assert: all kubectl invocations used the 'apply' subcommand
# -----------------------------------------------------------------------
echo ""
echo "--- Assertions: kubectl apply subcommand ---"

APPLY_COUNT=0
for inv_dir in "${INVOCATIONS_DIR}"/*/; do
    if grep -q "^apply$" "${inv_dir}/args" 2>/dev/null; then
        APPLY_COUNT=$((APPLY_COUNT + 1))
    fi
done

if [ "$APPLY_COUNT" -ge 2 ]; then
    pass "kubectl apply was called at least twice"
else
    fail "kubectl apply must be called at least twice — found ${APPLY_COUNT} 'apply' invocation(s)"
fi

# -----------------------------------------------------------------------
# Assert: manifests passed to kubectl have VERSION_PLACEHOLDER replaced
# with the actual version string from VERSION file
# -----------------------------------------------------------------------
echo ""
echo "--- Assertions: VERSION_PLACEHOLDER substitution in applied manifests ---"

DEPLOYMENT_FOUND=false
SERVICE_FOUND=false
VERSION_PLACEHOLDER_LEAK=false

for inv_dir in "${INVOCATIONS_DIR}"/*/; do
    MANIFEST_CONTENT=""
    if [ -f "${inv_dir}/manifest" ]; then
        MANIFEST_CONTENT="$(cat "${inv_dir}/manifest")"
    fi

    # Check if this invocation applied the deployment manifest (contains Deployment kind)
    if echo "$MANIFEST_CONTENT" | grep -q "kind: Deployment"; then
        DEPLOYMENT_FOUND=true

        # Assert VERSION_PLACEHOLDER is NOT present
        if echo "$MANIFEST_CONTENT" | grep -qF "VERSION_PLACEHOLDER"; then
            VERSION_PLACEHOLDER_LEAK=true
            fail "deployment manifest passed to kubectl still contains VERSION_PLACEHOLDER"
        else
            pass "deployment manifest does not contain VERSION_PLACEHOLDER"
        fi

        # Assert actual version IS present
        if echo "$MANIFEST_CONTENT" | grep -qF "$ACTUAL_VERSION"; then
            pass "deployment manifest contains actual version string '${ACTUAL_VERSION}'"
        else
            fail "deployment manifest does not contain actual version string '${ACTUAL_VERSION}'"
        fi
    fi

    # Check if this invocation applied the service manifest (contains Service kind)
    if echo "$MANIFEST_CONTENT" | grep -q "kind: Service"; then
        SERVICE_FOUND=true
        pass "service manifest was applied via kubectl"
    fi
done

if ! $DEPLOYMENT_FOUND; then
    fail "no kubectl invocation applied the Deployment manifest (kind: Deployment not found in any captured manifest)"
fi

if ! $SERVICE_FOUND; then
    fail "no kubectl invocation applied the Service manifest (kind: Service not found in any captured manifest)"
fi

# -----------------------------------------------------------------------
# Assert: sed -e form is used in the Makefile recipe (portability check)
# Inspect the Makefile recipe lines for the deploy-local target to confirm
# it uses `sed -e` rather than BSD-only `sed -i ''` or GNU-only `sed -i`.
# -----------------------------------------------------------------------
echo ""
echo "--- Assertions: Makefile recipe uses portable sed -e ---"

# Extract the recipe lines for deploy-local target
# (lines from 'deploy-local:' up to the next target or end of file)
DEPLOY_LOCAL_RECIPE="$(awk '/^deploy-local:/{found=1; next} found && /^\t/{print} found && /^[^ \t]/{exit}' "$MAKEFILE")"

if echo "$DEPLOY_LOCAL_RECIPE" | grep -q "sed"; then
    pass "deploy-local recipe contains sed invocation"

    # Confirm sed -e form is present (portable POSIX form)
    if echo "$DEPLOY_LOCAL_RECIPE" | grep -q "sed -e"; then
        pass "deploy-local recipe uses portable sed -e form"
    else
        fail "deploy-local recipe must use portable 'sed -e' form (not 'sed -i' or other non-POSIX flags)"
    fi
else
    fail "deploy-local recipe must invoke sed for VERSION_PLACEHOLDER substitution"
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
