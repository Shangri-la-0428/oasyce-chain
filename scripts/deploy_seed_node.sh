#!/bin/bash
# =============================================================================
# Deploy Oasyce Seed Node on VPS (Ubuntu 22.04/24.04)
#
# One-command setup for a public testnet seed node.
# Run this on a fresh VPS (Vultr, Hetzner, etc.) — 4CPU 8GB 100GB SSD minimum.
#
# Usage:
#   curl -sL <raw-url> | bash
#   # or:
#   bash scripts/deploy_seed_node.sh
#
# What it does:
#   1. Installs Go 1.22
#   2. Creates oasyce service user
#   3. Builds oasyced from source
#   4. Generates genesis (seed validator + faucet)
#   5. Configures node for public access
#   6. Sets up systemd service
#   7. Starts the node
#
# After running, you'll get:
#   - Node ID + peer string for other nodes to connect
#   - Faucet mnemonic (save it!)
#   - Validator mnemonic (save it!)
# =============================================================================
set -euo pipefail

# ── Configuration ──
CHAIN_ID="${CHAIN_ID:-oasyce-testnet-1}"
GO_VERSION="1.22.10"
REPO_URL="https://github.com/Shangri-la-0428/oasyce-chain.git"
BRANCH="${BRANCH:-main}"
SERVICE_USER="oasyce"
INSTALL_DIR="/opt/oasyce"
HOME_DIR="/home/${SERVICE_USER}"
NODE_HOME="${HOME_DIR}/.oasyced"
BINARY="${INSTALL_DIR}/bin/oasyced"
DENOM="uoas"

# Genesis allocation
VALIDATOR_BALANCE=1000000   # 1M OAS
VALIDATOR_STAKE=500000      # 500K OAS staked
FAUCET_BALANCE=50000000     # 50M OAS

echo "============================================"
echo "  Oasyce Seed Node — VPS Deployment"
echo "============================================"
echo ""
echo "  Chain ID:    $CHAIN_ID"
echo "  Go version:  $GO_VERSION"
echo "  Install dir: $INSTALL_DIR"
echo "  Node home:   $NODE_HOME"
echo ""

# ── Must be root ──
if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: Run as root (sudo bash deploy_seed_node.sh)"
    exit 1
fi

# ── Check OS ──
if [ ! -f /etc/os-release ]; then
    echo "ERROR: Only Ubuntu/Debian supported."
    exit 1
fi
source /etc/os-release
echo "==> Detected OS: $PRETTY_NAME"

# ══════════════════════════════════════════════════════════════════════════════
# 1. System dependencies
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Installing system dependencies..."
apt-get update -qq
apt-get install -y -qq git curl wget build-essential jq > /dev/null

# ══════════════════════════════════════════════════════════════════════════════
# 2. Install Go
# ══════════════════════════════════════════════════════════════════════════════
if command -v go &>/dev/null && go version | grep -q "go${GO_VERSION}"; then
    echo "==> Go ${GO_VERSION} already installed."
else
    echo "==> Installing Go ${GO_VERSION}..."
    ARCH=$(dpkg --print-architecture)
    GO_TAR="go${GO_VERSION}.linux-${ARCH}.tar.gz"
    wget -q "https://go.dev/dl/${GO_TAR}" -O /tmp/${GO_TAR}
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/${GO_TAR}
    rm /tmp/${GO_TAR}
    echo "    Go installed."
fi

export PATH="/usr/local/go/bin:$PATH"
go version

# ══════════════════════════════════════════════════════════════════════════════
# 3. Create service user
# ══════════════════════════════════════════════════════════════════════════════
if id "$SERVICE_USER" &>/dev/null; then
    echo "==> User '${SERVICE_USER}' already exists."
else
    echo "==> Creating user '${SERVICE_USER}'..."
    useradd -m -s /bin/bash "$SERVICE_USER"
    echo "    User created."
fi

# ══════════════════════════════════════════════════════════════════════════════
# 4. Build binary
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Cloning and building oasyced..."
mkdir -p "${INSTALL_DIR}/bin"

if [ -d "${INSTALL_DIR}/src" ]; then
    echo "    Updating existing source..."
    cd "${INSTALL_DIR}/src"
    git fetch origin
    git checkout "$BRANCH"
    git pull origin "$BRANCH"
else
    git clone --depth 1 -b "$BRANCH" "$REPO_URL" "${INSTALL_DIR}/src"
    cd "${INSTALL_DIR}/src"
