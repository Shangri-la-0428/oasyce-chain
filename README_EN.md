# Oasyce Chain

[![CI](https://github.com/oasyce/chain/actions/workflows/ci.yml/badge.svg)](https://github.com/oasyce/chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> 中文版: [README.md](README.md)

**A rights settlement layer for AI agents.**

Oasyce Chain is a Cosmos SDK appchain where every data access and capability invocation between AI agents is priced, escrowed, and settled automatically. Data has sovereignty. Capabilities have a price.

Think of it as **Stripe for the AI economy** — a payment and settlement layer purpose-built for machine-to-machine transactions.

---

## Why Oasyce?

Today's AI uses your data for free. Oasyce changes that:

- You take a photo, an AI wants to use it for training — the AI must pay, you earn automatically
- You build a translation API — other agents call it, you earn per invocation, quality backed by staked collateral
- Pricing is automatic (bonding curves), settlement is trustless (escrow), disputes are decentralized (jury voting)

---

## Modules

| Module | Purpose |
|--------|---------|
| **x/datarights** | Register data assets, buy/sell shares (Bancor curve), file disputes, jury voting, access level gating |
| **x/settlement** | Escrow lifecycle, bonding curve pricing, 2% token burn, fee distribution |
| **x/capability** | Register AI capabilities (endpoints), invoke via escrow-backed settlement |
| **x/reputation** | Feedback-based scoring with time decay, leaderboard |

### Key Features

- **Bancor Bonding Curve** — Automatic pricing: `tokens = supply * (sqrt(1 + payment/reserve) - 1)`. More buyers = higher price. No order book needed.
- **Sell Mechanism** — Sell shares back to the curve: `payout = reserve * (1 - (1 - tokens/supply)^2)`. 95% reserve solvency cap.
- **2% Token Burn** — Every escrow release burns 2% of the amount (93% provider, 5% protocol fee, 2% burn). Deflationary by design.
- **Access Level Gating** — Hold equity to unlock tiered access: >=0.1% -> L0, >=1% -> L1, >=5% -> L2, >=10% -> L3. Capped by reputation.
- **Jury Voting** — Disputes resolved by deterministic jury selection (`sha256(disputeID + nodeID) * log(1 + reputation)`), 2/3 majority threshold.
- **Escrow Settlement** — Lock funds before execution, release after quality verification. Automatic expiry and refund.

---

## Quick Start

### Build

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd oasyce-chain
make build
```

### Run a Local Node

```bash
./scripts/init_testnet.sh
./scripts/start_testnet.sh
```

The node exposes:
- **RPC**: `localhost:26657`
- **REST API**: `localhost:1317`
- **gRPC**: `localhost:9090`

### Run Tests

```bash
make test
```

### Docker (4-Node Testnet)

```bash
make docker-build
docker-compose up
```

---

## CLI Examples

```bash
# Register a data asset
oasyced tx datarights register \
  --name "Medical Imaging Dataset" \
  --content-hash "abc123..." \
  --rights-type 1 \
  --tags "medical,imaging" \
  --from alice

# Buy shares of a data asset
oasyced tx datarights buy-shares \
  --asset-id DATA_xxxx \
  --amount 1000000uoas \
  --from bob

# Sell shares (inverse Bancor curve)
oasyced tx datarights sell-shares \
  --asset-id DATA_xxxx \
  --shares 100 \
  --from bob

# Create an escrow
oasyced tx settlement create-escrow \
  --provider cosmos1xxx \
  --amount 1000000uoas \
  --from alice

# Register an AI capability
oasyced tx oasyce_capability register \
  --name "Translation API" \
  --endpoint "https://api.example.com/translate" \
  --price 500000uoas \
  --from provider

# Query reputation
oasyced query reputation show cosmos1xxx
```

---

## Architecture

```
                    +---------------------------+
                    |      oasyce-chain (Go)    |
                    |    Cosmos SDK v0.50.10    |
                    |   CometBFT Consensus     |
                    |   -----------------------|
                    |   x/datarights            |
                    |   x/settlement            |
                    |   x/capability            |
                    |   x/reputation            |
                    |   gRPC :9090 / REST :1317 |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   oasyce (Python CLI)     |
                    |   Thin client + Dashboard |
                    |   pip install oasyce      |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   DataVault (AI Skill)    |
                    |   Local data management   |
                    |   scan/classify/privacy   |
                    |   pip install odv[oasyce] |
                    +---------------------------+
```

### Ecosystem

| Component | Role | Install |
|-----------|------|---------|
| [oasyce-chain](https://github.com/Shangri-la-0428/oasyce-chain) (this repo) | L1 consensus, state, settlement | `make build` |
| [oasyce](https://github.com/Shangri-la-0428/Oasyce_Claw_Plugin_Engine) | Python thin client, CLI, Dashboard | `pip install oasyce` |
| [DataVault](https://github.com/Shangri-la-0428/DataVault) | AI agent skill for data asset management | `pip install odv[oasyce]` |

---

## Protocol Economics

| Parameter | Value |
|-----------|-------|
| Token | OAS (uoas = 10^-6 OAS) |
| Bonding Curve | Bancor, CW = 0.5 |
| Bootstrap Price | 1 uoas per token |
| Protocol Fee | 5% on escrow release |
| Burn Rate | 2% on escrow release |
| Reserve Solvency Cap | 95% max payout on sell |
| Jury Size | 5 jurors per dispute |
| Jury Threshold | 2/3 majority to uphold |

---

## Current Progress

### Phase A: Core Chain — Complete

- 4 custom modules fully implemented (datarights, settlement, capability, reputation)
- 16 protobuf files migrated, gRPC + REST fully operational
- Bancor bonding curve + 2% burn + sell mechanism + access gating + jury voting
- All CLI commands (tx + query)
- E2E verification passing
- CI/CD, Docker 4-node testnet, GitHub infrastructure

### Phase B: Production Readiness (In Progress)

- IBC cross-chain integration
- Governance module
- Mainnet genesis configuration
- Security audit
- Swagger API documentation
- Validator incentive program
- Public testnet launch

### Phase C: Proof of Useful Work ✅

- x/work module: AI compute task submission + redundant execution + majority consensus settlement
- Commit-reveal anti-copy scheme, deterministic executor assignment, reputation-weighted
- Economics: 90% executor / 5% protocol / 2% burn / 3% submitter rebate
- 6 tx commands + 8 query commands, 13 unit tests

### Phase D: Ecosystem Growth (Planned)

- Cross-chain data rights, privacy-preserving compute, mobile wallet, multi-language SDK

### Phase E: Decentralized AI Marketplace (Long-term)

- Agent auto-discovery, federated learning, data DAOs, revenue sharing

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, and PR process.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting. Do NOT open public issues for security vulnerabilities.

## License

[Apache 2.0](LICENSE)

## Community

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
