# API layer: why `archive` was missing, and how to stop it recurring

Written 2026-07-27 after a real incident: cleaning out a ClickUp list, the CLI
reported the list clean when it was not, twice, and could not archive a task at
all. Both failures trace to the same structural cause, and that cause will keep
producing failures until the migration `api/GO_CLICKUP_GAPS.md` started is
finished.

## The incident

1. `clickup task archive <id>` printed help and exited 0. Nothing happened, and
   nothing said so. Archiving had to be done with raw `curl` against
   `PUT /api/v2/task/{id}`, which works first time.
2. `clickup task list --list-id <apps>` reported 2 tasks; the UI showed 8. The
   other 6 were **multi-list** — homed elsewhere, linked into this list. The CLI
   cannot see or report that relationship, so it reported a filtered view as a
   complete one.
3. Three subtasks of an archived parent survived a bulk operation, because
   `task list` excludes subtasks unless asked and archiving does not cascade.

Each is the CLI being confidently wrong rather than unhelpful. That is the
expensive kind: every one of these was acted on before it was caught.

## Root cause: two API layers, and commands use the thinner one

`make api-gen` generates a complete client from ClickUp's published specs. That
layer has `Archived` on ~20 types and `Locations` on 5 — everything the incident
needed.

Alongside it sits a hand-written layer: `internal/clickup/types.go` plus the
`*Local` helpers in `internal/apiv2/local.go`, inherited from the go-clickup
replacement. **Commands use the hand-written layer for exactly the operations
where it is incomplete.**

| | Generated | Hand-written |
| --- | --- | --- |
| Operations | 136 | 18 |
| Command files using it | 40 | 26 |
| `Archived` on task update | yes | **no** |
| `Locations` on task | yes | **no** |

`pkg/cmd/task/edit.go:179` builds a `clickup.TaskUpdateRequest` — 17 fields,
none of them `archived` — and sends it via `apiv2.UpdateTaskLocal`.

So archive was never *unwired*. It was never reachable. The generated client
knows about it; nothing on the path commands actually take does. Multi-list is
the same defect wearing different clothes: `GET /list/{id}/task` returns
home-list tasks only, and with no `Locations` on the hand-written `Task` there
is no way to report what is being withheld.

`GO_CLICKUP_GAPS.md` recommended replacing go-clickup incrementally with the
generated client. The generation half shipped; the adoption half stopped. The
seam left behind is where new API surface goes to die — and it will swallow the
next field too, silently, unless the last two phases below happen.

## Brittle parts, ranked by expected cost

### 1. The spec is unpinned (highest)

Generated code is deliberately not committed — a fine choice, but it makes the
**spec the source of truth**, and it is fetched at build time from a live URL
with no checksum and no version:

```make
SPEC_V2_URL := https://developer.clickup.com/openapi/clickup-api-v2-reference.json
```

Two clean clones a week apart can produce different clients with no signal.
ClickUp reshapes this spec — hence the 229-line `patch-v2-spec.jq` — and the
first symptom would be a compile error somewhere unrelated, or a silent
behaviour change.

### 2. `ensure-gen` tests the wrong file

```make
ensure-gen:
	@[ -f api/clickupv3/client.gen.go ] || $(MAKE) api-gen
```

Only v3 is checked. A missing or stale **v2** client is invisible, and v2 is
where nearly every task operation lives.

### 3. Post-generation fixes are regex over generated Go

`api/fix-codegen.sh` rewrites output with `perl -pi -e`; `make api-gen` applies
a further inline `perl -0pi` for the V3 `Parent` default. If upstream changes
shape these silently match nothing. The V3 fix exists in both places and can
drift.

### 4. `omitempty` on every mutable field

All 17 fields of `TaskUpdateRequest` are `omitempty`, so nothing can be
*cleared* — zero is indistinguishable from unset. Already worked around with
sentinels (`--points` → `pointsNotSet`, `--type` → `-1`). Any new boolean hits
the same wall: `--unarchive` would serialise to nothing.

### 5. Unknown subcommands exit 0

