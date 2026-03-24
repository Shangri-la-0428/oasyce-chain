#!/bin/bash
# =============================================================================
# One-click join Oasyce Testnet
#
# Works on macOS, Windows (Git Bash/WSL), and Linux.
# Uses Docker — no firewall configuration needed.
#
# Usage:
#   bash scripts/join_testnet.sh
#
# Prerequisites:
#   - Docker Desktop (Mac/Windows) or Docker Engine (Linux)
#   - Internet connection
#
# What it does:
#   1. Builds the Docker image (or pulls if available)
#   2. Initializes a fresh node
#   3. Downloads genesis.json
#   4. Configures seed peer
#   5. Starts the node
# =============================================================================
set -euo pipefail

# ── Configuration (edit these for your testnet) ──
CHAIN_ID="${CHAIN_ID:-oasyce-testnet-1}"
MONIKER="${MONIKER:-oasyce-node-$(openssl rand -hex 3)}"
SEED_NODE="${SEED_NODE:-}"       # e.g., "abc123@1.2.3.4:26656"
GENESIS_URL="${GENESIS_URL:-}"   # e.g., "https://github.com/.../genesis.json"
IMAGE="oasyce/chain:latest"
DATA_DIR="$HOME/.oasyce-docker"

echo "============================================"
echo "  Oasyce Testnet — One-Click Join"
echo "============================================"
echo ""
echo "  Chain ID:  $CHAIN_ID"
echo "  Moniker:   $MONIKER"
echo "  Data dir:  $DATA_DIR"
echo ""

# ── Check Docker ──
if ! command -v docker &>/dev/null; then
    echo "ERROR: Docker not found."
    echo ""
    echo "Install Docker Desktop:"
    echo "  macOS:   https://docs.docker.com/desktop/install/mac-install/"
    echo "  Windows: https://docs.docker.com/desktop/install/windows-install/"
    echo "  Linux:   https://docs.docker.com/engine/install/"
    exit 1
fi

if ! docker info &>/dev/null; then
    echo "ERROR: Docker is installed but not running."
    echo "Please start Docker Desktop and try again."
    exit 1
fi

# ── Build image if not present ──
if ! docker image inspect "$IMAGE" &>/dev/null; then
    echo "==> Building Docker image (first time only)..."
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    REPO_DIR="$(dirname "$SCRIPT_DIR")"
    docker build -t "$IMAGE" "$REPO_DIR"
    echo "    Image built."
else
    echo "==> Docker image found."
fi

# ── Initialize node ──
mkdir -p "$DATA_DIR"

if [ ! -f "$DATA_DIR/config/genesis.json" ]; then
    echo "==> Initializing node..."
    docker run --rm -v "$DATA_DIR:/root/.oasyced" "$IMAGE" \
        oasyced init "$MONIKER" --chain-id "$CHAIN_ID" --home /root/.oasyced 2>/dev/null
    echo "    Node initialized: $MONIKER"
else
    echo "==> Node already initialized (using existing config)."
fi

# ── Download genesis ──
if [ -n "$GENESIS_URL" ]; then
    echo "==> Downloading genesis.json..."
    curl -sL "$GENESIS_URL" -o "$DATA_DIR/config/genesis.json"
    echo "    Genesis downloaded."
elif [ -z "$SEED_NODE" ]; then
    echo ""
    echo "NOTE: No GENESIS_URL or SEED_NODE configured."
    echo "For local testing, the default genesis is fine."
    echo ""
    echo "For public testnet, re-run with:"
    echo "  GENESIS_URL=<url> SEED_NODE=<id@ip:26656> bash scripts/join_testnet.sh"
    echo ""
fi

# ── Configure peers ──
if [ -n "$SEED_NODE" ]; then
    echo "==> Configuring seed node: $SEED_NODE"
    CONFIG="$DATA_DIR/config/config.toml"
    if [ -f "$CONFIG" ]; then
        # Use portable sed (works on both macOS and Linux)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$SEED_NODE\"|" "$CONFIG"
        else
            sed -i "s|persistent_peers = \".*\"|persistent_peers = \"$SEED_NODE\"|" "$CONFIG"
        fi
        echo "    Peer configured."
    fi
fi

# ── Patch genesis params for testnet ──
echo "==> Patching genesis parameters..."
python3 << 'PYEOF' || true
import json, os
genesis_path = os.path.expanduser("$DATA_DIR/config/genesis.json")
try:
    with open(genesis_path) as f:
        g = json.load(f)
    changed = False
    if "oasyce_capability" in g.get("app_state", {}):
        g["app_state"]["oasyce_capability"]["params"]["min_provider_stake"] = {"denom": "uoas", "amount": "0"}
        changed = True
    if "onboarding" in g.get("app_state", {}):
        g["app_state"]["onboarding"]["params"]["pow_difficulty"] = 8
        changed = True
    if changed:
        with open(genesis_path, "w") as f:
            json.dump(g, f, indent=2)
        print("    Genesis patched.")
    else:
        print("    No patching needed.")
except Exception as e:
    print(f"    Skipping genesis patch: {e}")
PYEOF

# ── Enable REST API ──
APP_TOML="$DATA_DIR/config/app.toml"
if [ -f "$APP_TOML" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/enable = false/enable = true/' "$APP_TOML"
        sed -i '' 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
    else
        sed -i 's/enable = false/enable = true/' "$APP_TOML"
        sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
    fi
fi

# ── Configure P2P to listen on all interfaces ──
CONFIG="$DATA_DIR/config/config.toml"
if [ -f "$CONFIG" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's|laddr = "tcp://127.0.0.1:26656"|laddr = "tcp://0.0.0.0:26656"|' "$CONFIG"
    else
        sed -i 's|laddr = "tcp://127.0.0.1:26656"|laddr = "tcp://0.0.0.0:26656"|' "$CONFIG"
    fi
fi

# ── Start ──
echo "==> Starting node..."
echo ""

docker run -d \
    --name oasyce-node \
    --restart always \
    -p 26656:26656 \
    -p 26657:26657 \
    -p 1317:1317 \
    -p 9090:9090 \
    -v "$DATA_DIR:/root/.oasyced" \
    "$IMAGE" \
    oasyced start \
      --home /root/.oasyced \
      --minimum-gas-prices 0uoas \
      --api.enable=true \
      --api.address=tcp://0.0.0.0:1317 \
      --grpc.address=0.0.0.0:9090 \
    2>/dev/null && STARTED=true || STARTED=false

if [ "$STARTED" = true ]; then
    echo "============================================"
    echo "  Node started!"
    echo "============================================"
    echo ""
    echo "  Container:  oasyce-node"
    echo "  Data dir:   $DATA_DIR"
    echo ""
    echo "  Ports:"
    echo "    P2P:      localhost:26656"
    echo "    RPC:      localhost:26657"
    echo "    REST API: localhost:1317"
    echo "    gRPC:     localhost:9090"
    echo ""
    echo "  Commands:"
    echo "    docker logs -f oasyce-node          # watch logs"
    echo "    docker exec oasyce-node oasyced status   # check sync"
    echo "    docker stop oasyce-node              # stop"
    echo "    docker start oasyce-node             # restart"
    echo ""
    echo "  No firewall configuration needed — Docker handles port mapping."
else
    echo ""
    echo "Container 'oasyce-node' may already exist. To reset:"
    echo "  docker rm -f oasyce-node"
    echo "  bash scripts/join_testnet.sh"
fi