fi

# Determine version from git tag or branch
VERSION=$(git describe --tags --always 2>/dev/null || echo "testnet-1")

CGO_ENABLED=0 go build \
    -ldflags "-X github.com/cosmos/cosmos-sdk/version.Name=oasyce \
              -X github.com/cosmos/cosmos-sdk/version.AppName=oasyced \
              -X github.com/cosmos/cosmos-sdk/version.Version=${VERSION}" \
    -o "$BINARY" ./cmd/oasyced

chmod +x "$BINARY"
echo "    Built: $($BINARY version 2>/dev/null || echo $VERSION)"

# Symlink for convenience
ln -sf "$BINARY" /usr/local/bin/oasyced

# ══════════════════════════════════════════════════════════════════════════════
# 5. Initialize node + generate genesis
# ══════════════════════════════════════════════════════════════════════════════
SECRETS_DIR="${HOME_DIR}/secrets"
mkdir -p "$SECRETS_DIR"

if [ -f "${NODE_HOME}/config/genesis.json" ]; then
    echo "==> Node already initialized. Skipping genesis generation."
    echo "    To re-initialize: rm -rf ${NODE_HOME} && re-run this script."
else
    echo "==> Initializing node..."

    # Init as service user
    sudo -u "$SERVICE_USER" "$BINARY" init "oasyce-seed-0" \
        --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null

    # Create validator key
    echo "==> Creating seed validator key..."
    VAL_JSON=$(sudo -u "$SERVICE_USER" "$BINARY" keys add seed-validator \
        --dry-run --output json 2>&1)
    VAL_ADDR=$(echo "$VAL_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
    VAL_MNEMONIC=$(echo "$VAL_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")

    # Import validator key
    echo "$VAL_MNEMONIC" | sudo -u "$SERVICE_USER" "$BINARY" keys add seed-validator \
        --recover --keyring-backend test --home "$NODE_HOME" 2>/dev/null
    echo "    Validator: $VAL_ADDR"

    # Create faucet key
    echo "==> Creating faucet key..."
    FAUCET_JSON=$(sudo -u "$SERVICE_USER" "$BINARY" keys add faucet \
        --dry-run --output json 2>&1)
    FAUCET_ADDR=$(echo "$FAUCET_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
    FAUCET_MNEMONIC=$(echo "$FAUCET_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['mnemonic'])")

    # Import faucet key
    echo "$FAUCET_MNEMONIC" | sudo -u "$SERVICE_USER" "$BINARY" keys add faucet \
        --recover --keyring-backend test --home "$NODE_HOME" 2>/dev/null
    echo "    Faucet: $FAUCET_ADDR"

    # Save mnemonics
    cat > "${SECRETS_DIR}/seed-validator.mnemonic" << EOF
# Oasyce Seed Validator — $CHAIN_ID
# Address: $VAL_ADDR
# KEEP THIS SECURE
$VAL_MNEMONIC
EOF

    cat > "${SECRETS_DIR}/faucet.mnemonic" << EOF
# Oasyce Faucet — $CHAIN_ID
# Address: $FAUCET_ADDR
# KEEP THIS SECURE
$FAUCET_MNEMONIC
EOF
    chmod 600 "${SECRETS_DIR}"/*.mnemonic
    chown -R "$SERVICE_USER:$SERVICE_USER" "$SECRETS_DIR"

    # Genesis accounts
    echo "==> Adding genesis accounts..."
    VAL_UOAS=$((VALIDATOR_BALANCE * 1000000))
    FAUCET_UOAS=$((FAUCET_BALANCE * 1000000))

    sudo -u "$SERVICE_USER" "$BINARY" genesis add-genesis-account "$VAL_ADDR" \
        "${VAL_UOAS}${DENOM}" --home "$NODE_HOME"
    sudo -u "$SERVICE_USER" "$BINARY" genesis add-genesis-account "$FAUCET_ADDR" \
        "${FAUCET_UOAS}${DENOM}" --home "$NODE_HOME"

    # Gentx
    echo "==> Creating validator gentx..."
    STAKE_UOAS=$((VALIDATOR_STAKE * 1000000))
    sudo -u "$SERVICE_USER" "$BINARY" genesis gentx seed-validator "${STAKE_UOAS}${DENOM}" \
        --chain-id "$CHAIN_ID" \
        --moniker "oasyce-seed-0" \
        --commission-rate "0.10" \
        --commission-max-rate "0.20" \
        --commission-max-change-rate "0.01" \
        --min-self-delegation "1" \
        --keyring-backend test \
        --home "$NODE_HOME" 2>/dev/null

    sudo -u "$SERVICE_USER" "$BINARY" genesis collect-gentxs --home "$NODE_HOME" 2>/dev/null

    # Patch genesis
    echo "==> Patching genesis parameters..."
    GENESIS="${NODE_HOME}/config/genesis.json"
    python3 << PYEOF
import json
with open("${GENESIS}") as f:
    g = json.load(f)
if "oasyce_capability" in g["app_state"]:
    g["app_state"]["oasyce_capability"]["params"]["min_provider_stake"] = {"denom": "uoas", "amount": "0"}
if "datarights" in g["app_state"]:
    g["app_state"]["datarights"]["params"]["dispute_deposit"] = {"denom": "uoas", "amount": "1000000"}
if "work" in g["app_state"]:
    g["app_state"]["work"]["params"]["min_executor_reputation"] = 0
if "onboarding" in g["app_state"]:
    g["app_state"]["onboarding"]["params"]["pow_difficulty"] = 8
with open("${GENESIS}", "w") as f:
    json.dump(g, f, indent=2)
print("    Genesis patched.")
PYEOF

    # Validate
    sudo -u "$SERVICE_USER" "$BINARY" genesis validate --home "$NODE_HOME" 2>/dev/null \
        && echo "    Genesis validated." \
        || echo "    WARNING: Genesis validation issue."

    # Copy genesis for distribution
    cp "$GENESIS" "${HOME_DIR}/genesis.json"
    chown "$SERVICE_USER:$SERVICE_USER" "${HOME_DIR}/genesis.json"
fi

# ══════════════════════════════════════════════════════════════════════════════
# 6. Configure node
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Configuring node..."
CONFIG="${NODE_HOME}/config/config.toml"
APP_TOML="${NODE_HOME}/config/app.toml"

# Detect public IP
PUBLIC_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || echo "UNKNOWN")
echo "    Public IP: $PUBLIC_IP"

# config.toml — P2P on all interfaces + external address
sed -i "s|laddr = \"tcp://127.0.0.1:26656\"|laddr = \"tcp://0.0.0.0:26656\"|" "$CONFIG"
sed -i "s|external_address = \"\"|external_address = \"${PUBLIC_IP}:26656\"|" "$CONFIG"
# RPC on all interfaces (for external queries)
sed -i "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:26657\"|" "$CONFIG"
# Enable Prometheus
sed -i "s|prometheus = false|prometheus = true|" "$CONFIG"
echo "    config.toml configured."

# app.toml — REST API + gRPC on all interfaces
sed -i 's/enable = false/enable = true/' "$APP_TOML"
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|' "$APP_TOML"
sed -i 's|minimum-gas-prices = ""|minimum-gas-prices = "0.025uoas"|' "$APP_TOML"
echo "    app.toml configured."

# Fix ownership
chown -R "$SERVICE_USER:$SERVICE_USER" "$NODE_HOME"

# ══════════════════════════════════════════════════════════════════════════════
# 7. Firewall
# ══════════════════════════════════════════════════════════════════════════════
if command -v ufw &>/dev/null; then
    echo "==> Configuring firewall..."
    ufw allow 22/tcp comment "SSH" > /dev/null 2>&1
    ufw allow 26656/tcp comment "Oasyce P2P" > /dev/null 2>&1
    ufw allow 26657/tcp comment "Oasyce RPC" > /dev/null 2>&1
    ufw allow 1317/tcp comment "Oasyce REST" > /dev/null 2>&1
    ufw allow 9090/tcp comment "Oasyce gRPC" > /dev/null 2>&1
    ufw allow 26660/tcp comment "Prometheus" > /dev/null 2>&1
    ufw allow 8080/tcp comment "Oasyce Faucet" > /dev/null 2>&1
    ufw --force enable > /dev/null 2>&1
    echo "    Firewall rules set."
else
    echo "    ufw not found — configure firewall manually."
fi

# ══════════════════════════════════════════════════════════════════════════════
# 8. Systemd service
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Setting up systemd service..."

cat > /etc/systemd/system/oasyced.service << EOF
[Unit]
Description=Oasyce Chain Node
After=network-online.target
Wants=network-online.target

[Service]
User=${SERVICE_USER}
ExecStart=${BINARY} start --home ${NODE_HOME} --minimum-gas-prices 0.025uoas --api.enable=true --api.address=tcp://0.0.0.0:1317 --grpc.address=0.0.0.0:9090
Restart=always
RestartSec=3
LimitNOFILE=65535

# Hardening
ProtectSystem=full
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable oasyced

# ══════════════════════════════════════════════════════════════════════════════
# 8b. Journal log rotation
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Configuring journal log limits..."
mkdir -p /etc/systemd/journald.conf.d
cat > /etc/systemd/journald.conf.d/oasyced.conf << EOF
[Journal]
SystemMaxUse=500M
SystemMaxFileSize=50M
MaxRetentionSec=7day
Compress=yes
EOF
systemctl restart systemd-journald
echo "    Journal: max 500MB, 7-day retention."

# ══════════════════════════════════════════════════════════════════════════════
# 8c. Faucet service
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Setting up faucet service..."

cat > /etc/systemd/system/oasyce-faucet.service << EOF
[Unit]
Description=Oasyce Testnet Faucet
After=oasyced.service
Requires=oasyced.service

[Service]
User=${SERVICE_USER}
Environment=FAUCET_PORT=8080
Environment=FAUCET_AMOUNT=100
Environment=CHAIN_ID=${CHAIN_ID}
Environment=OASYCE_HOME=${NODE_HOME}
Environment=FAUCET_KEY=faucet
Environment=FAUCET_RATE_FILE=${HOME_DIR}/.faucet_rate.json
ExecStart=/usr/bin/python3 ${INSTALL_DIR}/src/scripts/faucet_server.py
Restart=always
RestartSec=5
LimitNOFILE=4096
ProtectSystem=full
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable oasyce-faucet
systemctl start oasyce-faucet
echo "    Faucet running on :8080"

# ══════════════════════════════════════════════════════════════════════════════
# 9. Start node
# ══════════════════════════════════════════════════════════════════════════════
echo "==> Starting node..."
systemctl start oasyced

# Wait for node to start
sleep 3

if systemctl is-active --quiet oasyced; then
    NODE_ID=$(sudo -u "$SERVICE_USER" "$BINARY" tendermint show-node-id --home "$NODE_HOME" 2>/dev/null || echo "UNKNOWN")
    PEER_STRING="${NODE_ID}@${PUBLIC_IP}:26656"

    echo ""
    echo "============================================"
    echo "  Seed Node Running!"
    echo "============================================"
    echo ""
    echo "  Chain ID:     $CHAIN_ID"
    echo "  Node ID:      $NODE_ID"
    echo "  Public IP:    $PUBLIC_IP"
    echo ""
    echo "  Peer string (share this with validators):"
    echo "    $PEER_STRING"
    echo ""
    echo "  Endpoints:"
    echo "    P2P:         ${PUBLIC_IP}:26656"
    echo "    RPC:         http://${PUBLIC_IP}:26657"
    echo "    REST API:    http://${PUBLIC_IP}:1317"
    echo "    gRPC:        ${PUBLIC_IP}:9090"
    echo "    Faucet:      http://${PUBLIC_IP}:8080/faucet?address=oasyce1..."
    echo "    Prometheus:  http://${PUBLIC_IP}:26660"
    echo ""
    echo "  Genesis file:  ${HOME_DIR}/genesis.json"
    echo "  Secrets:       ${SECRETS_DIR}/"
    echo ""
    echo "  Commands:"
    echo "    journalctl -u oasyced -f              # watch logs"
    echo "    oasyced status --home ${NODE_HOME}     # check sync"
    echo "    systemctl stop oasyced                 # stop"
    echo "    systemctl restart oasyced              # restart"
    echo ""
    echo "  For other nodes to join:"
    echo "    GENESIS_URL=http://${PUBLIC_IP}:1317/cosmos/base/tendermint/v1beta1/node_info"
    echo "    SEED_NODE=${PEER_STRING}"
    echo "    bash scripts/join_testnet.sh"
    echo ""
    echo "  IMPORTANT: Save the mnemonics in ${SECRETS_DIR}/ securely!"
    echo ""
else
    echo ""
    echo "ERROR: Node failed to start."
    echo "Check logs: journalctl -u oasyced -n 50"
    exit 1
fi
