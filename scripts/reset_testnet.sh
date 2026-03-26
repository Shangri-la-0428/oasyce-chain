#!/bin/bash
# =============================================================================
# Reset Oasyce Testnet on VPS
#
# Stops the chain, wipes all chain data, regenerates genesis with correct
# parameters, and restarts. Recovers existing validator/faucet keys so
# addresses stay the same.
#
# Prerequisites:
#   - New binary already deployed to /usr/local/bin/oasyced
#   - Existing mnemonics in /home/oasyce/secrets/
#   - Run as root on the VPS
#
# Usage:
#   bash /opt/oasyce/src/scripts/reset_testnet.sh
# =============================================================================
set -euo pipefail

CHAIN_ID="oasyce-testnet-1"
DENOM="uoas"
BINARY="/usr/local/bin/oasyced"
SERVICE_USER="oasyce"
NODE_HOME="/home/${SERVICE_USER}/.oasyced"
SECRETS_DIR="/home/${SERVICE_USER}/secrets"

# Genesis allocation (OAS)
VALIDATOR_BALANCE=1000000   # 1M OAS
VALIDATOR_STAKE=500000      # 500K OAS staked
FAUCET_BALANCE=50000000     # 50M OAS

echo "============================================"
echo "  Oasyce Testnet — Chain Reset"
echo "============================================"
echo ""
echo "  Chain ID:   $CHAIN_ID"
echo "  Node home:  $NODE_HOME"
echo "  Binary:     $($BINARY version 2>/dev/null || echo 'unknown')"
echo ""

# Must be root
if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: Run as root."
    exit 1
fi

# ── 1. Stop services ──
echo "==> Stopping services..."
systemctl stop oasyce-faucet 2>/dev/null || true
systemctl stop oasyced 2>/dev/null || true
sleep 2
echo "    Services stopped."

# ── 2. Verify mnemonics exist ──
if [ ! -f "${SECRETS_DIR}/seed-validator.mnemonic" ]; then
    echo "ERROR: ${SECRETS_DIR}/seed-validator.mnemonic not found."
    echo "       Cannot recover keys. Aborting."
    exit 1
fi
if [ ! -f "${SECRETS_DIR}/faucet.mnemonic" ]; then
    echo "ERROR: ${SECRETS_DIR}/faucet.mnemonic not found."
    echo "       Cannot recover keys. Aborting."
    exit 1
fi
echo "==> Mnemonics found. Keys will be recovered."

# ── 3. Wipe chain data ──
echo "==> Wiping chain data..."
rm -rf "$NODE_HOME"
echo "    ${NODE_HOME} removed."

# ── 4. Re-initialize ──
echo "==> Initializing fresh chain..."
sudo -u "$SERVICE_USER" "$BINARY" init "seed-node" \
    --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null
echo "    Chain initialized."

# ── 5. Recover keys ──
echo "==> Recovering validator key..."
VAL_MNEMONIC=$(grep -v '^#' "${SECRETS_DIR}/seed-validator.mnemonic" | grep -v '^$' | head -1)
echo "$VAL_MNEMONIC" | sudo -u "$SERVICE_USER" "$BINARY" keys add seed-validator \
    --recover --keyring-backend test --home "$NODE_HOME" 2>/dev/null
VAL_ADDR=$(sudo -u "$SERVICE_USER" "$BINARY" keys show seed-validator -a \
    --keyring-backend test --home "$NODE_HOME")
echo "    Validator: $VAL_ADDR"

echo "==> Recovering faucet key..."
FAUCET_MNEMONIC=$(grep -v '^#' "${SECRETS_DIR}/faucet.mnemonic" | grep -v '^$' | head -1)
echo "$FAUCET_MNEMONIC" | sudo -u "$SERVICE_USER" "$BINARY" keys add faucet \
    --recover --keyring-backend test --home "$NODE_HOME" 2>/dev/null
FAUCET_ADDR=$(sudo -u "$SERVICE_USER" "$BINARY" keys show faucet -a \
    --keyring-backend test --home "$NODE_HOME")
echo "    Faucet:    $FAUCET_ADDR"

# ── 6. Genesis accounts ──
echo "==> Adding genesis accounts..."
VAL_UOAS=$((VALIDATOR_BALANCE * 1000000))
FAUCET_UOAS=$((FAUCET_BALANCE * 1000000))

sudo -u "$SERVICE_USER" "$BINARY" genesis add-genesis-account "$VAL_ADDR" \
    "${VAL_UOAS}${DENOM}" --home "$NODE_HOME"
sudo -u "$SERVICE_USER" "$BINARY" genesis add-genesis-account "$FAUCET_ADDR" \
    "${FAUCET_UOAS}${DENOM}" --home "$NODE_HOME"

TOTAL_OAS=$(( (VAL_UOAS + FAUCET_UOAS) / 1000000 ))
echo "    Total supply: ${TOTAL_OAS} OAS"

# ── 7. Gentx ──
echo "==> Creating validator gentx..."
STAKE_UOAS=$((VALIDATOR_STAKE * 1000000))
sudo -u "$SERVICE_USER" "$BINARY" genesis gentx seed-validator "${STAKE_UOAS}${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --moniker "seed-node" \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --min-self-delegation "1" \
    --keyring-backend test \
    --home "$NODE_HOME" 2>/dev/null

