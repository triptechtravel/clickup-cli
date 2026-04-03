#!/usr/bin/env bash
# fix-codegen.sh — Post-generation fixes for oapi-codegen-exp output.
#
# The ClickUp specs have patterns the experimental codegen can't
# resolve cleanly. This script applies mechanical fixes.
set -euo pipefail

V2="${1:-clickupv2/client.gen.go}"
V3="${2:-clickupv3/client.gen.go}"

# ── V2 fixes ──────────────────────────────────────────────────────
if [ -f "$V2" ]; then
  # Recursive type aliases: `type Foo = []Foo` → `type Foo = []any`
  perl -pi -e 's/^type (\w+) = \[\]\1$/type $1 = []any/' "$V2"

  # Null enum constants: `FooNull SomeType = null` → `= 0`
  perl -pi -e 's/(\w+) = null$/\1 = 0/' "$V2"

  echo "Fixed $V2"
fi

# ── V3 fixes ──────────────────────────────────────────────────────
if [ -f "$V3" ]; then
  # Bad default: assigns string literal to struct pointer
  perl -0pi -e 's/if s\.Parent == nil \{\n\t\tv := "null"\n\t\ts\.Parent = &v\n\t\}//' "$V3"

  echo "Fixed $V3"
fi
