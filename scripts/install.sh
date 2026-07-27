#!/bin/sh
# scripts/install.sh — one-line installer for the `clickup` CLI.
#
# Detects OS/arch, downloads the matching release tarball from GitHub,
# verifies its SHA-256 against checksums.txt, and installs the binary.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/triptechtravel/clickup-cli/main/scripts/install.sh | sh
#
# Environment overrides:
#   CLICKUP_VERSION      Tag to install (e.g. v0.35.1). Default: latest.
#   CLICKUP_INSTALL_DIR  Target directory. Default: /usr/local/bin if
#                        writable (or sudo is available), else ~/.local/bin.
#   CLICKUP_BASE_URL     Release base URL. Override to test against a local
#                        server instead of GitHub.
#   CLICKUP_NO_SUDO      Set to 1 to never escalate; falls back to ~/.local/bin.
#
# POSIX sh on purpose: this runs under dash, busybox ash, and bash.

set -eu

REPO="triptechtravel/clickup-cli"
BIN="clickup"
VERSION="${CLICKUP_VERSION:-latest}"
BASE_URL="${CLICKUP_BASE_URL:-https://github.com/${REPO}/releases}"

TMPDIR_INSTALL=""

log()  { printf '%s\n' "$*" >&2; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

cleanup() {
	[ -n "$TMPDIR_INSTALL" ] && rm -rf "$TMPDIR_INSTALL"
}
trap cleanup EXIT INT TERM

# ── platform detection ────────────────────────────────────────────────

detect_os() {
	os="$(uname -s)"
	case "$os" in
		Linux)  printf 'linux' ;;
		Darwin) printf 'darwin' ;;
		*)      die "unsupported OS: $os (Windows users: download the .zip from ${BASE_URL})" ;;
	esac
}

detect_arch() {
	arch="$(uname -m)"
	case "$arch" in
		x86_64|amd64)   printf 'amd64' ;;
		aarch64|arm64)  printf 'arm64' ;;
		*)              die "unsupported architecture: $arch (available: amd64, arm64)" ;;
	esac
}

# ── download helpers (curl or wget; busybox wget is enough) ───────────

have() { command -v "$1" >/dev/null 2>&1; }

download() {
	# download <url> <dest>
	if have curl; then
		curl -fsSL "$1" -o "$2"
	elif have wget; then
		wget -q -O "$2" "$1"
	else
		die "need curl or wget to download files"
	fi
}

sha256_of() {
	# sha256_of <file> — print the bare hex digest
	if have sha256sum; then
		sha256sum "$1" | cut -d' ' -f1
	elif have shasum; then
		shasum -a 256 "$1" | cut -d' ' -f1
	elif have openssl; then
		openssl dgst -sha256 "$1" | awk '{print $NF}'
	else
		printf ''
	fi
}

# ── install location ──────────────────────────────────────────────────

# Sets SUDO to "sudo" when escalation is needed and possible, else "".
resolve_install_dir() {
	SUDO=""
	if [ -n "${CLICKUP_INSTALL_DIR:-}" ]; then
		INSTALL_DIR="$CLICKUP_INSTALL_DIR"
		mkdir -p "$INSTALL_DIR" 2>/dev/null || die "cannot create $INSTALL_DIR"
		[ -w "$INSTALL_DIR" ] || die "$INSTALL_DIR is not writable"
		return
	fi

	default_dir="/usr/local/bin"
	if [ -d "$default_dir" ] && [ -w "$default_dir" ]; then
		INSTALL_DIR="$default_dir"
		return
	fi

	if [ "${CLICKUP_NO_SUDO:-0}" != "1" ] && have sudo; then
		# Only claim sudo if it works without prompting into a broken TTY.
		if sudo -n true 2>/dev/null || [ -t 0 ]; then
			INSTALL_DIR="$default_dir"
			SUDO="sudo"
			return
		fi
	fi

	INSTALL_DIR="${HOME}/.local/bin"
	mkdir -p "$INSTALL_DIR" || die "cannot create $INSTALL_DIR"
}

on_path() {
	case ":${PATH}:" in
		*":$1:"*) return 0 ;;
		*)        return 1 ;;
	esac
}

# ── main ──────────────────────────────────────────────────────────────

main() {
	os="$(detect_os)"
	arch="$(detect_arch)"
	asset="${BIN}_${os}_${arch}.tar.gz"

	if [ "$VERSION" = "latest" ]; then
		url="${BASE_URL}/latest/download/${asset}"
		sums_url="${BASE_URL}/latest/download/checksums.txt"
	else
		url="${BASE_URL}/download/${VERSION}/${asset}"
		sums_url="${BASE_URL}/download/${VERSION}/checksums.txt"
	fi

	TMPDIR_INSTALL="$(mktemp -d)"

	log "Downloading ${asset} (${VERSION})..."
	download "$url" "${TMPDIR_INSTALL}/${asset}" \
		|| die "download failed: $url"

	# Checksum verification is best-effort on the fetch (a release without
	# checksums.txt shouldn't hard-fail), but a *mismatch* always aborts.
	if download "$sums_url" "${TMPDIR_INSTALL}/checksums.txt" 2>/dev/null; then
		expected="$(grep " ${asset}\$" "${TMPDIR_INSTALL}/checksums.txt" | cut -d' ' -f1 || true)"
		actual="$(sha256_of "${TMPDIR_INSTALL}/${asset}")"
		if [ -z "$actual" ]; then
			warn "no sha256 tool found — skipping checksum verification"
		elif [ -z "$expected" ]; then
			warn "${asset} not listed in checksums.txt — skipping verification"
		elif [ "$expected" != "$actual" ]; then
			die "checksum mismatch for ${asset}
  expected: ${expected}
  actual:   ${actual}"
		else
			log "Checksum OK."
		fi
	else
		warn "could not fetch checksums.txt — skipping verification"
	fi

	tar -xzf "${TMPDIR_INSTALL}/${asset}" -C "$TMPDIR_INSTALL" \
		|| die "failed to extract ${asset}"
	[ -f "${TMPDIR_INSTALL}/${BIN}" ] || die "${BIN} not found in archive"
	chmod +x "${TMPDIR_INSTALL}/${BIN}"

	resolve_install_dir
	log "Installing to ${INSTALL_DIR}/${BIN}..."
	$SUDO mkdir -p "$INSTALL_DIR"
	$SUDO cp "${TMPDIR_INSTALL}/${BIN}" "${INSTALL_DIR}/${BIN}"

	installed="$("${INSTALL_DIR}/${BIN}" version 2>/dev/null | head -1 || printf 'installed')"
	log ""
	log "✓ ${installed}"
	log "  ${INSTALL_DIR}/${BIN}"

	if ! on_path "$INSTALL_DIR"; then
		log ""
		warn "${INSTALL_DIR} is not on your PATH. Add it:"
		log "    export PATH=\"${INSTALL_DIR}:\$PATH\""
	fi
}

main "$@"