sudo -u "$SERVICE_USER" "$BINARY" genesis collect-gentxs --home "$NODE_HOME" 2>/dev/null
echo "    Gentx created and collected."

# ── 8. Patch genesis parameters for testnet ──
echo "==> Patching genesis parameters..."
GENESIS="${NODE_HOME}/config/genesis.json"

python3 - "$GENESIS" << 'PYEOF'
import json, sys

genesis_path = sys.argv[1]
with open(genesis_path) as f:
    g = json.load(f)

app = g["app_state"]
changes = []

# capability: min_provider_stake = 0 (no barrier for testnet)
if "oasyce_capability" in app:
    app["oasyce_capability"]["params"]["min_provider_stake"] = {
        "denom": "uoas", "amount": "0"
    }
    changes.append("capability.min_provider_stake = 0")

# datarights: dispute_deposit = 1 OAS (lowered from 10 OAS default)
if "datarights" in app:
    app["datarights"]["params"]["dispute_deposit"] = {
        "denom": "uoas", "amount": "1000000"
    }
    changes.append("datarights.dispute_deposit = 1 OAS")

# work: min_executor_reputation = 0 (no barrier for testnet)
if "work" in app:
    app["work"]["params"]["min_executor_reputation"] = 0
    changes.append("work.min_executor_reputation = 0")

# reputation: feedback_cooldown = 60s (faster testing, default is 3600s)
if "reputation" in app:
    app["reputation"]["params"]["feedback_cooldown_seconds"] = 60
    changes.append("reputation.feedback_cooldown_seconds = 60")

# onboarding: pow_difficulty stays at code default (16)
# Halving Epoch 0 minimum is also 16, so effective = max(16, 16) = 16
# No patch needed — code default matches documentation

with open(genesis_path, "w") as f:
    json.dump(g, f, indent=2)

for c in changes:
    print(f"    {c}")
print("    Genesis patched.")
PYEOF

# Validate genesis
sudo -u "$SERVICE_USER" "$BINARY" genesis validate --home "$NODE_HOME" 2>/dev/null \
    && echo "    Genesis validated." \
    || { echo "ERROR: Genesis validation failed!"; exit 1; }

# ── 9. Configure node ──
echo "==> Configuring node..."
CONFIG="${NODE_HOME}/config/config.toml"
APP_TOML="${NODE_HOME}/config/app.toml"

PUBLIC_IP=$(curl -s -4 --connect-timeout 5 ifconfig.me || echo "47.93.32.88")

# config.toml
sed -i "s|laddr = \"tcp://127.0.0.1:26656\"|laddr = \"tcp://0.0.0.0:26656\"|" "$CONFIG"
sed -i "s|external_address = \"\"|external_address = \"${PUBLIC_IP}:26656\"|" "$CONFIG"
sed -i "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:26657\"|" "$CONFIG"
sed -i "s|prometheus = false|prometheus = true|" "$CONFIG"

# app.toml
sed -i 's/enable = false/enable = true/' "$APP_TOML"
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|' "$APP_TOML"
sed -i 's|minimum-gas-prices = ""|minimum-gas-prices = "0.025uoas"|' "$APP_TOML"
echo "    Node configured."

# Fix ownership
chown -R "$SERVICE_USER:$SERVICE_USER" "$NODE_HOME"

# ── 10. Start services ──
echo "==> Starting services..."
systemctl start oasyced
sleep 3

if ! systemctl is-active --quiet oasyced; then
    echo "ERROR: oasyced failed to start."
    echo "Check: journalctl -u oasyced -n 50"
    exit 1
fi

systemctl start oasyce-faucet
sleep 1

# ── 11. Print results ──
NODE_ID=$(sudo -u "$SERVICE_USER" "$BINARY" tendermint show-node-id --home "$NODE_HOME" 2>/dev/null || echo "UNKNOWN")
PEER_STRING="${NODE_ID}@${PUBLIC_IP}:26656"

echo ""
echo "============================================"
echo "  Testnet Reset Complete!"
echo "============================================"
echo ""
echo "  Chain ID:     $CHAIN_ID"
echo "  Version:      $($BINARY version 2>/dev/null || echo 'unknown')"
echo "  Node ID:      $NODE_ID"
echo "  Peer string:  $PEER_STRING"
echo ""
echo "  Accounts:"
echo "    Validator:  $VAL_ADDR"
echo "    Faucet:     $FAUCET_ADDR"
echo ""
echo "  Endpoints:"
echo "    RPC:     http://${PUBLIC_IP}:26657"
echo "    REST:    http://${PUBLIC_IP}:1317"
echo "    gRPC:    ${PUBLIC_IP}:9090"
echo "    Faucet:  http://${PUBLIC_IP}:8080/faucet?address=oasyce1..."
echo ""
echo "  Verify parameters:"
echo "    curl -s ${PUBLIC_IP}:1317/oasyce/work/v1/params"
echo "    curl -s ${PUBLIC_IP}:1317/oasyce/datarights/v1/params"
echo "    curl -s ${PUBLIC_IP}:1317/oasyce/onboarding/v1/params"
echo "    curl -s ${PUBLIC_IP}:1317/oasyce/reputation/v1/params"
echo "    curl -s ${PUBLIC_IP}:1317/cosmos/gov/v1/params"
echo ""
echo "  UPDATE DOCS with new peer string: $PEER_STRING"
echo ""