Group commands set no `Args` or `RunE`, so Cobra prints help and returns success
for anything unrecognised. For a tool this heavily scripted, silent success on a
typo'd verb is the worst available default.

### 6. List views hide two classes of task by default

`--include-subtasks` defaults false; multi-list membership is invisible
entirely. Individually defensible — together, a listing can omit tasks with no
indication anything was omitted.

---

# Strategy: generate the plumbing, hand-write the ergonomics

Before the phases, the design decision that shapes them. Almost all the
hand-coding in this repo is avoidable, but not all of it — and being precise
about the boundary is what stops this plan turning into a rewrite.

**Three mechanisms, in order of leverage:**

### 1. `clickup api` — a generic passthrough (~150 lines, covers everything forever)

What `gh api` is to `gh`. One command that takes a method, a path and fields:

```bash
clickup api -X PUT task/4n6u4x9 -f archived=true
clickup api list/216668663/task --paginate --jq '.tasks[].name'
```

This is the highest value-per-line change available. It covers 100% of current
**and future** API surface with zero per-endpoint work, and it never needs
updating when ClickUp ships something new. Had it existed, the archive incident
would have been a one-liner instead of a blocker plus a raw `curl` plus a
keychain excavation.

It also changes the risk profile of every other gap: with a passthrough, a
missing command is an inconvenience, not a dead end. **This should land in
Phase 1, not at the end.**

### 2. Refactor `gen-api` to emit the flag layer too

`cmd/gen-api` (816 lines) already parses the spec and emits typed operations.
It is the right place to also emit **flag binding** for every request field —
one pipeline, spec → gen-api → types + operations + flags, rather than a second
mechanism bolted alongside.

For each mutating operation, emit a `<Op>Flags` struct plus `Register(cmd)` and
`Apply(req)`:

- `Archived *bool` → `--archived` / `--no-archived`
- new spec field → `make api-gen` → **the flag exists, with no Go written**

Generated rather than reflected on purpose: the output is greppable, shows up in
diffs, is debuggable with a stack trace, and needs no runtime type magic. It
also means a spec change is *visible in review* as a diff in the generated flag
set, which is the whole point of Phase 5.

Two things to get right, once, centrally — rather than 17 times by hand:

- **Pointer semantics.** Brittleness #4 (`omitempty` making fields unclearable)
  dissolves: a `*bool` that is nil is unset, false is false. The sentinel hacks
  for `--points` and `--type` can go.
- **An override table** in `gen-api` for the fields needing a nicer name, custom
  help, or a value parser (dates, assignee add/remove, custom fields). Small,
  explicit, additive — the default is "just works", overrides are the exception.

### 3. Generated commands for the long tail

Same generator, one step further: emit whole Cobra commands for operations with
**no hand-written equivalent** — goals, guests, roles, shared hierarchy, user
groups, bulk time-in-status, everything in the `GO_CLICKUP_GAPS.md` "not in
go-clickup" list.

## Where generation must stop

Be honest about this or the plan overreaches. A good CLI is not a 1:1 mapping of
REST endpoints. Generated commands look like:

```
clickup put-v2-task-taskid --body '{"status":"in progress"}'
```

which is strictly worse than the passthrough for a human. Everything that makes
this tool worth using is *absent from the spec by definition*:

- fuzzy status matching (`review` → `code review`)
- task ID auto-detection from the git branch
- `--current` resolving the active sprint list
- bulk ID handling across every mutating command
- `@username` resolution and markdown comment rendering
- conventional task naming, tag reuse prompts

So the target split is:

| Layer | How it is written | Covers |
| --- | --- | --- |
| Types + operations | generated from spec | all 136 operations |
| Flags on mutating commands | reflection over generated types | every request field, automatically |
| ~20 ergonomic commands | hand-written | the daily-driver paths |
| Everything else | `clickup api` passthrough | the entire rest of the API, forever |

The hand-written surface shrinks to the ergonomics, which is the only part that
deserves human attention — and it is the part a spec can never supply.

---

# Plan

Six phases, each independently shippable. Tests first in every phase: write the
failing test, then the implementation.

