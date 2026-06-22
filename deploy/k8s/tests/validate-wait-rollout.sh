#!/usr/bin/env bash
# validate-wait-rollout.sh -- TDD validation harness for the wait-rollout Makefile target
#
# Usage: ./deploy/k8s/tests/validate-wait-rollout.sh
# Exits 0 only when all assertions pass.
#
# Strategy: Uses a mock kubectl wrapper that captures invocation arguments and can
# simulate non-zero exit codes to verify exit-code propagation. Does NOT require a
# live k3d cluster.
#
# RED phase: exits non-zero because the wait-rollout target does not yet exist in the
# Makefile. Once the target is implemented, this script is expected to exit 0.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MAKEFILE="${PROJECT_ROOT}/Makefile"

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
    if echo "$haystack" | grep -qF -- "$needle"; then
        pass "$label"
    else
        fail "$label — expected to contain '$needle'"
    fi
}

echo "=== validate-wait-rollout.sh ==="
echo ""

# -----------------------------------------------------------------------
# Guard: wait-rollout target must exist in the Makefile
# This is the primary RED-phase gate. The test exits non-zero here until
# the target is added.
# -----------------------------------------------------------------------
echo "--- Prerequisite: Makefile target ---"

if ! grep -q '^wait-rollout:' "$MAKEFILE"; then
    fail "wait-rollout target must be defined in Makefile"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    echo ""
    echo "RED: wait-rollout target not found in Makefile — implement the target to make this test pass."
    exit 1
fi
pass "wait-rollout target exists in Makefile"

# -----------------------------------------------------------------------
# Helper: create a temporary directory with a mock kubectl
# Arguments:
#   $1 -- exit code the mock kubectl should return
# Sets globals: TMP_DIR, MOCK_KUBECTL, INVOCATIONS_DIR
# -----------------------------------------------------------------------
setup_mock_kubectl() {
    local mock_exit_code="${1:-0}"

    TMP_DIR="$(mktemp -d)"
    MOCK_KUBECTL="${TMP_DIR}/kubectl"
    INVOCATIONS_DIR="${TMP_DIR}/invocations"
    mkdir -p "$INVOCATIONS_DIR"

    cat > "$MOCK_KUBECTL" << MOCK_EOF
#!/usr/bin/env bash
# Mock kubectl -- records invocations and returns a configurable exit code.
set -euo pipefail

INVOCATIONS_DIR="\${KUBECTL_INVOCATIONS_DIR:?}"
MOCK_EXIT_CODE="\${KUBECTL_MOCK_EXIT_CODE:-0}"

INVOCATION_ID="\$(ls -1 "\${INVOCATIONS_DIR}"/ 2>/dev/null | wc -l | tr -d ' ')"
RECORD_DIR="\${INVOCATIONS_DIR}/\${INVOCATION_ID}"
mkdir -p "\$RECORD_DIR"

printf '%s\n' "\$@" > "\${RECORD_DIR}/args"

exit \$MOCK_EXIT_CODE
MOCK_EOF

    chmod +x "$MOCK_KUBECTL"
    export KUBECTL_INVOCATIONS_DIR="$INVOCATIONS_DIR"
    export KUBECTL_MOCK_EXIT_CODE="$mock_exit_code"
}

cleanup_tmp() {
    if [ -n "${TMP_DIR:-}" ]; then
        rm -rf "$TMP_DIR"
    fi
}

# -----------------------------------------------------------------------
# Test 1: Correct arguments passed to kubectl
# Run make wait-rollout with a mock kubectl that exits 0, then inspect args.
# -----------------------------------------------------------------------
echo ""
echo "--- Test 1: kubectl receives correct rollout status arguments ---"

setup_mock_kubectl 0
trap 'cleanup_tmp' EXIT

MAKE_EXIT=0
make -C "$PROJECT_ROOT" wait-rollout \
    PATH="${TMP_DIR}:${PATH}" \
    > "${TMP_DIR}/make-stdout.txt" 2> "${TMP_DIR}/make-stderr.txt" \
    || MAKE_EXIT=$?

if [ $MAKE_EXIT -ne 0 ]; then
    fail "make wait-rollout exited with code ${MAKE_EXIT} (expected 0 when kubectl succeeds)"
    echo "  stdout: $(cat "${TMP_DIR}/make-stdout.txt")"
    echo "  stderr: $(cat "${TMP_DIR}/make-stderr.txt")"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "make wait-rollout exits 0 when kubectl exits 0"

