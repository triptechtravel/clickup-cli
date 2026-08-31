# patch-v2-spec.jq — Fixes known type mismatches in the ClickUp V2 OpenAPI spec.
#
# The official spec declares several response fields with incorrect types:
#   - time_spent / time_estimate / duration: declared as string|null (or plain
#     integer), but the API is inconsistent about which it sends — the same
#     response can carry a number on one task and a string on the next. Patched
#     to the clickup.Millis Go type, which accepts both.
#   - assignees: declared as string[], API returns object[] with {id, username, email, ...}
#   - watchers: declared as string[], API returns object[] with same shape as assignees
#   - tags: declared as string[], API returns object[] with {name, tag_fg, tag_bg}
#   - group_assignees / checklists / dependencies / linked_tasks: declared as string[],
#     API returns object[]. Patches here widen the items to generic objects so
#     json.Unmarshal does not fail when they are populated.
#   - checklist-item response `assignee` (singular): declared as string|null, but
#     the API returns a full user object on assigned items. Patched (response
#     schemas only — request bodies still send a scalar assignee id) so the
#     checklist item edit/resolve responses decode.
#
# Usage: jq -f patch-v2-spec.jq clickup-v2.json > clickup-v2-patched.json
#
# Reported to ClickUp: https://feedback.clickup.com/public-api

# Helper: the Go type every millisecond field is generated as.
#
# The spec's declared type is not enough on its own: ClickUp returns these
# fields as a JSON number on one task and a JSON string on the next — in the
# same response — so any single Go scalar aborts the decode on the other form.
# clickup.Millis accepts both and marshals back as a number.
# x-omitempty is not cosmetic: clickup.Millis is an int64, and `omitempty` on an
# integer kind drops a zero before MarshalJSON is ever consulted. Without it a
# task with no tracked time loses time_spent from --json entirely — the
# generated Nullable[int] this replaced kept it.
def millis(desc):
  {
    "type": ["integer", "null"],
    "description": desc,
    "x-go-type": "clickup.Millis",
    "x-go-type-import": {
      "name": "clickup",
      "path": "github.com/triptechtravel/clickup-cli/internal/clickup"
    },
    "x-omitempty": false
  };

# Helper: patch time_spent and time_estimate in a properties object
def fix_time_fields:
  if .time_spent then
    .time_spent = millis("Time spent in milliseconds")
  else . end
  | if .time_estimate then
    .time_estimate = millis("Time estimate in milliseconds")
  else . end;

# Helper: patch a time entry's own fields — its duration, documented as an
# integer but returned as a string by the time-entries endpoints (a running
# timer reports -1), and its timestamps, where the spec's own stop-timer example
# shows a string start beside a numeric end while the running-timer example
# makes all three strings.
#
# Applied to the time-entry endpoints only (below). `duration`, `start` and
# `end` are generic enough to mean other things elsewhere in the spec — a media
# length in seconds, a retry window — and a field silently retyped as
# milliseconds is worse than one left alone: it decodes clean and is wrong by a
# factor of 1000.
def fix_time_entry_fields:
  reduce ("duration", "start", "end", "at") as $f (.;
    if .[$f] then .[$f] = millis("Milliseconds") else . end
  );

# Helper: patch assignees from string[] to object[]
def fix_assignees:
  if .assignees.items.type == "string" then
    .assignees.items = {
      "type": "object",
      "properties": {
        "id": {"type": "integer"},
        "username": {"type": "string"},
        "email": {"type": "string"},
        "color": {"type": ["string", "null"]},
        "initials": {"type": ["string", "null"]},
        "profilePicture": {"type": ["string", "null"]}
      }
    }
  else . end;

# Helper: patch watchers from string[] to object[] (same shape as assignees)
def fix_watchers:
  if .watchers.items.type == "string" then
    .watchers.items = {
      "type": "object",
      "properties": {
        "id": {"type": "integer"},
        "username": {"type": "string"},
        "email": {"type": "string"},
        "color": {"type": ["string", "null"]},
        "initials": {"type": ["string", "null"]},
        "profilePicture": {"type": ["string", "null"]}
      }
    }
  else . end;

# Helper: patch tags from string[] to object[]
def fix_tags:
  if .tags.items.type == "string" then
    .tags.items = {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "tag_fg": {"type": "string"},
        "tag_bg": {"type": "string"}
      }
    }
  else . end;

# Helper: widen group_assignees items to generic object (API returns group objects)
def fix_group_assignees:
  if .group_assignees.items.type == "string" then
    .group_assignees.items = {"type": "object"}
  else . end;

# Helper: widen checklists items to generic object (API returns checklist objects)
def fix_checklists:
  if .checklists.items.type == "string" then
    .checklists.items = {"type": "object"}
  else . end;

# Helper: widen dependencies items to generic object (API returns dependency objects)
def fix_dependencies:
  if .dependencies.items.type == "string" then
    .dependencies.items = {"type": "object"}
  else . end;

# Helper: widen linked_tasks items to generic object (API returns linked-task objects)
def fix_linked_tasks:
  if .linked_tasks.items.type == "string" then
    .linked_tasks.items = {"type": "object"}
  else . end;

# Helper: widen the singular checklist-item `assignee` (response) to a generic
# object. Applied only to response schemas via path (below) so request bodies,
# which send a scalar assignee id, keep their string type. The API returns a
# user object on assigned items, which otherwise breaks json.Unmarshal.
def fix_checklist_item_assignee:
  if .properties.checklist.properties.items.items.properties.assignee then
    .properties.checklist.properties.items.items.properties.assignee = {"type": "object"}
  else . end;

