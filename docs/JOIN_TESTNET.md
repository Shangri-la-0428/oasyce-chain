# Join Oasyce Testnet

> 3 minutes from zero to a running node.

## Network Info

| | |
|---|---|
| Chain ID | `oasyce-testnet-1` |
| Seed node | `390f9b726d7ab105aade989f444f06585bc06186@47.93.32.88:26656` |
| RPC | `http://47.93.32.88:26657` |
| REST API | `http://47.93.32.88:1317` |
| gRPC | `47.93.32.88:9090` |
| Faucet | `http://47.93.32.88:8080/faucet?address=<your-address>` |
| Binary | [GitHub Releases v0.5.0](https://github.com/Shangri-la-0428/oasyce-chain/releases/tag/v0.5.0) |
| Genesis SHA256 | `dcc6508926567bc384220d1e92ef538d25c8e5431c380420459b0210d30c7739` |

---

## Option A: One-Click Docker (Recommended)

```bash
# Prerequisites: Docker
bash <(curl -sL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/join_testnet.sh)
```

Done. Your node syncs automatically. Check status:

```bash
docker logs -f oasyce-node
docker exec oasyce-node oasyced status | jq '.SyncInfo.catching_up'
# false = fully synced
```

---

## Option B: Binary Install

### 1. Download

```bash
# Linux amd64
curl -L -o oasyced https://github.com/Shangri-la-0428/oasyce-chain/releases/download/v0.5.0/oasyced-linux-amd64

# macOS Apple Silicon
curl -L -o oasyced https://github.com/Shangri-la-0428/oasyce-chain/releases/download/v0.5.0/oasyced-darwin-arm64

# macOS Intel
curl -L -o oasyced https://github.com/Shangri-la-0428/oasyce-chain/releases/download/v0.5.0/oasyced-darwin-amd64

chmod +x oasyced && sudo mv oasyced /usr/local/bin/
oasyced version  # should print v0.5.0
```

### 2. Initialize

```bash
oasyced init my-node --chain-id oasyce-testnet-1
```

### 3. Genesis

```bash
curl -L -o ~/.oasyced/config/genesis.json \
  https://github.com/Shangri-la-0428/oasyce-chain/releases/download/testnet-1/genesis.json

# Verify
echo "dcc6508926567bc384220d1e92ef538d25c8e5431c380420459b0210d30c7739  $HOME/.oasyced/config/genesis.json" | sha256sum -c
```

### 4. Configure

```bash
# Add seed peer
sed -i.bak 's/persistent_peers = ""/persistent_peers = "390f9b726d7ab105aade989f444f06585bc06186@47.93.32.88:26656"/' \
  ~/.oasyced/config/config.toml

# Enable REST API
sed -i.bak 's/enable = false/enable = true/' ~/.oasyced/config/app.toml
```

### 5. Start

```bash
oasyced start --minimum-gas-prices 0uoas
```

Wait for sync (`catching_up: false`), then you're live.

---

## Get Test Tokens

```bash
# Create a key
oasyced keys add mykey --keyring-backend test

# Request tokens (100 OAS, 1 request/hour/address)
curl "http://47.93.32.88:8080/faucet?address=$(oasyced keys show mykey -a --keyring-backend test)"
```

---

## Become a Validator (Optional)

After syncing:

```bash
oasyced tx staking create-validator \
  --amount=500000000uoas \
  --pubkey=$(oasyced tendermint show-validator) \
  --moniker="my-validator" \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --from=mykey \
  --chain-id=oasyce-testnet-1 \
  --keyring-backend=test \
  --fees=10000uoas \
  --yes
```

---

## Run as systemd Service (Linux)

```bash
sudo tee /etc/systemd/system/oasyced.service << EOF
[Unit]
Description=Oasyce Chain Node
After=network-online.target

[Service]
User=$USER
ExecStart=$(which oasyced) start --minimum-gas-prices 0uoas
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now oasyced
journalctl -u oasyced -f  # watch logs
```

---

## Verify Connection

```bash
# Check sync
oasyced status | jq '.SyncInfo'

# Check peers
curl -s localhost:26657/net_info | jq '.result.n_peers'

# Query chain modules
curl -s http://localhost:1317/oasyce/capability/v1/params | jq
```

---

## Ports

| Port | Protocol | Required |
|------|----------|----------|
| 26656 | P2P | Yes (must be open) |
| 26657 | RPC | Optional (local queries) |
| 1317 | REST API | Optional |
| 9090 | gRPC | Optional |

## Minimum Hardware

| | |
|---|---|
| CPU | 2 cores |
| RAM | 2 GB |
| Disk | 40 GB SSD |
| Network | Stable, port 26656 open |

---

## Troubleshooting

**Node won't sync?**
- Check peer: `curl -s localhost:26657/net_info | jq '.result.peers[].node_info.moniker'`
- Ensure port 26656 is open: `ufw allow 26656/tcp`

**Genesis mismatch?**
- Re-download and verify SHA256 checksum above

**Out of memory?**
- Reduce `db_backend` to `goleveldb` (default), avoid `pebbledb` on <4GB RAM

## Links

- [Full Validator Guide](VALIDATOR_SETUP.md)
- [Architecture](ARCHITECTURE.md)
- [Economics](ECONOMICS.md)
- [Agent Workflows](AGENT_WORKFLOWS.md)
- [AI Integration](../AGENTS.md)
