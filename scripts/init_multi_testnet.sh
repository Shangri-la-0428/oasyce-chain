#!/bin/bash
# Initialize a 4-validator local testnet for development.
#
# Each validator gets its own home directory under ~/.oasyce-localnet/nodeN.
# Port allocation (offset +100 per node):
#   Node 0: P2P=26656  RPC=26657  API=1317  gRPC=9090  gRPC-web=9091
#   Node 1: P2P=26756  RPC=26757  API=1417  gRPC=9190  gRPC-web=9191
#   Node 2: P2P=26856  RPC=26857  API=1517  gRPC=9290  gRPC-web=9291
#   Node 3: P2P=26956  RPC=26957  API=1617  gRPC=9390  gRPC-web=9391
#
# Works around Cosmos SDK v0.50 keyring serialization bugs by using
# --dry-run + --recover with --keyring-backend test.
#
set -euo pipefail

CHAIN_ID="oasyce-localnet-1"
DENOM="uoas"
NUM_VALIDATORS=4
BINARY="${BINARY:-oasyced}"
BASE_DIR="$HOME/.oasyce-localnet"

# Base ports for node 0 (each subsequent node adds i*100)
BASE_P2P_PORT=26656
BASE_RPC_PORT=26657
BASE_API_PORT=1317
BASE_GRPC_PORT=9090
BASE_GRPC_WEB_PORT=9091

