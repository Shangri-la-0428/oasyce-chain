# Validator Setup Guide

> Run an Oasyce Chain validator node

## Prerequisites

- Go 1.22+
- 4 CPU cores, 8 GB RAM, 100 GB SSD (minimum)
- Stable network with open ports 26656 (P2P) and 26657 (RPC)

## Build from Source

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd oasyce-chain
make build
```

Binary: `build/oasyced`

## Initialize Node

```bash
# Replace <moniker> with your node name
oasyced init <moniker> --chain-id oasyce-1

# Import or create validator key
oasyced keys add validator --keyring-backend file
# Save the mnemonic!
```

## Configure Genesis

For **testnet**, download the shared genesis file:

```bash
# Replace with actual testnet genesis URL
curl -o ~/.oasyced/config/genesis.json <genesis-url>
```

For **local development**, generate genesis:

```bash
oasyced keys add validator --keyring-backend test
oasyced genesis add-genesis-account validator 1000000000uoas --keyring-backend test
oasyced genesis gentx validator 500000000uoas \
  --chain-id oasyce-local-1 \
  --keyring-backend test
oasyced genesis collect-gentxs
```

### Patch Genesis Parameters

Some defaults need adjustment for testnet:

```bash
# Lower min_provider_stake (default 10B uoas is too high for testing)
python3 -c "
import json
g = json.load(open('$HOME/.oasyced/config/genesis.json'))
g['app_state']['oasyce_capability']['params']['min_provider_stake'] = {
    'denom': 'uoas', 'amount': '0'
}
json.dump(g, open('$HOME/.oasyced/config/genesis.json', 'w'), indent=2)
"
```

## Configure Node

### app.toml

```bash
# Enable REST API
sed -i.bak 's/enable = false/enable = true/' ~/.oasyced/config/app.toml

# Set minimum gas price
sed -i.bak 's/minimum-gas-prices = ""/minimum-gas-prices = "0.025uoas"/' ~/.oasyced/config/app.toml
```

### config.toml

```bash
# Add persistent peers (replace with actual peer addresses)
sed -i.bak 's/persistent_peers = ""/persistent_peers = "<node-id>@<ip>:26656"/' ~/.oasyced/config/config.toml

# Optional: enable Prometheus metrics
sed -i.bak 's/prometheus = false/prometheus = true/' ~/.oasyced/config/config.toml
```

## Start Node

```bash
oasyced start
```

The node will:
1. Connect to peers via P2P (port 26656)
2. Sync blocks from the network
3. Expose RPC on port 26657, REST on 1317, gRPC on 9090

### Run as systemd Service

```ini
# /etc/systemd/system/oasyced.service
[Unit]
Description=Oasyce Chain Node
After=network-online.target

[Service]
User=oasyce
ExecStart=/usr/local/bin/oasyced start
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable oasyced
sudo systemctl start oasyced
journalctl -u oasyced -f  # watch logs
```

## Create Validator

After your node is fully synced:

```bash
oasyced tx staking create-validator \
  --amount=500000000uoas \
  --pubkey=$(oasyced tendermint show-validator) \
  --moniker="<your-moniker>" \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --from=validator \
  --chain-id=oasyce-1 \
  --keyring-backend=file \
  --gas=auto \
  --gas-adjustment=1.5 \
  --yes
```

## Verify

```bash
# Check sync status
oasyced status | jq '.SyncInfo.catching_up'
# false = fully synced

# Check your validator
oasyced query staking validator $(oasyced keys show validator --bech val -a)
```

## Docker (4-Node Local Testnet)

For quick local testing with 4 validators:

```bash
docker-compose up -d
```

| Node | P2P | RPC | REST | gRPC |
|------|-----|-----|------|------|
| node0 | 26656 | 26657 | 1317 | 9090 |
| node1 | 26756 | 26757 | 1417 | 9190 |
| node2 | 26856 | 26857 | 1517 | 9290 |
| node3 | 26956 | 26957 | 1617 | 9390 |

## Ports Reference

| Port | Protocol | Description |
|------|----------|-------------|
| 26656 | TCP | P2P — must be open to other validators |
| 26657 | TCP | RPC — Tendermint queries (can restrict) |
| 1317 | TCP | REST API — gRPC gateway (can restrict) |
| 9090 | TCP | gRPC — direct proto queries (can restrict) |

## Backup & Recovery

```bash
# Backup validator key (CRITICAL — loss = slashing)
cp ~/.oasyced/config/priv_validator_key.json /secure-backup/

# Backup node key
cp ~/.oasyced/config/node_key.json /secure-backup/

# State can be re-synced from network, but key backup is essential
```

## Monitoring

Enable Prometheus in `config.toml`:

```toml
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
```

Key metrics:
- `cometbft_consensus_height` — current block height
- `cometbft_consensus_validators` — active validator count
- `cometbft_p2p_peers` — connected peers
