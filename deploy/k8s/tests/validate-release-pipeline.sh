#!/usr/bin/env bash
# validate-release-pipeline.sh -- TDD validation harness for release pipeline Makefile targets
#
# Usage: ./deploy/k8s/tests/validate-release-pipeline.sh
# Exits 0 only when all assertions pass.
#
# Strategy: Inspects Makefile target prerequisites via grep/awk to assert the
# seven-step chain for release-patch, release-minor, and release-major without
# executing a live build or cluster deploy.
#
# Expected chain (release-patch):
#   test -> bump-patch -> build-prod -> docker-build -> docker-push -> deploy-local -> wait-rollout
#
# Expected chain (release-minor):
#   test -> bump-minor -> build-prod -> docker-build -> docker-push -> deploy-local -> wait-rollout
#
# Expected chain (release-major):
#   test -> bump-major -> build-prod -> docker-build -> docker-push -> deploy-local -> wait-rollout
#
# RED phase: exits non-zero because:
#   - release-patch only chains: test bump-patch build-prod (missing docker-build, docker-push, deploy-local, wait-rollout)
#   - release-minor and release-major do not exist in the Makefile

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

assert_not_contains() {
    local label="$1"
    local needle="$2"
    local haystack="$3"
    if echo "$haystack" | grep -qF -- "$needle"; then
        fail "$label — expected NOT to contain '$needle'"
    else
        pass "$label"
    fi
}

echo "=== validate-release-pipeline.sh ==="
echo ""

# -----------------------------------------------------------------------
# Helper: extract_prerequisites <target>
#
# Extracts the prerequisites (deps) from a Makefile target definition line.
# A target definition line has the form:
#   <target>: [prereq1 prereq2 ...]
#
# Returns the prerequisites as a single space-separated string, or empty
# if the target is not found or has no prerequisites.
# -----------------------------------------------------------------------
extract_prerequisites() {
    local target="$1"
    # Match lines like: release-patch: test bump-patch build-prod
    # Capture everything after the colon (the prerequisite list).
    # Strip leading/trailing whitespace.
    grep -E "^${target}[[:space:]]*:" "$MAKEFILE" \
        | head -n1 \
        | sed -e "s/^${target}[[:space:]]*:[[:space:]]*//" \
        | tr -s ' ' \
        | sed -e 's/[[:space:]]*$//'
}

# -----------------------------------------------------------------------
# Helper: extract_recipe_lines <target>
#
# Extracts the recipe (tab-prefixed) lines for a Makefile target.
# Returns lines from the target definition up to the next non-indented line.
# -----------------------------------------------------------------------
extract_recipe_lines() {
    local target="$1"
    awk "/^${target}[[:space:]]*:/{found=1; next} found && /^\t/{print} found && /^[^ \t]/{exit}" "$MAKEFILE"
}