## Phase 1 — stop lying

Small, high value, no API changes.

1. **Fail on unknown subcommands.** `Args: cobra.NoArgs` plus a `RunE` returning
   `cmdutil.FlagErrorf` on every group command (`task`, `list`, `folder`,
   `space`, `doc`, `comment`, `sprint`, `status`, `tag`, `view`, `goal`,
   `webhook`, `template`, `attachment`, `field`, `member`, `chat`, `link`,
   `auth`).
   *Test:* `clickup task bogus` exits non-zero and names the bad verb.
2. **Fix `ensure-gen`** to require both v2 and v3 clients.
3. **Pin the spec.** `SPEC_V2_SHA` / `SPEC_V3_SHA` in the Makefile, verified
   after download, failing with a clear message on mismatch. Add
   `make api-update` to re-pin deliberately. Reproducible input, generated code
   still out of git.

## Phase 2 — make membership visible

Fixes the incident directly.

4. **Add `Locations` to the task type** and surface it in `task view`.
5. **`task list --linked`** to include tasks linked in from other lists.
6. **Never present a filtered view as complete.** When subtasks or linked tasks
   are excluded, print a footer saying so with counts. This is the general fix;
   the flags are the specific one.
   *Test:* fixture task whose home list differs from the queried list appears
   under `--linked` and is counted in the footer without it.

## Phase 3 — archive support

7. **`clickup task archive <ids...>` and `task unarchive <ids...>`**,
   bulk-capable like `task edit`, via the generated `UpdateTask` operation
   rather than extending the hand-written request.
8. If it must route through `TaskUpdateRequest`, make `Archived` a `*bool` —
   do not add a third sentinel.
9. **No cascade by default.** Archiving a parent does not archive its subtasks
   (confirmed against a live workspace). Archiving is destructive-adjacent, so
   the safe default is to touch only what was named:
   - default: archive the named tasks only, and **report orphaned subtasks
     loudly** — `! 3 subtasks of 86abc123 are still active (use --cascade)` —
     with their IDs, on stderr, non-suppressible.
   - `--cascade`: archive named tasks and all descendants, reporting the total.
   - The same rule applies to `unarchive`.
   *Test:* parent with subtasks; assert default leaves subtasks untouched and
   the warning names every orphan; assert `--cascade` archives all.

## Phase 4 — the adoption half (revised after implementing it)

**The original plan here was wrong, and implementing it proved so.** It called
for porting all 18 `*Local` helpers and every `clickup.Task` reference to
generated types. Two findings killed that:

1. **The `*Local` helpers are the ergonomic layer, not plumbing.**
   `GetTasksLocal` returns `[]clickup.Task`. The generated equivalent returns
   `[]GetV2ListListIDTask200ResponseJSON2` with pointers throughout. Porting
   would make every call site worse.
2. **`Nullable[T]` is `map[bool]T`.** Porting `edit.go`'s date and
   time-estimate handling onto the generated request would replace a readable
   `clickup.NullDate()` with map juggling — losing exactly the null-versus-
   absent clarity a hand-rolled CLI exists to provide.

The incident was never caused by hand-written types existing. It was caused by
them **drifting**: `archived` was in the spec and absent from the struct. So the
fix is a field-level guard, not a type-level ban —
`internal/clickup.TestSpecDrift` compares every hand-written request and
response type against its generated counterpart and fails the build on a
missing field. It found six gaps immediately, including the two
(`markdown_content`, `points`) this document had already recorded as separate
raw-HTTP workarounds.

**What was actually cleaned up:** six genuinely dead types deleted
(`TaskAttachementOptions`, `GetTaskOptions`, `GetTasksOptions`,
`DeleteDependencyOptions`, `ChecklistRequest`, `ChecklistItemRequest`), plus
`GetUserLocal` which had no callers. Everything remaining is either the
ergonomic layer or covered by the drift guard.

### The original port plan, kept for reference

Finish the migration `GO_CLICKUP_GAPS.md` started. One command group per PR,
deleting each hand-written type as its last caller goes. Sequenced by blast
radius, smallest first.

