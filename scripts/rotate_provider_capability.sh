#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROVIDER_AGENT="${PROVIDER_AGENT:-$SCRIPT_DIR/provider_agent.py}"
SERVICE_NAME="${SERVICE_NAME:-oasyce-provider}"
CAPABILITY_ENV_FILE="${CAPABILITY_ENV_FILE:-/etc/oasyce/provider-capability.env}"

NAME="${NAME:-Claude AI (Opus 4.6)}"
PRICE_UOAS="${PRICE_UOAS:-500000}"
DESCRIPTION="${DESCRIPTION:-Claude Opus 4.6 AI assistant via Oasyce proxy. Send {\"prompt\": \"your question\"} to get AI responses.}"
TAGS="${TAGS:-ai,claude,llm,chat}"

PROVIDER_USER="${PROVIDER_USER:-oasyce}"
OASYCED_BIN="${OASYCED_BIN:-oasyced}"
CHAIN_ID="${OASYCED_CHAIN_ID:-oasyce-testnet-1}"
KEYRING="${OASYCED_KEYRING:-test}"
PROVIDER_KEY="${OASYCE_PROVIDER_KEY:-validator}"
CHAIN_REST="${OASYCE_CHAIN_REST:-http://127.0.0.1:11317}"
CHAIN_RPC="${OASYCE_CHAIN_RPC:-http://127.0.0.1:26667}"
UPSTREAM_API_URL="${UPSTREAM_API_URL:-http://127.0.0.1:8090/v1/chat}"
PROVIDER_PORT="${PROVIDER_PORT:-8430}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    fail "run as root on the provider host"
  fi
}

current_capability_id() {
  if [ -f "$CAPABILITY_ENV_FILE" ]; then
    awk -F= '$1=="OASYCE_CAPABILITY_ID"{print $2}' "$CAPABILITY_ENV_FILE" | tail -n 1
  fi
}

is_capability_active() {
  local cap_id="$1"
  [ -n "$cap_id" ] || return 1
  "$OASYCED_BIN" q oasyce_capability get "$cap_id" --node "$CHAIN_RPC" --output json 2>/dev/null | \
    python3 -c 'import json,sys; d=json.load(sys.stdin); print("1" if d.get("capability",{}).get("is_active") else "0")' | \
    grep -q '^1$'
}

register_new_capability() {
  sudo -u "$PROVIDER_USER" env \
    OASYCE_PROVIDER_KEY="$PROVIDER_KEY" \
    OASYCED_CHAIN_ID="$CHAIN_ID" \
    OASYCED_KEYRING="$KEYRING" \
    OASYCE_CHAIN_REST="$CHAIN_REST" \
    OASYCE_CHAIN_RPC="$CHAIN_RPC" \
    UPSTREAM_API_URL="$UPSTREAM_API_URL" \
    PROVIDER_PORT="$PROVIDER_PORT" \
    "$PROVIDER_AGENT" --register --name "$NAME" --price "$PRICE_UOAS" --description "$DESCRIPTION" --tags "$TAGS" | \
    awk '/^  ID:/{print $2}' | tail -n 1
}

write_current_capability() {
  local cap_id="$1"
  install -d -m 755 "$(dirname "$CAPABILITY_ENV_FILE")"
  cat > "$CAPABILITY_ENV_FILE" <<EOF
OASYCE_CAPABILITY_ID=$cap_id
EOF
  chmod 644 "$CAPABILITY_ENV_FILE"
}

restart_and_verify() {
  systemctl daemon-reload
  systemctl restart "$SERVICE_NAME"
  sleep 2
  systemctl is-active "$SERVICE_NAME" >/dev/null
  curl -fsS "http://127.0.0.1:${PROVIDER_PORT}/health?probe=1" >/dev/null
}

deactivate_capability() {
  local cap_id="$1"
  [ -n "$cap_id" ] || return 0
  sudo -u "$PROVIDER_USER" "$OASYCED_BIN" tx oasyce_capability deactivate "$cap_id" \
    --from "$PROVIDER_KEY" \
    --keyring-backend "$KEYRING" \
    --chain-id "$CHAIN_ID" \
    --gas auto \
    --gas-adjustment 1.5 \
    --fees 10000uoas \
    --yes \
    --output json >/dev/null
}

main() {
  require_root
  command -v "$OASYCED_BIN" >/dev/null 2>&1 || fail "missing $OASYCED_BIN"
  command -v curl >/dev/null 2>&1 || fail "missing curl"
  command -v python3 >/dev/null 2>&1 || fail "missing python3"
  [ -f "$PROVIDER_AGENT" ] || fail "provider agent not found: $PROVIDER_AGENT"

  local old_cap new_cap
  old_cap="$(current_capability_id || true)"
  log "Current capability: ${old_cap:-<none>}"

  new_cap="$(register_new_capability)"
  [ -n "$new_cap" ] || fail "failed to register new capability"
  log "Registered new capability: $new_cap"

  write_current_capability "$new_cap"
  restart_and_verify
  log "Provider restarted and verified with capability: $new_cap"

  if [ -n "$old_cap" ] && [ "$old_cap" != "$new_cap" ] && is_capability_active "$old_cap"; then
    deactivate_capability "$old_cap"
    log "Retired previous capability: $old_cap"
  fi

  log "Current capability is now pinned in: $CAPABILITY_ENV_FILE"
}

main "$@"
