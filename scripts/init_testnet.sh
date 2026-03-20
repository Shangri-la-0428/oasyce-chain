#!/bin/bash
# Initialize a single-node testnet for development.
#
# Works around Cosmos SDK v0.50 keyring serialization bugs:
#   - Uses --keyring-backend test (no password required, simpler for local dev)
#   - If keys show fails, falls back to --dry-run + address-based genesis account
#
set -euo pipefail

CHAIN_ID="oasyce-testnet-1"
MONIKER="oasyce-validator"
HOME_DIR="$HOME/.oasyced"
DENOM="uoas"
BINARY="${BINARY:-oasyced}"

# Clean previous state
echo "==> Cleaning previous state..."
rm -rf "$HOME_DIR"

# Init chain
echo "==> Initializing chain (${CHAIN_ID})..."
$BINARY init "$MONIKER" --chain-id "$CHAIN_ID" 2>/dev/null

# ---------------------------------------------------------------------------
# Key generation with serialization bug workaround.
#
# Strategy:
#   1. Generate key with --dry-run to get mnemonic + address (no keyring write).
#   2. Recover into --keyring-backend test.
#   3. Verify with keys show; if it fails, use the address directly.
# ---------------------------------------------------------------------------
echo "==> Generating validator key..."
KEY_JSON=$($BINARY keys add validator --dry-run --output json 2>&1)
VALIDATOR_ADDR=$(echo "$KEY_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
MNEMONIC=$(echo "$KEY_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")

echo "Validator address: $VALIDATOR_ADDR"
echo "Mnemonic (save this!): $MNEMONIC"
echo ""

# Recover key into test keyring (no password needed)
echo "$MNEMONIC" | $BINARY keys add validator --recover --keyring-backend test 2>/dev/null

# Verify key is usable
KEY_OK=true
$BINARY keys show validator --keyring-backend test -a 2>/dev/null && echo "Key recovered successfully into test keyring." || {
  echo "Warning: keys show failed (SDK serialization bug). Using address directly."
  KEY_OK=false
}

# Add genesis account with initial balance (1M OAS = 100_000_000_000_000 uoas)
echo "==> Adding genesis account..."
$BINARY genesis add-genesis-account "$VALIDATOR_ADDR" "100000000000000${DENOM}"

# Create gentx
echo "==> Creating gentx..."
if $KEY_OK; then
  $BINARY genesis gentx validator "10000000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test 2>&1 | tail -1
else
  # Fallback: pipe mnemonic to recover + gentx in one shot
  echo "$MNEMONIC" | $BINARY keys add validator --recover --keyring-backend test --home "$HOME_DIR" 2>/dev/null || true
  $BINARY genesis gentx validator "10000000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test 2>&1 | tail -1
fi

# Collect gentxs
echo "==> Collecting gentxs..."
$BINARY genesis collect-gentxs 2>/dev/null

# Enable REST API for dev
sed -i '' 's/^enable = false$/enable = true/' "$HOME_DIR/config/app.toml"
# Set pruning to nothing so REST queries work at any height
sed -i '' 's/^pruning = "default"/pruning = "nothing"/' "$HOME_DIR/config/app.toml"
# Disable IAVL fast node to avoid "version does not exist" errors on queries
sed -i '' 's/^iavl-disable-fastnode = false/iavl-disable-fastnode = true/' "$HOME_DIR/config/app.toml"

# Verify
GENTX_COUNT=$(python3 -c "import json; g=json.load(open('$HOME_DIR/config/genesis.json')); print(len(g['app_state']['genutil']['gen_txs']))")
echo ""
echo "========================================="
echo "  Oasyce single-node testnet initialized!"
echo "  Chain ID:  $CHAIN_ID"
echo "  Validator: $VALIDATOR_ADDR"
echo "  Gentxs:    $GENTX_COUNT"
echo "  Keyring:   test (no password)"
echo ""
echo "  Start with:"
echo "    $BINARY start"
echo "========================================="
