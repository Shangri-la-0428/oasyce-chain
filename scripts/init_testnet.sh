#!/bin/bash
# Initialize a single-node testnet for development.
set -euo pipefail

CHAIN_ID="oasyce-testnet-1"
MONIKER="oasyce-validator"
HOME_DIR="$HOME/.oasyced"
DENOM="uoas"

# Clean previous state
rm -rf "$HOME_DIR"

# Init
oasyced init "$MONIKER" --chain-id "$CHAIN_ID" 2>/dev/null

# Create validator key and capture address from output
KEY_OUTPUT=$(oasyced keys add validator --keyring-backend test --output json 2>&1 || true)
VALIDATOR_ADDR=$(echo "$KEY_OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])" 2>/dev/null || echo "$KEY_OUTPUT" | grep -oE 'oasyce1[a-z0-9]+')
echo "Validator address: $VALIDATOR_ADDR"

# Add genesis account with initial balance (1M OAS = 100_000_000_000_000 uoas)
oasyced genesis add-genesis-account "$VALIDATOR_ADDR" "100000000000000${DENOM}"

# Create gentx (stake 100K OAS = 10_000_000_000_000 uoas)
oasyced genesis gentx validator "10000000000000${DENOM}" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test

# Collect gentxs
oasyced genesis collect-gentxs 2>/dev/null

echo ""
echo "========================================="
echo "  Oasyce testnet initialized!"
echo "  Chain ID: $CHAIN_ID"
echo "  Validator: $VALIDATOR_ADDR"
echo "  Run: oasyced start"
echo "========================================="
