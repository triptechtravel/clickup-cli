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
TIMER_STARTED=""
TIME_ENTRY_IDS=()

cleanup() {
  set +e
  [ -n "$SMOKE_CACHE" ] && rm -rf "$SMOKE_CACHE"
  # A timer is server-side state on a real account: if the run aborts between
  # start and stop — which is exactly what the timer assertions exist to catch —
  # it would otherwise keep accruing against a task this trap is about to
  # delete. Same for the logged entry, which outlives its task.
  if [ -n "$TIMER_STARTED" ]; then
    "$BIN" task time stop > /dev/null 2>&1 || true
  fi
  for entry in "${TIME_ENTRY_IDS[@]+"${TIME_ENTRY_IDS[@]}"}"; do
    "$BIN" task time delete "$entry" > /dev/null 2>&1 || true
  done
  if [ ${#SUBTASK_IDS[@]} -gt 0 ]; then
    "$BIN" task delete "${SUBTASK_IDS[@]}" -y > /dev/null 2>&1 || true
  fi
  if [ -n "$PARENT_ID" ]; then
    "$BIN" task delete "$PARENT_ID" -y > /dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

# search_finds <expected-id> <query> [extra args...]
#
# Runs a search and asserts the id appears in stdout. Deliberately captures to a
# file rather than piping into `grep -q`: grep exits on its first match, the CLI
# takes SIGPIPE mid-write, and under `set -o pipefail` that surfaces as a failed
# assertion at random. It also cost a real debugging session, so it is written
# down here rather than rediscovered.
search_finds() {
  local expected="$1" query="$2"; shift 2
  local out err rc
  out="$(mktemp)"; err="$(mktemp)"
  set +e
  "$BIN" task search "$query" "$@" > "$out" 2> "$err"
  rc=$?
  set -e
  if ! grep -q "$expected" "$out"; then
    printf '  search exited %s\n  stdout:\n' "$rc" >&2
    sed 's/^/    /' "$out" >&2
    printf '  stderr:\n' >&2
    sed 's/^/    /' "$err" >&2
    rm -f "$out" "$err"
    return 1
  fi
  rm -f "$out" "$err"
}

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

# --- time tracking (issue #27) --------------------------------------------
# ClickUp returns time_spent as a number until time is tracked and as a string
# afterwards, in the same response. Logging time on the subtask and then reading
# the parent is the exact shape that used to make the parent undecodable — and
# only the real API produces the mixed types.
step "task time log — track time on the subtask"
set +e
LOG_OUT=$("$BIN" task time log "$SUB_ID" --duration 34m --description "Smoke probe $TOKEN" 2>&1)
rc=$?
set -e
[ $rc -eq 0 ] || fail "task time log failed: $LOG_OUT"
# `task time log` has no --json; it prints the new entry id as "(entry <id>)".
# Capturing it lets cleanup() delete the entry, which outlives its task.
ENTRY_ID=$(printf '%s' "$LOG_OUT" | sed -n 's/.*(entry \([0-9]*\)).*/\1/p' | head -1)
[ -n "$ENTRY_ID" ] && TIME_ENTRY_IDS+=("$ENTRY_ID")
ok "logged 34m on $SUB_ID (entry ${ENTRY_ID:-unknown})"

# set +e around the capture: under `set -e` a failing decode — the very thing
# these steps exist to catch — would abort the script before its fail() message.
step "task time list — duration decodes whichever type the API sends"
set +e
LOGGED=$("$BIN" task time list "$SUB_ID" --json --jq 'length' 2>&1 | tail -1)
set -e
[ "${LOGGED:-0}" -ge 1 ] 2>/dev/null || fail "task time list returned no entries for $SUB_ID: $LOGGED"
ok "read $LOGGED time entry/entries"

step "task view parent — a tracked subtask must not break the parent decode"
set +e
SPENT=$("$BIN" task view "$PARENT_ID" --json --jq '.subtasks | length' --raw 2>&1 | tail -1)
rc=$?
set -e
[ $rc -eq 0 ] || fail "task view failed on a parent whose subtask has tracked time (issue #27): $SPENT"
# An assertion, not an observation: the subtask fetch swallows its own error, so
# a decode regression there returns an empty list and exit 0. A run that cannot
# see the tracked subtask has not tested anything.
[ "${SPENT:-0}" -ge 1 ] 2>/dev/null \
  || fail "task view returned $SPENT subtasks for a parent known to have one (issue #27)"
ok "parent decoded with $SPENT subtask(s)"

step "task time start/running/stop — the timer round trip"
# ClickUp allows one running timer per user, and none of these commands take an
# id — they act on whatever the account has running. Displacing someone's real
# tracking to run a smoke test is not a trade this script gets to make, so it
# skips the round trip rather than clobbering it.
set +e
ALREADY=$("$BIN" task time running --json --jq '.data.id' --raw 2>/dev/null | tail -1)
set -e
if [ -n "$ALREADY" ] && [ "$ALREADY" != "null" ]; then
  ok "skipped: a timer ($ALREADY) is already running on this account"
else
  # The stop response is the one that must not fail: the timer is already
  # stopped server-side by the time the CLI decodes it, so an error here leaves
  # the user retrying against a timer that is no longer running.
  set +e
  OUT=$("$BIN" task time start "$SUB_ID" --description "Smoke timer $TOKEN" 2>&1)
  rc=$?
  set -e
  [ $rc -eq 0 ] || fail "task time start failed: $OUT"
  TIMER_STARTED=1

  set +e
  OUT=$("$BIN" task time running --json --jq '.data.id' --raw 2>&1 | tail -1)
  rc=$?
  set -e
  [ $rc -eq 0 ] || fail "task time running failed: $OUT"
  [ -n "$OUT" ] && [ "$OUT" != "null" ] || fail "task time running reported no timer moments after starting one"

  set +e
  OUT=$("$BIN" task time stop 2>&1)
  rc=$?
  set -e
  [ $rc -eq 0 ] || fail "task time stop failed to decode its own response: $OUT"
  TIMER_STARTED=""
  ok "timer started, read and stopped"
fi

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

# stderr kept and the exit status checked: with 2>/dev/null and a bare grep, an
# API error made this probe *pass*, reporting the assumption as confirmed when
# the call had failed outright.
if ! OLDEST_PAGE="$("$BIN" api "team/$WORKSPACE/task?include_closed=true&page=0&order_by=updated&reverse=true" \
  --jq '.tasks[].id' --raw)"; then
  fail "could not fetch the reverse=true page; the ordering assumption is unverified"
fi
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
# An assertion, not an observation. The sweep's correctness rests on "only an
# empty page ends the corpus", so a run that cannot demonstrate it must say so
# rather than printing a tick.
[ "$P1" -gt 0 ] || fail "page 1 is empty on a workspace with >100 tasks; cannot verify the pagination assumption"
if [ "$P0" -lt 100 ]; then
  ok "page 0 returned $P0 rows yet page 1 has $P1 — short page is not end-of-corpus"
else
  ok "page 0 was a full 100 rows and page 1 has $P1 — pagination continues past a full page"
fi

# --- search end-to-end ----------------------------------------------------
step "task search --no-cache — live sweep finds a freshly created task"
search_finds "$PARENT_ID" "$TOKEN" --no-cache --exact \
  || fail "live search did not find $PARENT_ID"
ok "live sweep found it"

step "task search (indexed) — builds an index, finds it, and persists"
SMOKE_CACHE="$(mktemp -d)"
# First run is a cold build over the whole workspace and is expected to be slow.
CLICKUP_CACHE_DIR="$SMOKE_CACHE" search_finds "$PARENT_ID" "$TOKEN" --exact \
  || fail "indexed search did not find $PARENT_ID"
ls "$SMOKE_CACHE"/index-*.json > /dev/null 2>&1 || fail "no index written"
ok "index built and search matched"

step "task search (warm index) — second run stays correct"
CLICKUP_CACHE_DIR="$SMOKE_CACHE" search_finds "$PARENT_ID" "$TOKEN" --exact \
  || fail "warm indexed search lost $PARENT_ID"
ok "warm index still matched"

# --- bulk task delete (zsh-quirk regression) ------------------------------
step "task delete — bulk + ExpandIDArgs regression"
ALL_IDS="$PARENT_ID ${SUBTASK_IDS[*]}"
# Pass as a single argument with embedded spaces (the original v0.34.0 zsh bug).
"$BIN" task delete "$ALL_IDS" -y > /dev/null 2>&1 \
  || fail "bulk task delete failed (ExpandIDArgs regression)"
ok "bulk delete succeeded for ${#SUBTASK_IDS[@]} subtask(s) + parent"

# --- search after delete --------------------------------------------------
step "task search --refresh — a deleted task leaves the index"
# Incremental syncs cannot see deletions, so without --refresh a deleted task
# lingers until the weekly reconcile. This asserts the lever works; the branch
# claimed the weekly rebuild handled it, which a capped rebuild does not.
if CLICKUP_CACHE_DIR="$SMOKE_CACHE" search_finds "$PARENT_ID" "$TOKEN" --refresh --exact 2>/dev/null; then
  fail "deleted task $PARENT_ID still returned after --refresh"
fi
ok "deleted task gone after --refresh"

# Cleanup trap is now a no-op — the resources are gone.
PARENT_ID=""
SUBTASK_IDS=()

printf '\n✓ all smoke tests passed\n'
