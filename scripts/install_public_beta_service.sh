#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_NAME="${SERVICE_NAME:-oasyced-public-beta}"
RUN_USER="${RUN_USER:-$USER}"
HOME_DIR="${HOME_DIR:-$HOME/.oasyced}"
MIN_GAS_PRICES="${MIN_GAS_PRICES:-0uoas}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
SUDO_BIN="${SUDO_BIN:-sudo}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_linux() {
  case "$(uname -s)" in
    Linux) ;;
    *) fail "systemd service install is supported on Linux only" ;;
  esac
}

run_as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  else
    "$SUDO_BIN" "$@"
  fi
}

main() {
  require_linux
  command -v "$SYSTEMCTL_BIN" >/dev/null 2>&1 || fail "systemctl is required"

  log "==> Preparing public beta node state"
  INSTALL_CLI=1 START_NODE=0 HOME_DIR="$HOME_DIR" bash "$SCRIPT_DIR/bootstrap_public_beta_node.sh"

  local oasyced_path service_path
  oasyced_path="$(command -v oasyced || true)"
  [ -n "$oasyced_path" ] || fail "oasyced not found after bootstrap"
  service_path="/etc/systemd/system/${SERVICE_NAME}.service"

  log "==> Writing ${service_path}"
  run_as_root tee "$service_path" >/dev/null <<EOF
[Unit]
Description=Oasyce Public Beta Node
After=network-online.target
Wants=network-online.target

[Service]
User=${RUN_USER}
ExecStart=${oasyced_path} start --home ${HOME_DIR} --minimum-gas-prices ${MIN_GAS_PRICES}
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

  log "==> Enabling and starting ${SERVICE_NAME}"
  run_as_root "$SYSTEMCTL_BIN" daemon-reload
  run_as_root "$SYSTEMCTL_BIN" enable --now "${SERVICE_NAME}"

  log ""
  log "Service installed:"
  log "  ${SERVICE_NAME}"
  log ""
  log "Useful commands:"
  log "  ${SYSTEMCTL_BIN} status ${SERVICE_NAME}"
  log "  journalctl -u ${SERVICE_NAME} -f"
}

main "$@"
