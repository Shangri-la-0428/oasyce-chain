#!/bin/bash
# Initialize a 2-validator LAN testnet for internal testing.
#
# Node 0: Operator (you)    — P2P=26656  RPC=26657  API=1317  gRPC=9090
# Node 1: Seed user          — P2P=26756  RPC=26757  API=1417  gRPC=9190
#
# After init, patches genesis to lower thresholds for testing.
#
set -euo pipefail

CHAIN_ID="oasyce-lantest-1"
DENOM="uoas"
NUM_VALIDATORS=2
BINARY="${BINARY:-./build/oasyced}"
BASE_DIR="$HOME/.oasyce-lantest"

BASE_P2P_PORT=26656
BASE_RPC_PORT=26657
BASE_API_PORT=1317
BASE_GRPC_PORT=9090
BASE_GRPC_WEB_PORT=9091

echo "==> Cleaning previous state..."
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# Step 1: Init nodes + generate keys
declare -a NODE_IDS
declare -a VAL_ADDRS
declare -a MNEMONICS

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  MONIKER="validator-${i}"

  echo "==> Initializing node${i} (${MONIKER})..."
  $BINARY init "$MONIKER" --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null

  KEY_JSON=$($BINARY keys add "val${i}" --dry-run --output json 2>&1)
  ADDR=$(echo "$KEY_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
  MNEMONIC=$(echo "$KEY_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")

  echo "$MNEMONIC" | $BINARY keys add "val${i}" --recover --keyring-backend test --home "$NODE_HOME" 2>/dev/null

  NODE_ID=$($BINARY comet show-node-id --home "$NODE_HOME" 2>&1)

  NODE_IDS+=("$NODE_ID")
  VAL_ADDRS+=("$ADDR")
  MNEMONICS+=("$MNEMONIC")

  echo "    Address: $ADDR"
  echo "    Node ID: $NODE_ID"
done

# Step 2: Build genesis with generous test allocations
GENESIS_HOME="$BASE_DIR/node0"

echo ""
echo "==> Adding genesis accounts (100K OAS each)..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  # 100,000 OAS = 100_000_000_000 uoas
  $BINARY genesis add-genesis-account "${VAL_ADDRS[$i]}" "100000000000${DENOM}" --home "$GENESIS_HOME"
  echo "    Added ${VAL_ADDRS[$i]} with 100,000 OAS"
done

# Copy genesis to node1 before gentx
cp "$GENESIS_HOME/config/genesis.json" "$BASE_DIR/node1/config/genesis.json"

echo "==> Creating gentxs..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  # Each validator stakes 10,000 OAS = 10_000_000_000 uoas
  $BINARY genesis gentx "val${i}" "10000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test \
    --home "$NODE_HOME" 2>/dev/null
  echo "    Created gentx for val${i}"
done

# Step 3: Collect gentxs
echo "==> Collecting gentxs..."
cp "$BASE_DIR/node1/config/gentx/"*.json "$GENESIS_HOME/config/gentx/" 2>/dev/null || true
$BINARY genesis collect-gentxs --home "$GENESIS_HOME" 2>/dev/null

# Step 4: Patch genesis for testnet-friendly params
echo "==> Patching genesis parameters..."
GENESIS_FILE="$GENESIS_HOME/config/genesis.json"
python3 -c "
import json
g = json.load(open('$GENESIS_FILE'))
# Lower capability min_provider_stake to 0
g['app_state']['oasyce_capability']['params']['min_provider_stake'] = {'denom':'uoas','amount':'0'}
# Lower datarights dispute_deposit to 10 OAS
g['app_state']['datarights']['params']['dispute_deposit'] = {'denom':'uoas','amount':'10000000'}
# Lower onboarding pow_difficulty for faster testing
g['app_state']['onboarding']['params']['pow_difficulty'] = 8
json.dump(g, open('$GENESIS_FILE','w'), indent=2)
print('    min_provider_stake: 0 uoas')
print('    dispute_deposit: 10 OAS')
print('    pow_difficulty: 8 bits (~256 attempts)')
"

# Distribute final genesis to node1
cp "$GENESIS_FILE" "$BASE_DIR/node1/config/genesis.json"

# Step 5: Configure ports and peers
echo "==> Configuring peers and ports..."

PEERS=""
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  P2P_PORT=$((BASE_P2P_PORT + i * 100))
  if [ -n "$PEERS" ]; then PEERS="${PEERS},"; fi
  PEERS="${PEERS}${NODE_IDS[$i]}@127.0.0.1:${P2P_PORT}"
done

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  CONFIG_FILE="$NODE_HOME/config/config.toml"
  APP_CONFIG="$NODE_HOME/config/app.toml"

  P2P_PORT=$((BASE_P2P_PORT + i * 100))
  RPC_PORT=$((BASE_RPC_PORT + i * 100))
  API_PORT=$((BASE_API_PORT + i * 100))
  GRPC_PORT=$((BASE_GRPC_PORT + i * 100))
  GRPC_WEB_PORT=$((BASE_GRPC_WEB_PORT + i * 100))

  NODE_PEERS=$(echo "$PEERS" | sed "s/${NODE_IDS[$i]}@127.0.0.1:${P2P_PORT}//g" | sed 's/^,//;s/,$//;s/,,/,/g')
  PPROF_PORT=$((6060 + i))
  PROMETHEUS_PORT=$((26660 + i * 100))

  # Patch config.toml + app.toml via python3 (macOS sed is unreliable for TOML)
  python3 << PYEOF
import re

# --- config.toml ---
cfg = open('$CONFIG_FILE').read()
cfg = cfg.replace('laddr = "tcp://0.0.0.0:26656"', 'laddr = "tcp://0.0.0.0:${P2P_PORT}"', 1)
cfg = cfg.replace('laddr = "tcp://127.0.0.1:26657"', 'laddr = "tcp://127.0.0.1:${RPC_PORT}"', 1)
cfg = re.sub(r'^persistent_peers = ".*?"', 'persistent_peers = "${NODE_PEERS}"', cfg, count=1, flags=re.MULTILINE)
cfg = cfg.replace('pprof_laddr = "localhost:6060"', 'pprof_laddr = "localhost:${PPROF_PORT}"', 1)
cfg = cfg.replace('prometheus_listen_addr = ":26660"', 'prometheus_listen_addr = ":${PROMETHEUS_PORT}"', 1)
open('$CONFIG_FILE', 'w').write(cfg)

# --- app.toml ---
app = open('$APP_CONFIG').read()
app = app.replace('address = "tcp://localhost:1317"', 'address = "tcp://localhost:${API_PORT}"', 1)
app = app.replace('address = "localhost:9090"', 'address = "localhost:${GRPC_PORT}"', 1)
app = app.replace('address = "localhost:9091"', 'address = "localhost:${GRPC_WEB_PORT}"', 1)
# Enable REST API (first enable = false after [api])
app = re.sub(r'(\[api\][^\[]*?)enable = false', r'\1enable = true', app, count=1)
open('$APP_CONFIG', 'w').write(app)
PYEOF

  echo "    node${i}: P2P=${P2P_PORT} RPC=${RPC_PORT} API=${API_PORT} gRPC=${GRPC_PORT}"
done

# Step 6: Save mnemonics
MNEMONIC_FILE="$BASE_DIR/mnemonics.txt"
echo "# LAN Test Validator Mnemonics" > "$MNEMONIC_FILE"
echo "# WARNING: For testing only!" >> "$MNEMONIC_FILE"
echo "" >> "$MNEMONIC_FILE"
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  echo "val${i} (${VAL_ADDRS[$i]}):" >> "$MNEMONIC_FILE"
  echo "  ${MNEMONICS[$i]}" >> "$MNEMONIC_FILE"
  echo "" >> "$MNEMONIC_FILE"
done

echo ""
echo "============================================"
echo "  2-Validator LAN Testnet Initialized!"
echo "  Chain ID:    $CHAIN_ID"
echo "  Base dir:    $BASE_DIR"
echo "  Accounts:    100,000 OAS each"
echo "  Staked:      10,000 OAS each"
echo "  PoW diff:    8 bits (fast)"
echo "  Dispute dep: 10 OAS"
echo "============================================"
echo ""
echo "Start nodes:"
echo "  Node 0 (you):        $BINARY start --home $BASE_DIR/node0 --minimum-gas-prices 0uoas"
echo "  Node 1 (seed user):  $BINARY start --home $BASE_DIR/node1 --minimum-gas-prices 0uoas"
echo ""
echo "Mnemonics saved to: $MNEMONIC_FILE"
echo "Copy node1 home to seed user's machine: scp -r $BASE_DIR/node1 user@ip:~/.oasyce-lantest/node1"
echo ""
