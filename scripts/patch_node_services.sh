#!/bin/bash
# =============================================================================
# Patch existing Oasyce seed node with logrotate + faucet service
#
# Run this on a VPS that already has oasyced running via deploy_seed_node.sh.
# Usage: sudo bash scripts/patch_node_services.sh
# =============================================================================
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: Run as root (sudo bash scripts/patch_node_services.sh)"
    exit 1
fi

INSTALL_DIR="/opt/oasyce"
SERVICE_USER="oasyce"
NODE_HOME="/home/${SERVICE_USER}/.oasyced"
CHAIN_ID="oasyce-testnet-1"

echo "============================================"
echo "  Oasyce Node — Service Patch"
echo "============================================"
echo ""

# ── 1. Journal log rotation ──
echo "==> Configuring journal limits..."
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

# ── 2. Logrotate (for future file-based logging) ──
echo "==> Installing logrotate config..."
mkdir -p /var/log/oasyced
chown "$SERVICE_USER:$SERVICE_USER" /var/log/oasyced
cat > /etc/logrotate.d/oasyced << 'EOF'
/var/log/oasyced/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    maxsize 100M
    create 0640 oasyce oasyce
}
EOF
echo "    Logrotate: daily, 7 rotations, max 100MB."

# ── 3. Faucet service ──
echo "==> Setting up faucet service..."
if [ ! -f "${INSTALL_DIR}/src/scripts/faucet_server.py" ]; then
    echo "    ERROR: faucet_server.py not found at ${INSTALL_DIR}/src/scripts/"
    echo "    Pull latest source: cd ${INSTALL_DIR}/src && git pull"
    exit 1
fi

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
Environment=FAUCET_RATE_FILE=/home/${SERVICE_USER}/.faucet_rate.json
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

# ── 4. Firewall ──
if command -v ufw &>/dev/null; then
    echo "==> Opening faucet port..."
    ufw allow 8080/tcp comment "Oasyce Faucet" > /dev/null 2>&1
    echo "    Port 8080 opened."
fi

# ── 5. Verify ──
echo ""
echo "============================================"
echo "  Patch Applied"
echo "============================================"
echo ""

if systemctl is-active --quiet oasyced; then
    echo "  oasyced:        ✓ running"
else
    echo "  oasyced:        ✗ NOT running"
fi

if systemctl is-active --quiet oasyce-faucet; then
    PUBLIC_IP=$(curl -s -4 ifconfig.me 2>/dev/null || echo "UNKNOWN")
    echo "  oasyce-faucet:  ✓ running on :8080"
    echo ""
    echo "  Faucet URL:  http://${PUBLIC_IP}:8080/faucet?address=oasyce1..."
else
    echo "  oasyce-faucet:  ✗ NOT running"
    echo "  Check: journalctl -u oasyce-faucet -n 20"
fi

echo ""
echo "  Journal limits:  500MB max, 7-day retention"
echo "  Logrotate:       /etc/logrotate.d/oasyced"
echo ""
echo "  Commands:"
echo "    journalctl -u oasyce-faucet -f   # faucet logs"
echo "    systemctl restart oasyce-faucet   # restart faucet"
echo "    journalctl --disk-usage           # check journal size"
echo ""
