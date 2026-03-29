#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHAIN_ID="${CHAIN_ID:-oasyce-testnet-1}"
KEY_NAME="${KEY_NAME:-agent}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
HOME_DIR="${HOME_DIR:-$HOME/.oasyced}"
REST_URL="${REST_URL:-http://47.93.32.88:1317}"
FAUCET_URL="${FAUCET_URL:-http://47.93.32.88:8080/faucet}"
REQUEST_FAUCET="${REQUEST_FAUCET:-1}"
INSTALL_CLI="${INSTALL_CLI:-1}"

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

ensure_home() {
  if [ ! -f "$HOME_DIR/config/client.toml" ]; then
    mkdir -p "$HOME_DIR"
    oasyced init "oasyce-account-bootstrap" --chain-id "$CHAIN_ID" --home "$HOME_DIR" >/dev/null 2>&1 || true
  fi
}

ensure_key() {
  local addr key_json key_output mnemonic_file mnemonic
  if addr="$(oasyced keys show "$KEY_NAME" -a --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" 2>/dev/null)"; then
    printf '%s' "$addr"
    return
  fi

  log "==> Creating key: $KEY_NAME"
  key_output="$(oasyced keys add "$KEY_NAME" --keyring-backend "$KEYRING_BACKEND" --home "$HOME_DIR" --output json 2>&1)"
  key_json="$(printf '%s\n' "$key_output" | python3 -c '
import json, sys
text = sys.stdin.read()
for line in reversed([line.strip() for line in text.splitlines() if line.strip()]):
    if line.startswith("{") and line.endswith("}"):
        json.loads(line)
        print(line)
        break
else:
    raise SystemExit("could not extract key JSON")
')"
  addr="$(printf '%s' "$key_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["address"])')"
  mnemonic="$(printf '%s' "$key_json" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("mnemonic",""))')"

  if [ -n "$mnemonic" ]; then
    mnemonic_file="$HOME_DIR/${KEY_NAME}.mnemonic"
    printf '%s\n' "$mnemonic" > "$mnemonic_file"
    chmod 600 "$mnemonic_file"
    log "    Mnemonic saved to: $mnemonic_file"
  fi

  printf '%s' "$addr"
}

request_faucet() {
  local address="$1"
  if [ "$REQUEST_FAUCET" != "1" ]; then
    return 0
  fi
  log "==> Requesting faucet funds"
  curl -fsSL "${FAUCET_URL}?address=${address}" >/dev/null
}

print_summary() {
  local address="$1"
  log ""
  log "============================================"
  log "  Oasyce Public Beta Account Ready"
  log "============================================"
  log "  Key name:   $KEY_NAME"
  log "  Address:    $address"
  log "  Home:       $HOME_DIR"
  log "  Keyring:    $KEYRING_BACKEND"
  log ""
  log "Next:"
  log "  oasyced keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home \"$HOME_DIR\""
  log "  curl \"$REST_URL/cosmos/bank/v1beta1/balances/$address\""
}

main() {
  command -v curl >/dev/null 2>&1 || fail "curl is required"
  install_cli_if_needed
  ensure_home
  local address
  address="$(ensure_key)"
  request_faucet "$address"
  print_summary "$address"
}

main "$@"
