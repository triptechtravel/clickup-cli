# patch-v2-spec.jq — Fixes known type mismatches in the ClickUp V2 OpenAPI spec.
#
# The official spec declares several response fields with incorrect types:
#   - time_spent / time_estimate: declared as string|null, API returns integer (ms)
#   - assignees: declared as string[], API returns object[] with {id, username, email, ...}
#   - watchers: declared as string[], API returns object[] with same shape as assignees
#   - tags: declared as string[], API returns object[] with {name, tag_fg, tag_bg}
#   - group_assignees / checklists / dependencies / linked_tasks: declared as string[],
#     API returns object[]. Patches here widen the items to generic objects so
#     json.Unmarshal does not fail when they are populated.
#
# Usage: jq -f patch-v2-spec.jq clickup-v2.json > clickup-v2-patched.json
#
# Reported to ClickUp: https://feedback.clickup.com/public-api

# Helper: patch time_spent and time_estimate in a properties object
def fix_time_fields:
  if .time_spent then
    .time_spent = {"type": ["integer", "null"], "description": "Time spent in milliseconds"}
  else . end
  | if .time_estimate then
    .time_estimate = {"type": ["integer", "null"], "description": "Time estimate in milliseconds"}
  else . end;

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

# Walk all schema properties objects and apply field-level fixes.
(.. | objects | select(has("properties")) | .properties) |= (
  fix_time_fields
  | fix_assignees
  | fix_watchers
  | fix_tags
  | fix_group_assignees
  | fix_checklists
  | fix_dependencies
  | fix_linked_tasks
)
# Then patch comment request bodies. The /v2/comment/{comment_id}/reply POST
# endpoint is undocumented in the public spec, so the reply command stays
# hand-rolled (see pkg/cmd/comment/reply.go).
| (.paths."/v2/task/{task_id}/comment".post.requestBody.content."application/json".schema) |= fix_comment_request
| (.paths."/v2/comment/{comment_id}".put.requestBody.content."application/json".schema) |= fix_comment_request
# Fix create-comment response shapes — `id` is documented as string but
# returned as a JSON number.
| (.paths."/v2/task/{task_id}/comment".post.responses."200".content."application/json".schema) |= fix_comment_response
| (.paths."/v2/list/{list_id}/comment".post.responses."200".content."application/json".schema) |= fix_comment_response
| (.paths."/v2/view/{view_id}/comment".post.responses."200".content."application/json".schema) |= fix_comment_response
