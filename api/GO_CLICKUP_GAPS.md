# go-clickup Gap Analysis

Comparison of `github.com/raksul/go-clickup` (v0.0.0-20241002) against the
official ClickUp API V2 spec (82 paths, 137 operations).

## Missing Fields in Existing Types

| Type | Missing Field | Spec Field | Impact |
|------|--------------|------------|--------|
| `TaskUpdateRequest` | `points` | `points: float32` | **HIGH** — we use raw HTTP PUT to set sprint points |
| `TaskUpdateRequest` | `markdown_content` | `markdown_content: string` | **HIGH** — we use raw HTTP PUT for markdown descriptions |
| `TaskPriority` | nullable handling | `priority: nullable` | **MEDIUM** — go-clickup returns `TaskPriority{}` instead of nil when null |
| `Task` | `points` | `points: float32` | MEDIUM — field exists but may not deserialize correctly |

## Missing Operations (our raw HTTP workarounds)

| Operation | Spec Path | Used By |
|-----------|-----------|---------|
| Time entry CRUD | `POST/GET/DELETE /team/{id}/time_entries` | `pkg/cmd/task/time.go` |
| Comment replies | `GET/POST /comment/{id}/reply` | `pkg/cmd/comment/reply.go`, `list.go` |
| Space status read | `GET /space/{id}` (status extraction) | `pkg/cmdutil/status.go` |
| Space status update | `PUT /space/{id}` (statuses array) | `pkg/cmd/status/add.go` |
| Get current user | `GET /user` | `pkg/cmdutil/recent.go`, `pkg/cmd/inbox/inbox.go` |
| Task tag add/remove | `POST/DELETE /task/{id}/tag/{name}` | `pkg/cmd/task/helpers.go` |
| Space tag get/create | `GET/POST /space/{id}/tag` | `pkg/cmdutil/tags.go` |
| Add/remove task to list | `POST/DELETE /list/{id}/task/{id}` | `pkg/cmd/task/helpers.go` |
| Search tasks (filtered) | `GET /team/{id}/task` (raw) | `pkg/cmd/task/search.go` |

## Known Bugs

| Bug | Detail |
|-----|--------|
| Tag creation sends tags in request body | ClickUp ignores body tags; must use `POST /task/{id}/tag/{name}` URL |
| Nullable priority parsed as empty struct | `"priority": null` → `TaskPriority{Priority:"", Color:""}` instead of nil |
| `PUT /list/{id}` silently ignores `folder_id` | Endpoint returns 200 and echoes the original folder; the list does not move. Verified 2026-04-22 against a live workspace. `POST /folder/{id}/list` with `list_id` also rejects (`List Name Invalid`). **There is no public-API way to reparent a list between folders.** Do NOT add a `list move` command (nick-preda's fork did and shipped a non-functional no-op). |

## Operations in Spec NOT in go-clickup

- Goals (`/goal/*`, `/key_result/*`)
- Guests (`/team/{id}/guest`, per-resource guest endpoints)
- Roles (`/team/{id}/customroles`)
- Shared hierarchy (`/team/{id}/shared`)
- Templates (`*/taskTemplate`, `*/list_template`, `*/folder_template`)
- Views (`*/view`)
- Webhooks (`/team/{id}/webhook`)
- Bulk time in status (`/task/bulk_time_in_status/task_ids`)
- User management (`/team/{id}/user`)
- User groups (`/group`)

## Recommendation

Replace go-clickup incrementally with auto-generated client from the V2 spec.
Priority replacements:
1. `UpdateTask` (points + markdown_content) — immediate
2. Tag operations — immediate (fixes body vs URL bug)
3. Time entries — immediate (entirely missing)
4. Comment replies — immediate (entirely missing)
5. Remaining operations — incremental over follow-up PRs
