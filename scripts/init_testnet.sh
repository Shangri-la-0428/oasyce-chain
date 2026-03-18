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
oasyced init "$MONIKER" --chain-id "$CHAIN_ID"

# Create validator key
oasyced keys add validator --keyring-backend test

# Add genesis account with initial balance (1M OAS = 100000000000000 uoas)
VALIDATOR_ADDR=$(oasyced keys show validator -a --keyring-backend test)
oasyced genesis add-genesis-account "$VALIDATOR_ADDR" "100000000000000${DENOM}"

# Create gentx (stake 100K OAS = 10000000000000 uoas)
oasyced genesis gentx validator "10000000000000${DENOM}" \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test

# Collect gentxs
oasyced genesis collect-gentxs

echo "Testnet initialized! Run: oasyced start"
