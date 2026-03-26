#!/bin/bash
# =============================================================================
# Initialize Oasyce Public Testnet Genesis
#
# Creates a genesis.json with:
#   - Seed validators (initial validator set)
#   - Faucet account (for distributing test tokens)
#   - Community pool allocation
#   - Correct module parameters for testnet
#
# Usage:
#   bash scripts/init_public_testnet.sh
#
# Output:
#   ~/.oasyce-testnet/genesis.json     — distribute to validators
#   ~/.oasyce-testnet/faucet.mnemonic  — faucet key (keep secure)
# =============================================================================
set -euo pipefail

CHAIN_ID="oasyce-testnet-1"
DENOM="uoas"
BINARY="${BINARY:-oasyced}"
TESTNET_DIR="$HOME/.oasyce-testnet"
KB="--keyring-backend test"

# Genesis allocation (in OAS, converted to uoas below)
# Total initial supply: 100M OAS
VALIDATOR_STAKE=500000     # 500K OAS per seed validator
VALIDATOR_BALANCE=1000000  # 1M OAS per seed validator (includes stake)
FAUCET_BALANCE=50000000    # 50M OAS for faucet (public distribution)

echo "==> Initializing Oasyce Public Testnet"
echo "    Chain ID: $CHAIN_ID"
echo "    Output:   $TESTNET_DIR"
echo ""

# Clean slate
rm -rf "$TESTNET_DIR"
mkdir -p "$TESTNET_DIR"

# Initialize node (for genesis template)
NODE_HOME="$TESTNET_DIR/node"
$BINARY init "oasyce-seed-0" --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null

