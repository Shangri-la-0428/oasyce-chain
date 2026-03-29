#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHAIN_ID="${CHAIN_ID:-oasyce-testnet-1}"
MONIKER="${MONIKER:-oasyce-node-$(openssl rand -hex 3)}"
HOME_DIR="${HOME_DIR:-$HOME/.oasyced}"
SEED_NODE="${SEED_NODE:-3e5a914ab7e7400091ddf461fb14992de785b0cb@47.93.32.88:26656}"
GENESIS_URL="${GENESIS_URL:-https://github.com/Shangri-la-0428/oasyce-chain/releases/download/testnet-1/genesis.json}"
GENESIS_SHA256="${GENESIS_SHA256:-dcc6508926567bc384220d1e92ef538d25c8e5431c380420459b0210d30c7739}"
INSTALL_CLI="${INSTALL_CLI:-1}"
START_NODE="${START_NODE:-0}"
MIN_GAS_PRICES="${MIN_GAS_PRICES:-0uoas}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

install_cli_if_needed() {
  if command -v oasyced >/dev/null 2>&1; then
    return
  fi
  if [ "$INSTALL_CLI" != "1" ]; then
    fail "oasyced not found and INSTALL_CLI=0"
  fi
  log "==> oasyced not found; installing latest CLI first"
  bash "$SCRIPT_DIR/install_oasyced.sh"
  export PATH="$HOME/.local/bin:$PATH"
  command -v oasyced >/dev/null 2>&1 || fail "oasyced still not found after install"
}

portable_sed() {
  local expr="$1"
  local file="$2"
  if [[ "${OSTYPE:-}" == darwin* ]]; then
    sed -i '' "$expr" "$file"
  else
    sed -i "$expr" "$file"
  fi
}

ensure_init() {
  if [ ! -f "$HOME_DIR/config/config.toml" ]; then
    log "==> Initializing node home"
    oasyced init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" >/dev/null 2>&1
  else
    log "==> Reusing existing node home"
  fi
}

download_genesis() {
  mkdir -p "$HOME_DIR/config"
  log "==> Downloading genesis.json"
  curl -fsSL "$GENESIS_URL" -o "$HOME_DIR/config/genesis.json"

  if command -v sha256sum >/dev/null 2>&1; then
    local actual
    actual="$(sha256sum "$HOME_DIR/config/genesis.json" | awk '{print $1}')"
    [ "$actual" = "$GENESIS_SHA256" ] || fail "genesis checksum mismatch"
  elif command -v shasum >/dev/null 2>&1; then
    local actual
    actual="$(shasum -a 256 "$HOME_DIR/config/genesis.json" | awk '{print $1}')"
    [ "$actual" = "$GENESIS_SHA256" ] || fail "genesis checksum mismatch"
  else
    log "WARNING: no SHA256 tool found; skipping checksum verification"
  fi
}

configure_node() {
  local config app
  config="$HOME_DIR/config/config.toml"
  app="$HOME_DIR/config/app.toml"

  log "==> Configuring seed peer and APIs"
  portable_sed "s|^persistent_peers = \".*\"|persistent_peers = \"$SEED_NODE\"|" "$config"
  portable_sed "s|^enable = false|enable = true|" "$app"
  portable_sed "s|^address = \"tcp://localhost:1317\"|address = \"tcp://0.0.0.0:1317\"|" "$app"
  portable_sed "s|^minimum-gas-prices = \".*\"|minimum-gas-prices = \"$MIN_GAS_PRICES\"|" "$app"
}

patch_genesis() {
  local genesis_path
  genesis_path="$HOME_DIR/config/genesis.json"
  python3 <<PYEOF
import json
path = "${genesis_path}"
with open(path) as f:
    g = json.load(f)
changed = False
if "oasyce_capability" in g.get("app_state", {}):
    params = g["app_state"]["oasyce_capability"]["params"]
    if params.get("min_provider_stake", {}).get("amount") != "0":
        params["min_provider_stake"] = {"denom": "uoas", "amount": "0"}
        changed = True
if "onboarding" in g.get("app_state", {}):
    params = g["app_state"]["onboarding"]["params"]
    if params.get("pow_difficulty") != 8:
        params["pow_difficulty"] = 8
        changed = True
if changed:
    with open(path, "w") as f:
        json.dump(g, f, indent=2)
PYEOF
}

start_node_if_requested() {
  if [ "$START_NODE" != "1" ]; then
    return
  fi
  log "==> Starting node"
  exec oasyced start --home "$HOME_DIR" --minimum-gas-prices "$MIN_GAS_PRICES"
}

print_summary() {
  log ""
  log "============================================"
  log "  Oasyce Public Beta Node Ready"
  log "============================================"
  log "  Chain ID:  $CHAIN_ID"
  log "  Moniker:   $MONIKER"
  log "  Home:      $HOME_DIR"
  log ""
  log "Next:"
  log "  oasyced start --home \"$HOME_DIR\" --minimum-gas-prices \"$MIN_GAS_PRICES\""
  log ""
  log "After sync:"
  log "  oasyced status --home \"$HOME_DIR\" | jq '.SyncInfo.catching_up'"
}

main() {
  command -v curl >/dev/null 2>&1 || fail "curl is required"
  command -v python3 >/dev/null 2>&1 || fail "python3 is required"

  install_cli_if_needed
  ensure_init
  download_genesis
  patch_genesis
  configure_node
  print_summary
  start_node_if_requested
}

main "$@"