# -----------------------------------------------------------------------
# Helper: assert_seven_step_chain <target> <bump-step>
#
# Asserts that the given release target has all seven prerequisites in the
# correct order: test <bump-step> build-prod docker-build docker-push deploy-local wait-rollout
# -----------------------------------------------------------------------
assert_seven_step_chain() {
    local target="$1"
    local bump_step="$2"
    local expected_chain="test ${bump_step} build-prod docker-build docker-push deploy-local wait-rollout"

    echo "--- Target: ${target} ---"

    # Assert the target exists
    if ! grep -qE "^${target}[[:space:]]*:" "$MAKEFILE"; then
        fail "${target} target must be defined in Makefile"
        return
    fi
    pass "${target} target exists in Makefile"

    # Extract prerequisites
    local prereqs
    prereqs="$(extract_prerequisites "$target")"

    # Assert exact prerequisite list matches expected chain
    assert_eq "${target} prerequisites match full seven-step chain" \
        "$expected_chain" \
        "$prereqs"

    # Assert individual steps are present (fine-grained failure messages)
    assert_contains "${target} prerequisites contain 'test'" \
        "test" \
        "$prereqs"

    assert_contains "${target} prerequisites contain '${bump_step}'" \
        "$bump_step" \
        "$prereqs"

    assert_contains "${target} prerequisites contain 'build-prod'" \
        "build-prod" \
        "$prereqs"

    assert_contains "${target} prerequisites contain 'docker-build'" \
        "docker-build" \
        "$prereqs"

    assert_contains "${target} prerequisites contain 'docker-push'" \
        "docker-push" \
        "$prereqs"

    assert_contains "${target} prerequisites contain 'deploy-local'" \
        "deploy-local" \
        "$prereqs"

    assert_contains "${target} prerequisites contain 'wait-rollout'" \
        "wait-rollout" \
        "$prereqs"

    # Assert ordering: each step appears before the next in the prerequisite string
    # We do this by checking the position of each step relative to the next.
    # Strategy: walk through expected steps and verify their relative order in prereqs.
    local steps="test ${bump_step} build-prod docker-build docker-push deploy-local wait-rollout"
    local prev_step=""
    local order_ok=true

    for step in $steps; do
        if [ -n "$prev_step" ]; then
            # prev_step must appear before step in prereqs
            local pos_prev pos_step
            pos_prev="$(echo "$prereqs" | awk "{n=split(\$0,a,\" \"); for(i=1;i<=n;i++){if(a[i]==\"${prev_step}\"){print i; exit}}}")"
            pos_step="$(echo "$prereqs" | awk "{n=split(\$0,a,\" \"); for(i=1;i<=n;i++){if(a[i]==\"${step}\"){print i; exit}}}")"

            if [ -z "$pos_prev" ] || [ -z "$pos_step" ]; then
                # Missing steps already caught above; skip ordering check
                true
            elif [ "$pos_prev" -lt "$pos_step" ]; then
                pass "${target}: '${prev_step}' appears before '${step}' (ordering correct)"
            else
                fail "${target}: '${prev_step}' must appear before '${step}' in prerequisite list — got positions prev=${pos_prev} step=${pos_step}"
                order_ok=false
            fi
        fi
        prev_step="$step"
    done
}

# -----------------------------------------------------------------------
# Helper: assert_no_error_suppression <target>
#
# Validates that the recipe lines for the given target do not suppress errors
# via '|| true' or a leading '-' prefix on recipe lines.
# -----------------------------------------------------------------------
assert_no_error_suppression() {
    local target="$1"

    echo ""
    echo "--- Error suppression check: ${target} ---"

    local recipe
    recipe="$(extract_recipe_lines "$target")"

    if [ -z "$recipe" ]; then
        # Target may have no recipe body (prerequisites only). That is fine —
        # Make's dependency chaining provides error propagation at the prerequisite level.
        pass "${target} has no recipe body (dependency chaining provides failure propagation)"
        return
    fi

    # Check for '|| true' pattern (explicit error suppression)
    assert_not_contains "${target} recipe does not contain '|| true'" \
        "|| true" \
        "$recipe"

    # Check for leading '-' prefix on recipe lines (Make-level error suppression)
    if echo "$recipe" | grep -qE $'^\t-'; then
        fail "${target} recipe must not use '-' prefix on recipe lines (suppresses errors)"
    else
        pass "${target} recipe does not use '-' prefix on recipe lines"
    fi
}

# -----------------------------------------------------------------------
# Main assertions
# -----------------------------------------------------------------------

echo "--- Makefile exists ---"
if [ ! -f "$MAKEFILE" ]; then
    fail "Makefile not found at ${MAKEFILE}"
    echo ""
    echo "Results: ${PASS} passed, ${FAIL} failed"
    exit 1
fi
pass "Makefile found at ${MAKEFILE}"

echo ""

# Assert seven-step chains for all three release targets
assert_seven_step_chain "release-patch" "bump-patch"
echo ""
assert_seven_step_chain "release-minor" "bump-minor"
echo ""
assert_seven_step_chain "release-major" "bump-major"

# Assert error suppression is absent in all three release target recipes
assert_no_error_suppression "release-patch"
assert_no_error_suppression "release-minor"
assert_no_error_suppression "release-major"

# -----------------------------------------------------------------------
# Structural integrity: assert all seven prerequisite targets are declared
# as .PHONY or have their own target definitions (they must exist in Makefile)
# -----------------------------------------------------------------------
echo ""
echo "--- Prerequisite targets exist in Makefile ---"

REQUIRED_TARGETS="test bump-patch bump-minor bump-major build-prod docker-build docker-push deploy-local wait-rollout"
for t in $REQUIRED_TARGETS; do
    if grep -qE "^${t}[[:space:]]*(:|$)" "$MAKEFILE"; then
        pass "prerequisite target '${t}' is defined in Makefile"
    else
        fail "prerequisite target '${t}' must be defined in Makefile"
    fi
done

# -----------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------
echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ $FAIL -gt 0 ]; then
    echo "RED: one or more assertions failed — implement the full release pipeline chains to make this test pass."
    exit 1
fi

exit 0
