# Join Oasyce Testnet

> 3 minutes from zero to a running node.

## Network Info

| | |
|---|---|
| Chain ID | `oasyce-testnet-1` |
| Seed node | `3e5a914ab7e7400091ddf461fb14992de785b0cb@47.93.32.88:26656` (PEX enabled — nodes discover each other) |
| RPC | `http://47.93.32.88:26657` |
| REST API | `http://47.93.32.88:1317` |
| gRPC | `47.93.32.88:9090` |
| Faucet | `http://47.93.32.88:8080/faucet?address=<your-address>` |
| Binary | [GitHub Releases latest](https://github.com/Shangri-la-0428/oasyce-chain/releases/latest) |
| Genesis SHA256 | `dcc6508926567bc384220d1e92ef538d25c8e5431c380420459b0210d30c7739` |

---

## If You Only Need an Address + Faucet Funds

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_account.sh)
```

This creates a local key in `~/.oasyced`, requests faucet funds, and prints your address plus balance query.

If you are on native Windows PowerShell, use the PowerShell scripts instead of Bash process substitution:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/install_oasyced.ps1 -OutFile install_oasyced.ps1
powershell -ExecutionPolicy Bypass -File .\install_oasyced.ps1
Invoke-WebRequest https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_account.ps1 -OutFile bootstrap_public_beta_account.ps1
powershell -ExecutionPolicy Bypass -File .\bootstrap_public_beta_account.ps1
```

---

## Option A: One-Click Docker (Fastest)

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

## Option B: Native CLI Install (Recommended)

### 1. Install `oasyced`

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/install_oasyced.sh)
oasyced version
```

Windows PowerShell:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/install_oasyced.ps1 -OutFile install_oasyced.ps1
powershell -ExecutionPolicy Bypass -File .\install_oasyced.ps1
oasyced version
```

By default the installer uses `~/.local/bin`. If that directory is not on your `PATH`, add it:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### 2. Or do the whole native setup in one command

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_node.sh)
```

This prepares `~/.oasyced` for `oasyce-testnet-1`, downloads `genesis.json`, patches testnet-friendly params, and configures the seed peer plus REST API.

If you want to start the node immediately in the current shell, use:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/run_public_beta_node.sh)
```

If you want a Linux service instead of a foreground process, use:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/install_public_beta_service.sh)
```

### 3. Manual path: initialize yourself

```bash
oasyced init my-node --chain-id oasyce-testnet-1
```

### 4. Genesis

```bash
curl -L -o ~/.oasyced/config/genesis.json \
  https://github.com/Shangri-la-0428/oasyce-chain/releases/download/testnet-1/genesis.json

# Verify
echo "dcc6508926567bc384220d1e92ef538d25c8e5431c380420459b0210d30c7739  $HOME/.oasyced/config/genesis.json" | sha256sum -c
```

### 5. Configure

```bash
# Add seed node (PEX discovers other peers automatically)
sed -i.bak 's/seeds = ""/seeds = "3e5a914ab7e7400091ddf461fb14992de785b0cb@47.93.32.88:26656"/' \
  ~/.oasyced/config/config.toml

# Allow peers behind NAT
sed -i.bak 's/addr_book_strict = true/addr_book_strict = false/' \
  ~/.oasyced/config/config.toml

# Enable REST API
sed -i.bak 's/enable = false/enable = true/' ~/.oasyced/config/app.toml
```

### 6. Start

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

| Port | Protocol | Who needs it |
|------|----------|-------------|
| 26656 | P2P | **Validators**: must be open. **Full nodes**: optional — outbound connections work behind NAT. |
| 26657 | RPC | Optional (local queries) |
| 1317 | REST API | Optional |
| 9090 | gRPC | Optional |

Full nodes behind NAT (home computers, laptops) work fine without opening any ports. They connect outbound to seed nodes, discover peers via PEX, sync blocks, and submit transactions.

## Minimum Hardware

| | |
|---|---|
| CPU | 2 cores |
| RAM | 2 GB |
| Disk | 40 GB SSD |
| Network | Stable internet connection |

---

## Troubleshooting

**Node won't sync?**
- Check peers: `curl -s localhost:26657/net_info | jq '.result.n_peers'`
- If 0 peers, verify seed config: `grep seeds ~/.oasyced/config/config.toml`
- Behind NAT? That's fine — outbound connections work. No need to open ports.

**Genesis mismatch?**
- Re-download and verify SHA256 checksum above

**Out of memory?**
- Reduce `db_backend` to `goleveldb` (default), avoid `pebbledb` on <4GB RAM

## Links

- [Full Validator Guide](VALIDATOR_SETUP.md)
- [Public Beta Guide](PUBLIC_BETA.md)
- [Architecture](ARCHITECTURE.md)
- [Economics](ECONOMICS.md)
- [Agent Workflows](AGENT_WORKFLOWS.md)
- [AI Integration](../AGENTS.md)
