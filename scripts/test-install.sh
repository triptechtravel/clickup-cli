#!/usr/bin/env bash
# scripts/test-install.sh — exercise scripts/install.sh across environments.
#
# The installer is POSIX sh and has to survive shells and toolchains we don't
# control, so the cases below deliberately vary the pieces most likely to
# break it: busybox ash vs bash, wget vs curl, root vs unprivileged, and a
# hermetic server that serves a deliberately corrupted tarball.
#
# Usage:
#   make test-install              # all cases
#   ./scripts/test-install.sh -k alpine   # only cases matching "alpine"
#
# Requires Docker. Network cases hit the real GitHub release; the checksum
# cases are hermetic (local HTTP server inside the container).

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INSTALLER="${ROOT}/scripts/install.sh"
FILTER="${2:-}"
PASS=0
FAIL=0
FAILED_CASES=()

[ -f "$INSTALLER" ] || { echo "missing $INSTALLER" >&2; exit 1; }
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }

c_pass=$'\033[32m'; c_fail=$'\033[31m'; c_dim=$'\033[2m'; c_off=$'\033[0m'
[ -t 1 ] || { c_pass=""; c_fail=""; c_dim=""; c_off=""; }

# run_case <name> <image> <script> [platform]
# The script runs inside the container with /install.sh mounted read-only.
# Pass a platform (e.g. linux/amd64) for images without a native manifest.
run_case() {
	local name="$1" image="$2" script="$3" platform="${4:-}"

	if [ -n "$FILTER" ] && [[ "$name" != *"$FILTER"* ]]; then
		return
	fi

	local plat_args=()
	[ -n "$platform" ] && plat_args=(--platform "$platform")

	printf '%-38s' "$name"
	local out
	if out=$(docker run --rm "${plat_args[@]}" \
		-v "${INSTALLER}:/install.sh:ro" \
		"$image" sh -c "$script" 2>&1); then
		printf '%sPASS%s\n' "$c_pass" "$c_off"
		PASS=$((PASS + 1))
	else
		printf '%sFAIL%s\n' "$c_fail" "$c_off"
		FAIL=$((FAIL + 1))
		FAILED_CASES+=("$name")
		printf '%s%s%s\n' "$c_dim" "$(echo "$out" | sed 's/^/    │ /' | tail -25)" "$c_off"
	fi
}

echo "Testing scripts/install.sh"
echo

# ── shell / downloader matrix ─────────────────────────────────────────
# alpine: busybox ash, busybox wget, no bash, no curl, no sudo, root.
run_case "alpine (busybox ash + wget)" alpine:latest '
	set -e
	sh /install.sh
	clickup version
'

run_case "debian (bash + curl)" debian:stable-slim '
	set -e
	apt-get -qq update && apt-get -qq install -y curl ca-certificates >/dev/null
	sh /install.sh
	clickup version
'

# Proves the wget branch works on a non-busybox wget too.
run_case "debian (wget only, no curl)" debian:stable-slim '
	set -e
	apt-get -qq update && apt-get -qq install -y wget ca-certificates >/dev/null
	sh /install.sh
	clickup version
'

# Arch publishes x86_64 only, so this doubles as our amd64 coverage
# (emulated when the host is arm64).
run_case "archlinux (curl, amd64)" archlinux:latest '
	set -e
	sh /install.sh
	clickup version
' linux/amd64

# ── privilege handling ────────────────────────────────────────────────
# Unprivileged, no sudo: must fall back to ~/.local/bin rather than fail.
run_case "non-root, no sudo -> ~/.local/bin" debian:stable-slim '
	set -e
	apt-get -qq update && apt-get -qq install -y curl ca-certificates >/dev/null
	useradd -m tester
	su tester -c "sh /install.sh"
	test -x /home/tester/.local/bin/clickup
	/home/tester/.local/bin/clickup version
'

run_case "CLICKUP_INSTALL_DIR override" alpine:latest '
	set -e
	CLICKUP_INSTALL_DIR=/opt/bin sh /install.sh
	test -x /opt/bin/clickup
	/opt/bin/clickup version
'

# ── version pinning ───────────────────────────────────────────────────
run_case "CLICKUP_VERSION pinned tag" alpine:latest '
	set -e
	CLICKUP_VERSION=v0.35.0 sh /install.sh
	clickup version | grep -q 0.35.0
'

# ── checksum verification (hermetic) ──────────────────────────────────
# Build a fake release locally and serve it, so we can control the digest.
# Case A: digest matches -> install succeeds.
# Case B: digest is wrong -> install MUST abort and leave nothing behind.
FAKE_RELEASE='
	set -e
	mkdir -p /srv/latest/download && cd /tmp
	printf "#!/bin/sh\necho clickup version 9.9.9\n" > clickup
	chmod +x clickup
	tar czf /srv/latest/download/clickup_linux_$ARCH.tar.gz clickup
	cd /srv && (python3 -m http.server 8000 >/dev/null 2>&1 &)
	sleep 1
'

run_case "checksum valid (hermetic)" python:3-alpine "
	ARCH=\$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
	$FAKE_RELEASE
	sha256sum /srv/latest/download/clickup_linux_\$ARCH.tar.gz \
		| sed 's|/srv/latest/download/||' > /srv/latest/download/checksums.txt
	CLICKUP_BASE_URL=http://127.0.0.1:8000 sh /install.sh
	clickup version | grep -q 9.9.9
"

run_case "checksum mismatch aborts" python:3-alpine "
	ARCH=\$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
	$FAKE_RELEASE
	echo \"$(printf '0%.0s' {1..64})  clickup_linux_\$ARCH.tar.gz\" > /srv/latest/download/checksums.txt
	if CLICKUP_BASE_URL=http://127.0.0.1:8000 sh /install.sh 2>/tmp/err; then
		echo 'BUG: installer succeeded despite bad checksum'; exit 1
	fi
	grep -q 'checksum mismatch' /tmp/err
	if command -v clickup >/dev/null 2>&1; then
		echo 'BUG: binary installed despite bad checksum'; exit 1
	fi
"

# ── static analysis ───────────────────────────────────────────────────
run_case "shellcheck (POSIX sh)" koalaman/shellcheck-alpine:stable '
	shellcheck -s sh /install.sh
'

echo
if [ "$FAIL" -eq 0 ]; then
	printf '%s%d passed%s\n' "$c_pass" "$PASS" "$c_off"
else
	printf '%s%d passed, %d failed%s\n' "$c_fail" "$PASS" "$FAIL" "$c_off"
	printf '  failed: %s\n' "${FAILED_CASES[*]}"
	exit 1
fi