# ---------------------------------------------------------------------------
# Create keys
# ---------------------------------------------------------------------------
echo "==> Creating seed validator key..."
VAL_JSON=$($BINARY keys add seed-validator --dry-run --output json 2>&1)
VAL_ADDR=$(echo "$VAL_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
VAL_MNEMONIC=$(echo "$VAL_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")
echo "$VAL_MNEMONIC" | $BINARY keys add seed-validator --recover $KB --home "$NODE_HOME" 2>/dev/null
echo "    Seed validator: $VAL_ADDR"

echo "==> Creating faucet key..."
FAUCET_JSON=$($BINARY keys add faucet --dry-run --output json 2>&1)
FAUCET_ADDR=$(echo "$FAUCET_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
FAUCET_MNEMONIC=$(echo "$FAUCET_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")
echo "$FAUCET_MNEMONIC" | $BINARY keys add faucet --recover $KB --home "$NODE_HOME" 2>/dev/null
echo "    Faucet: $FAUCET_ADDR"

# Save mnemonics
cat > "$TESTNET_DIR/seed-validator.mnemonic" << EOF
# Oasyce Public Testnet — Seed Validator Mnemonic
# Chain ID: $CHAIN_ID
# Address: $VAL_ADDR
# WARNING: Keep this secure. Controls seed validator stake.
$VAL_MNEMONIC
EOF

cat > "$TESTNET_DIR/faucet.mnemonic" << EOF
# Oasyce Public Testnet — Faucet Mnemonic
# Chain ID: $CHAIN_ID
# Address: $FAUCET_ADDR
# WARNING: Keep this secure. Controls faucet funds.
$FAUCET_MNEMONIC
EOF

chmod 600 "$TESTNET_DIR/seed-validator.mnemonic" "$TESTNET_DIR/faucet.mnemonic"

# ---------------------------------------------------------------------------
# Genesis accounts
# ---------------------------------------------------------------------------
echo "==> Adding genesis accounts..."

# Validator (1M OAS)
VAL_UOAS=$((VALIDATOR_BALANCE * 1000000))
$BINARY genesis add-genesis-account "$VAL_ADDR" "${VAL_UOAS}${DENOM}" --home "$NODE_HOME"
echo "    Seed validator: $(python3 -c "print(f'{$VALIDATOR_BALANCE:,}')") OAS"

# Faucet (50M OAS)
FAUCET_UOAS=$((FAUCET_BALANCE * 1000000))
$BINARY genesis add-genesis-account "$FAUCET_ADDR" "${FAUCET_UOAS}${DENOM}" --home "$NODE_HOME"
echo "    Faucet: $(python3 -c "print(f'{$FAUCET_BALANCE:,}')") OAS"

# ---------------------------------------------------------------------------
# Gentx (seed validator)
# ---------------------------------------------------------------------------
echo "==> Creating seed validator gentx..."
STAKE_UOAS=$((VALIDATOR_STAKE * 1000000))
$BINARY genesis gentx seed-validator "${STAKE_UOAS}${DENOM}" \
  --chain-id "$CHAIN_ID" \
  --moniker "oasyce-seed-0" \
  --commission-rate "0.10" \
  --commission-max-rate "0.20" \
  --commission-max-change-rate "0.01" \
  --min-self-delegation "1" \
  $KB --home "$NODE_HOME" 2>/dev/null

$BINARY genesis collect-gentxs --home "$NODE_HOME" 2>/dev/null

# ---------------------------------------------------------------------------
# Patch genesis parameters for public testnet
# ---------------------------------------------------------------------------
echo "==> Patching genesis parameters..."
GENESIS="$NODE_HOME/config/genesis.json"

python3 << PYEOF
import json

with open("$GENESIS") as f:
    g = json.load(f)

# --- Capability module: lower min_provider_stake for testnet ---
if "oasyce_capability" in g["app_state"]:
    g["app_state"]["oasyce_capability"]["params"]["min_provider_stake"] = {
        "denom": "uoas", "amount": "0"
    }
    print("    capability: min_provider_stake = 0")

# --- Datarights module: lower dispute deposit for testnet ---
if "datarights" in g["app_state"]:
    g["app_state"]["datarights"]["params"]["dispute_deposit"] = {
        "denom": "uoas", "amount": "1000000"  # 1 OAS (instead of 10)
    }
    print("    datarights: dispute_deposit = 1 OAS")

# --- Work module: lower min executor reputation for testnet ---
if "work" in g["app_state"]:
    g["app_state"]["work"]["params"]["min_executor_reputation"] = 0
    print("    work: min_executor_reputation = 0")

# --- Reputation: lower feedback cooldown for testnet ---
if "reputation" in g["app_state"]:
    g["app_state"]["reputation"]["params"]["feedback_cooldown_seconds"] = 60
    print("    reputation: feedback_cooldown = 60s")

# onboarding: pow_difficulty stays at code default (16) — matches Epoch 0 halving

with open("$GENESIS", "w") as f:
    json.dump(g, f, indent=2)

print("    Genesis patched successfully.")
PYEOF

# ---------------------------------------------------------------------------
# Copy genesis to output
# ---------------------------------------------------------------------------
cp "$GENESIS" "$TESTNET_DIR/genesis.json"

# Validate genesis
$BINARY genesis validate --home "$NODE_HOME" 2>/dev/null && echo "    Genesis validated." || echo "    WARNING: Genesis validation failed."

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
TOTAL_SUPPLY=$((VALIDATOR_BALANCE + FAUCET_BALANCE))

echo ""
echo "============================================"
echo "  Oasyce Public Testnet Genesis Ready!"
echo "============================================"
echo ""
echo "  Chain ID:      $CHAIN_ID"
echo "  Total supply:  $(python3 -c "print(f'{$TOTAL_SUPPLY:,}')") OAS"
echo ""
echo "  Allocations:"
echo "    Seed validator: $(python3 -c "print(f'{$VALIDATOR_BALANCE:,}')") OAS ($(python3 -c "print(f'{$VALIDATOR_STAKE:,}')") staked)"
echo "    Faucet:         $(python3 -c "print(f'{$FAUCET_BALANCE:,}')") OAS"
echo ""
echo "  Block Rewards (halving schedule):"
echo "    Blocks 0-10M:   4 OAS/block  (~25.2M OAS/year)"
echo "    Blocks 10M-20M: 2 OAS/block  (~12.6M OAS/year)"
echo "    Blocks 20M-30M: 1 OAS/block  (~6.3M OAS/year)"
echo "    Blocks 30M+:    0.5 OAS/block (~3.15M OAS/year)"
echo ""
echo "  Files:"
echo "    Genesis:              $TESTNET_DIR/genesis.json"
echo "    Validator mnemonic:   $TESTNET_DIR/seed-validator.mnemonic"
echo "    Faucet mnemonic:      $TESTNET_DIR/faucet.mnemonic"
echo ""
echo "  Next steps:"
echo "    1. Distribute genesis.json to validators"
echo "    2. Each validator: oasyced init <moniker> --chain-id $CHAIN_ID"
echo "    3. Replace genesis.json, configure peers"
echo "    4. Create validator: oasyced tx staking create-validator ..."
echo "    5. Start faucet: FAUCET_KEY=faucet CHAIN_ID=$CHAIN_ID bash scripts/faucet.sh"
echo ""