# Helper: fix the create-comment response schema. The spec declares `id` as
# string, but the API returns a JSON number (large enough to require int64).
# Reported on every comment create endpoint (task/list/view).
def fix_comment_response:
  if .properties and .properties.id and (.properties.id.type == "string") then
    .properties.id = {"type": "integer", "contentEncoding": "int64"}
  else . end;

# Helper: extend a comment request schema with structured `comment` blocks
# (Quill-delta format used by ClickUp's web app for rich formatting and
# @mentions) and `markdown_text`. Also relax `required` so callers can send
# any one of comment / comment_text / markdown_text and need not supply
# assignee/resolved on partial updates.
def fix_comment_request:
  if .properties and .properties.comment_text then
    .properties += {
      "comment": {
        "type": "array",
        "description": "Structured Quill-delta comment blocks (rich formatting + @mentions).",
        "items": {
          "type": "object",
          "properties": {
            "text": {"type": "string"},
            "type": {"type": "string"},
            "user": {"type": "object", "properties": {"id": {"type": "integer"}}},
            "attributes": {"type": "object", "additionalProperties": true}
          }
        }
      },
      "markdown_text": {
        "type": "string",
        "description": "Markdown body — alternative to comment_text/comment."
      }
    }
    | (if .required then .required = (.required - ["comment_text", "assignee", "resolved"]) else . end)
  else . end;

# Millisecond fields are patched in response schemas (and shared components)
# only. Request bodies keep their plain integer type: the CLI is the side
# sending them, so it always sends a number, and a plain scalar is what the
# generated flag sets need in order to bind --duration and friends.
def fix_millis_fields:
  (.. | objects | select(has("properties")) | .properties) |= fix_time_fields;

def fix_time_entry_props:
  (.. | objects | select(has("properties")) | .properties) |= fix_time_entry_fields;

(.paths[]?[]? | objects | select(has("responses")) | .responses) |= fix_millis_fields
# Shared schemas and responses, but not components.requestBodies — see above.
| (if (.components | type) == "object" then
     .components |= (
       (if has("schemas") then .schemas |= fix_millis_fields else . end)
       | (if has("responses") then .responses |= fix_millis_fields else . end)
     )
   else . end)
# Time-entry duration and timestamps, scoped to those endpoints by path.
| (if (.paths | type) == "object" then .paths |= with_entries(
     if (.key | test("time_entries|/time$|/time/")) then
       .value |= ((.[]? | objects | select(has("responses")) | .responses) |= fix_time_entry_props)
     else . end
   ) else . end)

# Walk all schema properties objects and apply the remaining field-level fixes.
| (.. | objects | select(has("properties")) | .properties) |= (
  fix_assignees
  | fix_watchers
  | fix_tags
  | fix_group_assignees
  | fix_checklists
  | fix_dependencies
  | fix_linked_tasks
)
# Add the undocumented POST /v2/comment/{comment_id}/reply endpoint so the
# reply command can use a generated wrapper. The schema is inlined (rather
# than $ref'd to the task-comment schema) because oapi-codegen-exp doesn't
# emit a fresh type for path-style refs, which gen-api then can't resolve.
| .paths."/v2/comment/{comment_id}/reply".post = {
    "summary": "Create Threaded Comment",
    "description": "Reply to an existing comment, creating a threaded reply.",
    "tags": ["Comments"],
    "operationId": "CreateThreadedComment",
    "parameters": [
      {
        "name": "comment_id",
        "in": "path",
        "required": true,
        "style": "simple",
        "schema": {"type": "string"}
      }
    ],
    "requestBody": {
      "required": true,
      "content": {
        "application/json": {
          "schema": {
            "type": "object",
            "properties": {
              "comment_text": {"type": "string"},
              "notify_all": {"type": "boolean"},
              "comment": {
                "type": "array",
                "description": "Structured Quill-delta comment blocks (rich formatting + @mentions).",
                "items": {
                  "type": "object",
                  "properties": {
                    "text": {"type": "string"},
                    "type": {"type": "string"},
                    "user": {"type": "object", "properties": {"id": {"type": "integer"}}},
                    "attributes": {"type": "object", "additionalProperties": true}
                  }
                }
              },
              "markdown_text": {"type": "string"}
            }
          }
        }
      }
    },
    "responses": {
      "200": {
        "description": "",
        "content": {
          "application/json": {
            "schema": {
              "type": "object",
              "properties": {
                "id": {"type": "integer", "contentEncoding": "int64"},
                "hist_id": {"type": "string"},
                "date": {"type": "integer", "contentEncoding": "int64"}
              }
            }
          }
        }
      }
    }
  }
| (.paths."/v2/task/{task_id}/comment".post.requestBody.content."application/json".schema) |= fix_comment_request
| (.paths."/v2/comment/{comment_id}".put.requestBody.content."application/json".schema) |= fix_comment_request
# Fix create-comment response shapes — `id` is documented as string but
# returned as a JSON number.
| (.paths."/v2/task/{task_id}/comment".post.responses."200".content."application/json".schema) |= fix_comment_response
| (.paths."/v2/list/{list_id}/comment".post.responses."200".content."application/json".schema) |= fix_comment_response
| (.paths."/v2/view/{view_id}/comment".post.responses."200".content."application/json".schema) |= fix_comment_response
# Fix checklist-item response `assignee` — documented as string but returned as
# a user object on assigned items (breaks edit/resolve response decoding).
| (.paths."/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}".put.responses."200".content."application/json".schema) |= fix_checklist_item_assignee
| (.paths."/v2/checklist/{checklist_id}/checklist_item".post.responses."200".content."application/json".schema) |= fix_checklist_item_assignee
