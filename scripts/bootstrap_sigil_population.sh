#!/usr/bin/env bash
# ============================================================
# Bootstrap a Sigil population on the testnet.
#
# Creates N Sigils with unique keys, funds them via faucet,
# and optionally BONDs pairs for initial relationship structure.
#
# Usage:
#   ./scripts/bootstrap_sigil_population.sh           # 5 Sigils, default node
#   SIGIL_COUNT=10 ./scripts/bootstrap_sigil_population.sh
#   NODE=http://localhost:1317 ./scripts/bootstrap_sigil_population.sh
#
# Each Sigil is a potential Loop — connect it to a running agent:
#   PSYCHE_SIGIL_ID=<sigil_id> oasyce-agent run
# ============================================================
set -euo pipefail

NODE="${NODE:-http://47.93.32.88:1317}"
RPC="${RPC:-http://47.93.32.88:26657}"
FAUCET="${FAUCET:-http://47.93.32.88:8080/faucet}"
CHAIN_ID="${CHAIN_ID:-oasyce-testnet-1}"
SIGIL_COUNT="${SIGIL_COUNT:-5}"
HOME_DIR="${HOME_DIR:-$HOME/.oasyced}"
KEYRING="${KEYRING:-test}"
GAS_PRICES="${GAS_PRICES:-0uoas}"

log() { printf '[sigil] %s\n' "$*"; }
fail() { printf '[sigil] ERROR: %s\n' "$*" >&2; exit 1; }

command -v oasyced >/dev/null 2>&1 || fail "oasyced not found. Run: bash <(curl -fsSL .../install_oasyced.sh)"
command -v jq >/dev/null 2>&1 || fail "jq is required"

CREATED_SIGILS=()

create_sigil() {
  local idx="$1"
  local key_name="sigil-${idx}"

  # Create key if it doesn't exist
  if ! oasyced keys show "$key_name" --keyring-backend "$KEYRING" --home "$HOME_DIR" >/dev/null 2>&1; then
    log "Creating key: $key_name"
    oasyced keys add "$key_name" --keyring-backend "$KEYRING" --home "$HOME_DIR" --output json 2>/dev/null | jq -r '.address' > /dev/null
  fi

  local addr
  addr=$(oasyced keys show "$key_name" -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
  log "Key $key_name: $addr"

  # Fund via faucet
  log "Funding $addr..."
  local faucet_resp
  faucet_resp=$(curl -sf "${FAUCET}?address=${addr}" 2>&1 || true)
  if echo "$faucet_resp" | grep -q "error\|rate"; then
    log "  Faucet: $faucet_resp (may already be funded)"
  else
    log "  Funded."
  fi

  # Wait for funds to land on-chain (faucet TX needs ~6s for 1 block)
  sleep 8

  # Get pubkey hex
  local pubkey_json
  pubkey_json=$(oasyced keys show "$key_name" --keyring-backend "$KEYRING" --home "$HOME_DIR" --output json 2>/dev/null)
  # Extract the raw pubkey bytes and convert to hex
  local pubkey_hex
  pubkey_hex=$(echo "$pubkey_json" | jq -r '.pubkey' | python3 -c "
import sys, json, base64
pk = json.loads(sys.stdin.read())
raw = base64.b64decode(pk['key'])
print(raw.hex())
" 2>/dev/null || echo "")

  if [ -z "$pubkey_hex" ]; then
    log "  WARNING: Could not extract pubkey hex, skipping Sigil creation"
    return
  fi

  # Create Sigil
  log "Creating Sigil for $key_name (pubkey: ${pubkey_hex:0:16}...)"
  local result
  result=$(oasyced tx sigil genesis "$pubkey_hex" \
    --metadata "{\"name\":\"sigil-${idx}\",\"bootstrap\":true}" \
    --from "$key_name" \
    --keyring-backend "$KEYRING" \
    --home "$HOME_DIR" \
    --chain-id "$CHAIN_ID" \
    --node "$RPC" \
    --fees 1000uoas \
    --yes \
    --output json 2>&1 || true)

  local txhash
  txhash=$(echo "$result" | jq -r '.txhash // empty' 2>/dev/null || echo "")

  if [ -n "$txhash" ]; then
    log "  TX: $txhash"
    # Derive the sigil ID (same as chain: SHA256(pubkey)[:16] → "SIG_" + hex)
    local sigil_id
    sigil_id=$(python3 -c "
import hashlib
h = hashlib.sha256(bytes.fromhex('${pubkey_hex}')).digest()[:16]
print('SIG_' + h.hex())
" 2>/dev/null || echo "unknown")
    log "  Sigil ID: $sigil_id"
    CREATED_SIGILS+=("$sigil_id")
  else
    log "  Failed: $result"
  fi
}

bond_sigils() {
  local sigil_a="$1"
  local sigil_b="$2"
  local from_key="$3"

  log "Bonding $sigil_a ↔ $sigil_b"
  oasyced tx sigil bond "$sigil_a" "$sigil_b" \
    --scope "bootstrap" \
    --from "$from_key" \
    --keyring-backend "$KEYRING" \
    --home "$HOME_DIR" \
    --chain-id "$CHAIN_ID" \
    --node "$RPC" \
    --fees 1000uoas \
    --yes \
    --output json 2>/dev/null | jq -r '.txhash // "failed"' || true
}

main() {
  log "Bootstrapping $SIGIL_COUNT Sigils on $CHAIN_ID"
  log "Node: $NODE | Faucet: $FAUCET"
  log ""

  # Create Sigils
  for i in $(seq 1 "$SIGIL_COUNT"); do
    create_sigil "$i"
    sleep 2  # Avoid faucet rate limit
  done

  log ""
  log "Created ${#CREATED_SIGILS[@]} Sigils:"
  for sid in "${CREATED_SIGILS[@]+"${CREATED_SIGILS[@]}"}"; do
    log "  $sid"
  done

  # Bond adjacent pairs (1↔2, 2↔3, 3↔4, ...)
  if [ "${#CREATED_SIGILS[@]}" -ge 2 ]; then
    log ""
    log "Creating initial BONDs..."
    sleep 6  # Wait for GENESIS txs to land
    for i in $(seq 0 $((${#CREATED_SIGILS[@]} - 2))); do
      local key_idx=$((i + 1))
      bond_sigils "${CREATED_SIGILS[$i]}" "${CREATED_SIGILS[$((i + 1))]}" "sigil-${key_idx}"
      sleep 2
    done
  fi

  log ""
  log "============================================"
  log "  Sigil Population Bootstrap Complete"
  log "============================================"
  log ""
  log "Next steps:"
  log "  1. Query your Sigils:"
  log "     oasyced query sigil active-count --node $RPC"
  log "     oasyced query sigil sigil <SIGIL_ID> --node $RPC"
  log ""
  log "  2. Connect a Sigil to an agent:"
  log "     PSYCHE_SIGIL_ID=<SIGIL_ID> oasyce-agent run"
  log ""
  log "  3. Keep Sigils alive — any chain TX from the Sigil's"
  log "     creator touches its liveness. Dormant after ~3 days"
  log "     of inactivity, dissolved after ~9 days."
}

main "$@"
