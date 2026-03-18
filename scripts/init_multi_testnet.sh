#!/bin/bash
# Initialize a 4-validator local testnet for development.
set -euo pipefail

CHAIN_ID="oasyce-localnet-1"
DENOM="uoas"
NUM_VALIDATORS=4
BASE_PORT=26600       # each node offsets by 100
BASE_RPC_PORT=26657   # node0 gets default, rest offset
BASE_P2P_PORT=26656
BASE_API_PORT=1317
BASE_GRPC_PORT=9090

BINARY="oasyced"
BASE_DIR="$HOME/.oasyce-localnet"

echo "==> Cleaning previous state..."
rm -rf "$BASE_DIR"

# ---------------------------------------------------------------------------
# Step 1: Create directories, keys, and init each node.
# ---------------------------------------------------------------------------
NODE_IDS=()
VAL_ADDRS=()

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  MONIKER="validator-${i}"

  echo "==> Initializing node${i} (${MONIKER})..."
  $BINARY init "$MONIKER" --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null

  # Create validator key.
  $BINARY keys add "val${i}" --keyring-backend test --home "$NODE_HOME" 2>/dev/null

  # Capture the node ID.
  NODE_ID=$($BINARY comet show-node-id --home "$NODE_HOME")
  NODE_IDS+=("$NODE_ID")

  # Capture the validator address.
  VAL_ADDR=$($BINARY keys show "val${i}" -a --keyring-backend test --home "$NODE_HOME")
  VAL_ADDRS+=("$VAL_ADDR")
done

# ---------------------------------------------------------------------------
# Step 2: Use node0's genesis as the canonical genesis.
# Add all validator accounts and gentxs.
# ---------------------------------------------------------------------------
GENESIS_HOME="$BASE_DIR/node0"

echo "==> Adding genesis accounts..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  VAL_ADDR="${VAL_ADDRS[$i]}"

  # Each validator gets 1M OAS = 100_000_000_000_000 uoas.
  $BINARY genesis add-genesis-account "$VAL_ADDR" "100000000000000${DENOM}" --home "$GENESIS_HOME"
done

# Copy the canonical genesis to all nodes so gentx validation succeeds.
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
  cp "$GENESIS_HOME/config/genesis.json" "$BASE_DIR/node${i}/config/genesis.json"
done

echo "==> Creating gentxs..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  # Each validator stakes 100K OAS = 10_000_000_000_000 uoas.
  $BINARY genesis gentx "val${i}" "10000000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test \
    --home "$NODE_HOME" 2>/dev/null
done

# ---------------------------------------------------------------------------
# Step 3: Collect all gentxs into node0, then distribute.
# ---------------------------------------------------------------------------
echo "==> Collecting gentxs..."
# Copy gentxs from all nodes into node0's gentx directory.
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
  cp "$BASE_DIR/node${i}/config/gentx/"*.json "$GENESIS_HOME/config/gentx/" 2>/dev/null || true
done

$BINARY genesis collect-gentxs --home "$GENESIS_HOME"

# Copy the final genesis to all other nodes.
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
  cp "$GENESIS_HOME/config/genesis.json" "$BASE_DIR/node${i}/config/genesis.json"
done

# ---------------------------------------------------------------------------
# Step 4: Configure persistent_peers and ports for each node.
# ---------------------------------------------------------------------------
echo "==> Configuring peers and ports..."

# Build the persistent_peers string: nodeID@127.0.0.1:p2pPort
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

  # Filter out this node's own address from the peers list.
  NODE_PEERS=$(echo "$PEERS" | sed "s/${NODE_IDS[$i]}@127.0.0.1:${P2P_PORT}//g" | sed 's/^,//;s/,$//;s/,,/,/g')

  # Update config.toml: laddr for p2p, rpc, and persistent_peers.
  sed -i.bak "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:${P2P_PORT}\"|" "$CONFIG_FILE"
  sed -i.bak "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://127.0.0.1:${RPC_PORT}\"|" "$CONFIG_FILE"
  sed -i.bak "s|persistent_peers = \"\"|persistent_peers = \"${NODE_PEERS}\"|" "$CONFIG_FILE"

  # Update app.toml: API and gRPC ports.
  sed -i.bak "s|address = \"tcp://localhost:1317\"|address = \"tcp://localhost:${API_PORT}\"|" "$APP_CONFIG"
  sed -i.bak "s|address = \"localhost:9090\"|address = \"localhost:${GRPC_PORT}\"|" "$APP_CONFIG"

  # Clean up backup files from sed.
  rm -f "$CONFIG_FILE.bak" "$APP_CONFIG.bak"
done

# ---------------------------------------------------------------------------
# Step 5: Print start commands.
# ---------------------------------------------------------------------------
echo ""
echo "=== 4-Validator Local Testnet Initialized ==="
echo ""
echo "Chain ID: $CHAIN_ID"
echo "Denom:    $DENOM"
echo ""
echo "Start each validator in a separate terminal:"
echo ""
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  RPC_PORT=$((BASE_RPC_PORT + i * 100))
  echo "  # Validator ${i} (RPC :${RPC_PORT})"
  echo "  $BINARY start --home $BASE_DIR/node${i}"
  echo ""
done