# Assert: exactly one kubectl invocation occurred
INVOCATION_COUNT="$(ls -1 "$INVOCATIONS_DIR" | wc -l | tr -d ' ')"
assert_eq "kubectl was invoked exactly once" "1" "$INVOCATION_COUNT"

# Read the recorded arguments for the single invocation
RECORDED_ARGS_FILE="${INVOCATIONS_DIR}/0/args"

if [ ! -f "$RECORDED_ARGS_FILE" ]; then
    fail "kubectl argument record not found at ${RECORDED_ARGS_FILE}"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi

RECORDED_ARGS="$(cat "$RECORDED_ARGS_FILE")"

# Assert each required argument is present (one per line from mock)
assert_contains "kubectl received 'rollout' subcommand" \
    "rollout" \
    "$RECORDED_ARGS"

assert_contains "kubectl received 'status' argument" \
    "status" \
    "$RECORDED_ARGS"

assert_contains "kubectl received deployment name 'deployment/eve-realm-mcp'" \
    "deployment/eve-realm-mcp" \
    "$RECORDED_ARGS"

assert_contains "kubectl received namespace flag '-n eve-realm'" \
    "eve-realm" \
    "$RECORDED_ARGS"

assert_contains "kubectl received timeout flag '--timeout=120s'" \
    "--timeout=120s" \
    "$RECORDED_ARGS"

# Assert the full argument sequence matches expected: rollout status deployment/eve-realm-mcp -n eve-realm --timeout=120s
# Reconstruct args as space-separated for full-sequence check
ARGS_INLINE="$(tr '\n' ' ' < "$RECORDED_ARGS_FILE" | sed -e 's/[[:space:]]*$//')"
assert_contains "kubectl argument sequence contains 'rollout status deployment/eve-realm-mcp'" \
    "rollout status deployment/eve-realm-mcp" \
    "$ARGS_INLINE"

# -----------------------------------------------------------------------
# Test 2: Non-zero exit from kubectl propagates through make wait-rollout
# Re-run with a mock kubectl that exits 1 and assert make also exits non-zero.
# -----------------------------------------------------------------------
echo ""
echo "--- Test 2: non-zero kubectl exit propagates as non-zero make exit ---"

cleanup_tmp
trap '' EXIT  # Reset trap before re-setup
setup_mock_kubectl 1
trap 'cleanup_tmp' EXIT

MAKE_FAIL_EXIT=0
make -C "$PROJECT_ROOT" wait-rollout \
    PATH="${TMP_DIR}:${PATH}" \
    > "${TMP_DIR}/make-fail-stdout.txt" 2> "${TMP_DIR}/make-fail-stderr.txt" \
    || MAKE_FAIL_EXIT=$?

if [ $MAKE_FAIL_EXIT -ne 0 ]; then
    pass "make wait-rollout exits non-zero (${MAKE_FAIL_EXIT}) when kubectl exits 1"
else
    fail "make wait-rollout must propagate non-zero exit from kubectl — got exit 0"
fi

# -----------------------------------------------------------------------
# Test 3: Makefile recipe inspection for correct kubectl command
# Parse the recipe lines for the wait-rollout target from the Makefile
# to statically verify the expected kubectl invocation pattern.
# -----------------------------------------------------------------------
echo ""
echo "--- Test 3: Makefile recipe static inspection ---"

# Extract recipe lines for wait-rollout target
WAIT_ROLLOUT_RECIPE="$(awk '/^wait-rollout:/{found=1; next} found && /^\t/{print} found && /^[^ \t]/{exit}' "$MAKEFILE")"

if [ -z "$WAIT_ROLLOUT_RECIPE" ]; then
    fail "wait-rollout recipe body is empty"
else
    pass "wait-rollout recipe body is non-empty"

    assert_contains "recipe invokes kubectl" \
        "kubectl" \
        "$WAIT_ROLLOUT_RECIPE"

    assert_contains "recipe invokes rollout status" \
        "rollout status" \
        "$WAIT_ROLLOUT_RECIPE"

    assert_contains "recipe targets deployment/eve-realm-mcp" \
        "deployment/eve-realm-mcp" \
        "$WAIT_ROLLOUT_RECIPE"

    assert_contains "recipe specifies namespace -n eve-realm" \
        "-n eve-realm" \
        "$WAIT_ROLLOUT_RECIPE"

    assert_contains "recipe specifies --timeout=120s" \
        "--timeout=120s" \
        "$WAIT_ROLLOUT_RECIPE"
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
