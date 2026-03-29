#!/usr/bin/env bash
set -euo pipefail

REPO="Shangri-la-0428/oasyce-chain"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="oasyced"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

detect_os() {
  local uname_s
  uname_s="$(uname -s)"
  case "$uname_s" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    MINGW*|MSYS*|CYGWIN*) printf 'windows' ;;
    *) fail "Unsupported OS: $uname_s" ;;
  esac
}

detect_arch() {
  local uname_m
  uname_m="$(uname -m)"
  case "$uname_m" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) fail "Unsupported architecture: $uname_m" ;;
  esac
}

resolve_version() {
  if [ "$VERSION" != "latest" ]; then
    printf '%s' "$VERSION"
    return
  fi

  local api tag
  api="https://api.github.com/repos/$REPO/releases/latest"
  tag="$(curl -fsSL "$api" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  [ -n "$tag" ] || fail "Could not resolve latest release tag from GitHub"
  printf '%s' "$tag"
}

main() {
  command -v curl >/dev/null 2>&1 || fail "curl is required"

  local os arch ext tag asset url target
  os="$(detect_os)"
  arch="$(detect_arch)"
  ext=""
  if [ "$os" = "windows" ]; then
    ext=".exe"
  fi

  tag="$(resolve_version)"
  asset="${BIN_NAME}-${os}-${arch}${ext}"
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"

  mkdir -p "$INSTALL_DIR"
  target="$INSTALL_DIR/${BIN_NAME}${ext}"

  log "==> Installing ${BIN_NAME} ${tag}"
  log "    Asset: ${asset}"
  log "    Target: ${target}"

  curl -fL "$url" -o "$target"
  chmod +x "$target"

  if "$target" version >/dev/null 2>&1; then
    log "==> Installed successfully"
    log "    Version: $("$target" version)"
  else
    log "==> Installed, but version check failed"
  fi

  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
      log ""
      log "Add this directory to PATH if needed:"
      log "  export PATH=\"$INSTALL_DIR:\$PATH\""
      ;;
  esac

  if [ "$os" = "windows" ]; then
    log ""
    log "Git Bash users can run:"
    log "  \"$target\" version"
  fi
}

main "$@"