echo "==> Cleaning previous state..."
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# ---------------------------------------------------------------------------
# Step 1: Initialize each node, generate keys.
# ---------------------------------------------------------------------------
declare -a NODE_IDS
declare -a VAL_ADDRS
declare -a MNEMONICS

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  MONIKER="validator-${i}"

  echo "==> Initializing node${i} (${MONIKER})..."
  $BINARY init "$MONIKER" --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null

  # Generate key with --dry-run (avoids keyring serialization issues)
  KEY_JSON=$($BINARY keys add "val${i}" --dry-run --output json 2>&1)
  ADDR=$(echo "$KEY_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
  MNEMONIC=$(echo "$KEY_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")

  # Recover key into test keyring
  echo "$MNEMONIC" | $BINARY keys add "val${i}" --recover --keyring-backend test --home "$NODE_HOME" 2>/dev/null

  # Capture the node ID (CometBFT outputs to stderr, need 2>&1)
  NODE_ID=$($BINARY comet show-node-id --home "$NODE_HOME" 2>&1)

  NODE_IDS+=("$NODE_ID")
  VAL_ADDRS+=("$ADDR")
  MNEMONICS+=("$MNEMONIC")

  echo "    Address: $ADDR"
  echo "    Node ID: $NODE_ID"
done

# ---------------------------------------------------------------------------
# Step 2: Build genesis using node0 as the canonical source.
#   - Add all validator accounts.
#   - Copy genesis to all nodes so gentx validation succeeds.
#   - Create gentx on each node.
# ---------------------------------------------------------------------------
GENESIS_HOME="$BASE_DIR/node0"

echo ""
echo "==> Adding genesis accounts..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  VAL_ADDR="${VAL_ADDRS[$i]}"
  # Each validator gets 1M OAS = 100_000_000_000_000 uoas
  $BINARY genesis add-genesis-account "$VAL_ADDR" "100000000000000${DENOM}" --home "$GENESIS_HOME"
  echo "    Added ${VAL_ADDR}"
done

# Distribute the genesis with all accounts to every node before gentx.
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
  cp "$GENESIS_HOME/config/genesis.json" "$BASE_DIR/node${i}/config/genesis.json"
done

echo "==> Creating gentxs..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  # Each validator stakes 100K OAS = 10_000_000_000_000 uoas
  $BINARY genesis gentx "val${i}" "10000000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test \
    --home "$NODE_HOME" 2>/dev/null
  echo "    Created gentx for val${i}"
done

# ---------------------------------------------------------------------------
# Step 3: Collect all gentxs into node0, then distribute final genesis.
# ---------------------------------------------------------------------------
echo "==> Collecting gentxs..."
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
  cp "$BASE_DIR/node${i}/config/gentx/"*.json "$GENESIS_HOME/config/gentx/" 2>/dev/null || true
done

$BINARY genesis collect-gentxs --home "$GENESIS_HOME" 2>/dev/null

# Distribute final genesis to all nodes.
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
  cp "$GENESIS_HOME/config/genesis.json" "$BASE_DIR/node${i}/config/genesis.json"
done

# ---------------------------------------------------------------------------
# Step 4: Configure ports and persistent_peers for each node.
# ---------------------------------------------------------------------------
echo "==> Configuring peers and ports..."

# Build the full peers string: nodeID@127.0.0.1:p2pPort
PEERS=""
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  P2P_PORT=$((BASE_P2P_PORT + i * 100))
  if [ -n "$PEERS" ]; then
    PEERS="${PEERS},"
  fi
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

  # Filter out this node's own address from the peers list.
  NODE_PEERS=$(echo "$PEERS" | sed "s/${NODE_IDS[$i]}@127.0.0.1:${P2P_PORT}//g" | sed 's/^,//;s/,$//;s/,,/,/g')

  # -- config.toml --
  # P2P listen address
  sed -i.bak "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:${P2P_PORT}\"|" "$CONFIG_FILE"
  # RPC listen address
  sed -i.bak "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://127.0.0.1:${RPC_PORT}\"|" "$CONFIG_FILE"
  # Persistent peers (use python3 — collect-gentxs may pre-populate this field)
  python3 -c "
import re, pathlib
cfg = pathlib.Path('$CONFIG_FILE')
txt = cfg.read_text()
txt = re.sub(r'^persistent_peers = \".*?\"', 'persistent_peers = \"${NODE_PEERS}\"', txt, flags=re.MULTILINE)
cfg.write_text(txt)
"
  # Allow multiple nodes on same IP (required for localhost multi-node)
  python3 -c "
import re, pathlib
cfg = pathlib.Path('$CONFIG_FILE')
txt = cfg.read_text()
txt = re.sub(r'^allow_duplicate_ip = false', 'allow_duplicate_ip = true', txt, flags=re.MULTILINE)
txt = re.sub(r'^addr_book_strict = true', 'addr_book_strict = false', txt, flags=re.MULTILINE)
cfg.write_text(txt)
"
  # pprof listen address (avoid conflict, offset by node index)
  PPROF_PORT=$((6060 + i))
  sed -i.bak "s|pprof_laddr = \"localhost:6060\"|pprof_laddr = \"localhost:${PPROF_PORT}\"|" "$CONFIG_FILE"
  # Prometheus listen port (avoid conflict)
  PROMETHEUS_PORT=$((26660 + i * 100))
  sed -i.bak "s|prometheus_listen_addr = \":26660\"|prometheus_listen_addr = \":${PROMETHEUS_PORT}\"|" "$CONFIG_FILE"

  # -- app.toml --
  # Enable REST API (required for agent access)
  python3 -c "
import re, pathlib
cfg = pathlib.Path('$APP_CONFIG')
txt = cfg.read_text()
# Enable API in the [api] section (first 'enable = false' after [api])
txt = re.sub(r'(\[api\][^\[]*?)enable = false', r'\1enable = true', txt, count=1, flags=re.DOTALL)
cfg.write_text(txt)
"
  # API server address
  sed -i.bak "s|address = \"tcp://localhost:1317\"|address = \"tcp://localhost:${API_PORT}\"|" "$APP_CONFIG"
  # gRPC server address
  sed -i.bak "s|address = \"localhost:9090\"|address = \"localhost:${GRPC_PORT}\"|" "$APP_CONFIG"
  # gRPC-web server address
  sed -i.bak "s|address = \"localhost:9091\"|address = \"localhost:${GRPC_WEB_PORT}\"|" "$APP_CONFIG"
  # Set minimum gas prices (anti-spam)
  sed -i.bak 's|minimum-gas-prices = ""|minimum-gas-prices = "0.025uoas"|' "$APP_CONFIG"

  # Clean up backup files from macOS sed -i
  rm -f "$CONFIG_FILE.bak" "$APP_CONFIG.bak"

  echo "    node${i}: P2P=${P2P_PORT} RPC=${RPC_PORT} API=${API_PORT} gRPC=${GRPC_PORT}"
done

# ---------------------------------------------------------------------------
# Step 5: Save mnemonics for reference.
# ---------------------------------------------------------------------------
MNEMONIC_FILE="$BASE_DIR/mnemonics.txt"
echo "# Validator mnemonics for oasyce-localnet" > "$MNEMONIC_FILE"
echo "# WARNING: For local development only. Never use these on a real network." >> "$MNEMONIC_FILE"
echo "" >> "$MNEMONIC_FILE"
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  echo "val${i} (${VAL_ADDRS[$i]}):" >> "$MNEMONIC_FILE"
  echo "  ${MNEMONICS[$i]}" >> "$MNEMONIC_FILE"
  echo "" >> "$MNEMONIC_FILE"
done
echo "==> Mnemonics saved to $MNEMONIC_FILE"

# ---------------------------------------------------------------------------
# Step 6: Print summary.
# ---------------------------------------------------------------------------
GENTX_COUNT=$(python3 -c "import json; g=json.load(open('$GENESIS_HOME/config/genesis.json')); print(len(g['app_state']['genutil']['gen_txs']))")

echo ""
echo "============================================"
echo "  4-Validator Local Testnet Initialized!"
echo "  Chain ID:    $CHAIN_ID"
echo "  Denom:       $DENOM"
echo "  Gentxs:      $GENTX_COUNT"
echo "  Base dir:    $BASE_DIR"
echo "  Keyring:     test (no password)"
echo "============================================"
echo ""
echo "Start all nodes:"
echo "  bash scripts/start_testnet.sh"
echo ""
echo "Or start individually:"
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  RPC_PORT=$((BASE_RPC_PORT + i * 100))
  echo "  $BINARY start --home $BASE_DIR/node${i}  # RPC :${RPC_PORT}"
done
echo ""