**Inventory — 18 `*Local` helpers:**

| Helper | Callers | Command groups |
| --- | --- | --- |
| `GetUserLocal` | **0** | *dead code — delete first* |
| `UpdateTaskLocal` | 1 | task |
| `CreateTaskLocal` | 1 | task |
| `DeleteTaskLocal` | 1 | task |
| `AddDependencyLocal` | 1 | task |
| `DeleteDependencyLocal` | 1 | task |
| `RemoveCustomFieldValueLocal` | 1 | task |
| `GetAccessibleCustomFieldsLocal` | 2 | field, task |
| `GetFilteredTeamTasksLocal` | 2 | inbox, cmdutil |
| `GetListLocal` | 2 | cmdutil, task |
| `SetCustomFieldValueLocal` | 2 | task |
| `GetSpacesLocal` | 3 | space, task |
| `GetTasksLocal` | 3 | sprint, task |
| `GetFolderlessListsLocal` | 4 | list, task |
| `GetTeamsLocal` | 4 | auth, comment, member, task |
| `GetFoldersLocal` | 6 | folder, sprint, task |
| `GetListsLocal` | 7 | list, cmdutil, sprint, task |
| `GetTaskLocal` | 8 | link, status, task |

**Order of work:**

1. Delete `GetUserLocal` (no callers).
2. **Write path first** — `UpdateTaskLocal`, `CreateTaskLocal`,
   `DeleteTaskLocal`. One caller each, and `UpdateTaskLocal` is what blocks
   archive, so Phase 3 can land on the generated operation directly rather than
   being ported later.
3. Single-caller reads: dependencies, custom fields.
4. Ascending caller count: `GetListLocal` → `GetSpacesLocal` → `GetTasksLocal`
   → `GetFolderlessListsLocal` → `GetTeamsLocal` → `GetFoldersLocal` →
   `GetListsLocal` → `GetTaskLocal`.
5. **`clickup.Task` last.** 40 references across `pkg/` — the bulk of the work.
   Bridge with a type alias to the generated task during the transition so the
   port can be split across PRs rather than landing as one 40-file diff.

**Types to delete once their callers are gone** (current `pkg/` reference
counts): `TaskUpdateRequest` (1), `TaskAssigneeUpdateRequest` (1),
`AddDependencyRequest` (1), `TaskRequest` (2), `Space` (1), `Folder` (2),
`User` (4), `TeamUser` (5), `List` (11), `Task` (40).

6. Consolidate the duplicated V3 perl fix into `fix-codegen.sh`; have
   `make api-gen` call the script rather than inlining a copy.

## Phase 5 — stop new API surface from going missing

Phase 4 removes today's gap. This phase stops tomorrow's. Two failure modes to
close, and they need different mechanisms.

**A. New surface must arrive.** Nothing currently notices when ClickUp publishes
new fields or operations — with a pinned spec (Phase 1.3), nothing ever would.

10. **Nightly spec-drift job.** Fetch both specs, diff against the pinned SHAs,
    and on change open a PR that re-pins, regenerates, and shows the spec diff
    in the description. Upstream changes become reviewable events instead of
    silent ones.

**B. New surface must be reachable.** The hand-written layer is what turned a
present field into an absent feature. Once Phase 4 empties it, keep it empty.

11. **Guard test:** no file under `pkg/` may reference a hand-written request
    type. Fails CI the moment a new command reintroduces the pattern. Cheap,
    and it is the specific regression that caused this.
12. **Operation coverage report.** `cmd/gen-api` already parses the spec —
    extend it to emit spec-operations vs. wired-commands, and fail CI when a
    command hand-rolls an operation the generated layer already provides. Also
    useful as a roadmap: it names every capability the API has and the CLI does
    not.

## Not doing

- **Committing generated code.** Deliberate project choice; Phase 1.3 fixes
  reproducibility at the input instead.
- **A `list move` command.** `GO_CLICKUP_GAPS.md` documents that the public API
  cannot reparent a list, verified against a live workspace. A previous fork
  shipped one as a non-functional no-op. Leave it alone.
