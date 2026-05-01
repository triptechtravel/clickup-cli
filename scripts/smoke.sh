#!/usr/bin/env bash
# scripts/smoke.sh — round-trip the typed-wrapper code paths against a real
# ClickUp workspace. Catches OpenAPI spec drift (response shape mismatches,
# argument encoding bugs) that unit tests can't see.
#
# Resources are created in `--current` (the configured sprint list) under a
# unique per-run token and torn down on exit. Failures abort the run and
# surface the underlying CLI error.
#
# Requirements:
#   - `clickup` binary on PATH (or `BIN=…` env override)
#   - logged-in via `clickup auth login` (or run `make smoke BIN=./bin/clickup`)
#   - `--current` must resolve (a sprint folder configured)
#
# Usage:
#   make smoke                 # uses installed `clickup`
#   BIN=./bin/clickup make smoke   # uses a local build

set -euo pipefail

BIN="${BIN:-clickup}"
TOKEN="smoke$(date +%s)"
PARENT_ID=""
SUBTASK_IDS=()

cleanup() {
  set +e
  if [ ${#SUBTASK_IDS[@]} -gt 0 ]; then
    "$BIN" task delete "${SUBTASK_IDS[@]}" -y > /dev/null 2>&1 || true
  fi
  if [ -n "$PARENT_ID" ]; then
    "$BIN" task delete "$PARENT_ID" -y > /dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

step() { printf '\n→ %s\n' "$1"; }
ok()   { printf '  ✓ %s\n' "$1"; }
fail() { printf '  ✗ %s\n' "$1" >&2; exit 1; }

step "auth check"
"$BIN" auth status > /dev/null 2>&1 || fail "not logged in (run 'clickup auth login')"
ok "logged in"

# --- task create (CreateTaskLocal) ----------------------------------------
step "task create — exercises the create response decode"
PARENT_ID="$("$BIN" task create --current \
  --name "[Smoke Test] $TOKEN parent" \
  --status "to do" \
  --json --jq '.id' --raw 2>/dev/null | tail -1)"
[ -n "$PARENT_ID" ] || fail "task create returned empty id"
ok "created parent $PARENT_ID"

# --- task create with parent (subtask path) -------------------------------
step "task create — subtask under parent"
SUB_ID="$("$BIN" task create --current \
  --name "[Smoke Test] $TOKEN subtask" \
  --parent "$PARENT_ID" \
  --status "to do" \
  --json --jq '.id' --raw 2>/dev/null | tail -1)"
[ -n "$SUB_ID" ] || fail "subtask create returned empty id"
SUBTASK_IDS+=("$SUB_ID")
ok "created subtask $SUB_ID"

# --- task list --include-subtasks (PR #17 wire-up) ------------------------
step "task list --include-subtasks — exercises the flag plumbing"
LIST_ID=$("$BIN" task view "$PARENT_ID" --json --jq '.list.id' --raw 2>/dev/null | tail -1)
COUNT=$("$BIN" task list --list-id "$LIST_ID" --include-subtasks --include-closed \
  --json --jq "map(select(.name | contains(\"$TOKEN\"))) | length" 2>/dev/null | tail -1)
[ "$COUNT" -ge 2 ] || fail "task list with --include-subtasks returned $COUNT for token $TOKEN, expected >=2"
ok "task list returned $COUNT items matching token (parent + subtask)"

# --- comment add (CreateTaskComment, typed response) ----------------------
step "comment add — exercises typed response decode (the v0.34.1 regression)"
COMMENT_ID="$("$BIN" comment add "$PARENT_ID" "Smoke probe: **bold** and \`code\` should render." \
  --json --jq '.id' --raw 2>/dev/null | tail -1)"
[ -n "$COMMENT_ID" ] && [ "$COMMENT_ID" != "null" ] \
  || fail "comment add did not return an id (response decode drift, or --json output broken)"
ok "comment added (id $COMMENT_ID)"

# --- comment list (read) --------------------------------------------------
step "comment list — read-side decode"
LIST_FIRST_ID=$("$BIN" comment list "$PARENT_ID" --json --jq '.[0].id' --raw 2>/dev/null | tail -1)
[ -n "$LIST_FIRST_ID" ] && [ "$LIST_FIRST_ID" != "null" ] || fail "comment list returned no id"
ok "comment listed (newest id $LIST_FIRST_ID)"

# --- comment edit (UpdateComment) -----------------------------------------
step "comment edit — exercises UpdateComment typed wrapper"
"$BIN" comment edit "$COMMENT_ID" "Smoke probe (edited)" > /dev/null 2>&1 \
  || fail "comment edit failed"
ok "comment edit succeeded"

# --- comment reply (CreateThreadedComment) --------------------------------
step "comment reply — exercises CreateThreadedComment typed wrapper"
"$BIN" comment reply "$COMMENT_ID" "Threaded reply" > /dev/null 2>&1 \
  || fail "comment reply failed"
ok "comment reply succeeded"

# --- comment delete (DeleteComment) ---------------------------------------
step "comment delete — exercises DeleteComment"
"$BIN" comment delete "$COMMENT_ID" -y > /dev/null 2>&1 \
  || fail "comment delete failed"
ok "comment deleted"

# --- bulk task delete (zsh-quirk regression) ------------------------------
step "task delete — bulk + ExpandIDArgs regression"
ALL_IDS="$PARENT_ID ${SUBTASK_IDS[*]}"
# Pass as a single argument with embedded spaces (the original v0.34.0 zsh bug).
"$BIN" task delete "$ALL_IDS" -y > /dev/null 2>&1 \
  || fail "bulk task delete failed (ExpandIDArgs regression)"
ok "bulk delete succeeded for ${#SUBTASK_IDS[@]} subtask(s) + parent"

# Cleanup trap is now a no-op — the resources are gone.
PARENT_ID=""
SUBTASK_IDS=()

printf '\n✓ all smoke tests passed\n'
