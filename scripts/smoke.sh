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
SMOKE_CACHE=""

cleanup() {
  set +e
  [ -n "$SMOKE_CACHE" ] && rm -rf "$SMOKE_CACHE"
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

# --- search: the API-shape assumptions unit tests cannot check ------------
#
# `task search` has no server-side filter to lean on — ClickUp accepts
# `search=` on team/{id}/task and ignores it — so every match is found by
# paginating and filtering locally. That makes the search correct only insofar
# as its assumptions about ordering and pagination are correct, and those are
# exactly the assumptions a mock will happily confirm while the real API
# disagrees. Both checks below were live bugs that a green unit suite missed.

WORKSPACE="$("$BIN" api user --jq '.user.id' --raw > /dev/null 2>&1 && \
  "$BIN" api team --jq '.teams[0].id' --raw 2>/dev/null | tail -1)"
[ -n "$WORKSPACE" ] || fail "could not resolve workspace id"

step "api ordering — order_by=updated must return newest first"
# The parent task was created seconds ago, so it must be on page 0 of a
# newest-first fetch. reverse=true flips this to ascending, which pointed the
# whole sweep at the oldest tasks in the workspace.
NEWEST_PAGE="$("$BIN" api "team/$WORKSPACE/task?include_closed=true&page=0&order_by=updated" \
  --jq '.tasks[].id' --raw 2>/dev/null)"
grep -q "^$PARENT_ID$" <<< "$NEWEST_PAGE" \
  || fail "freshly created task absent from page 0 of order_by=updated — ordering assumption broken"
ok "newest-first confirmed"

OLDEST_PAGE="$("$BIN" api "team/$WORKSPACE/task?include_closed=true&page=0&order_by=updated&reverse=true" \
  --jq '.tasks[].id' --raw 2>/dev/null)"
if grep -q "^$PARENT_ID$" <<< "$OLDEST_PAGE"; then
  printf '  ! reverse=true now returns newest-first too; the ordering comment in\n'
  printf '    fetchTeamTasks is stale and the sweep can drop reverse handling.\n'
else
  ok "reverse=true confirmed as ascending (hence not used)"
fi

step "api pagination — only an empty page proves the corpus is exhausted"
# A live page 0 comes back with 99 rows against a nominal size of 100. Treating
# "short page" as "end of corpus" stopped the sweep dead on page 0.
P0=$(wc -l <<< "$NEWEST_PAGE" | tr -d ' ')
P1=$("$BIN" api "team/$WORKSPACE/task?include_closed=true&page=1&order_by=updated" \
  --jq '.tasks | length' 2>/dev/null | tail -1)
if [ "$P0" -lt 100 ] && [ "$P1" -gt 0 ]; then
  ok "page 0 returned $P0 rows yet page 1 has $P1 — short page is not end-of-corpus"
elif [ "$P0" -eq 100 ]; then
  ok "page 0 returned a full 100 rows (short-page trap not reproducible here)"
else
  fail "page 0 returned $P0 rows and page 1 is empty — cannot verify the pagination assumption"
fi

# --- search end-to-end ----------------------------------------------------
step "task search --no-cache — live sweep finds a freshly created task"
"$BIN" task search "$TOKEN" --no-cache --exact 2>/dev/null | grep -q "$PARENT_ID" \
  || fail "live search did not find $PARENT_ID"
ok "live sweep found it"

step "task search (indexed) — builds an index, finds it, and persists"
SMOKE_CACHE="$(mktemp -d)"
# First run is a cold build over the whole workspace and is expected to be slow.
CLICKUP_CACHE_DIR="$SMOKE_CACHE" "$BIN" task search "$TOKEN" --exact 2>/dev/null \
  | grep -q "$PARENT_ID" || fail "indexed search did not find $PARENT_ID"
ls "$SMOKE_CACHE"/index-*.json > /dev/null 2>&1 || fail "no index written"
ok "index built and search matched"

step "task search (warm index) — second run stays correct"
CLICKUP_CACHE_DIR="$SMOKE_CACHE" "$BIN" task search "$TOKEN" --exact 2>/dev/null \
  | grep -q "$PARENT_ID" || fail "warm indexed search lost $PARENT_ID"
ok "warm index still matched"

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
