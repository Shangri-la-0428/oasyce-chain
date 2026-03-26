# Oasyce Chain

[![CI](https://github.com/Shangri-la-0428/oasyce-chain/actions/workflows/ci.yml/badge.svg)](https://github.com/Shangri-la-0428/oasyce-chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> Chinese version: [README.md](README.md) | LLM-optimized docs: [llms.txt](llms.txt)

**Property, contracts, and arbitration for the agent economy.**

When AI agents start collaborating, the problem isn't "how to transfer money" — it's: **Who owns the data? How is it priced? What happens when someone cheats? How are revenues shared?**

Stripe / x402 / Tempo solve "how to pay." Oasyce solves "why the payment is justified."

---

## Beyond Payments

| Problem | Payment Rails (Stripe, x402, Tempo) | Oasyce |
|---------|-------------------------------------|--------|
| **Data ownership** | Not addressed | Data securitization — bonding curve pricing, share trading, version migration |
| **Fair pricing** | Fixed price / off-chain negotiation | Bancor continuous curve — price rises with demand |
| **Service delivery** | Pay and hope | On-chain escrow + challenge window + dispute mechanism |
| **Trust** | None / platform reputation | On-chain credit scores (time-decaying, verifiable feedback) |
| **Disputes** | Chargebacks or nothing | On-chain jury voting, deterministic outcome |
| **Access** | KYC + corporate entity | PoW self-registration, permissionless |

---

## Economic System Architecture

### Core — Property & Contracts

| Module | Economic Function | TX | Queries |
|--------|------------------|-----|---------|
| **x/datarights** | Data securitization — register, share trading, bonding curves, tiered access, jury disputes, version migration | 11 | 9 |
| **x/capability** | Service contracts — register/invoke/complete/fail/claim/dispute, challenge window, auto-settlement | 8 | 4 |
| **x/settlement** | Transaction clearing — atomic escrow, bonding curve engine, 2% deflationary burn | 3 | 4 |

### Supporting Infrastructure

| Module | Economic Function | TX | Queries |
|--------|------------------|-----|---------|
| **x/reputation** | Credit scoring — time-decaying (30-day half-life), factors into pricing and jury selection | 2 | 3 |
| **x/work** | Verifiable compute — commit-reveal anti-copying, multi-executor consensus | 6 | 8 |
| **x/onboarding** | Permissionless access — PoW anti-sybil, airdrop halving economics | 2 | 3 |

**Total**: 32 transaction types, 33 query endpoints, 66 CLI commands.

---

## Quick Start

### Build

```bash
git clone https://github.com/Shangri-la-0428/oasyce-chain.git
cd oasyce-chain
CGO_ENABLED=0 make build
```

### Run 4-Validator Local Testnet

```bash
bash scripts/init_multi_testnet.sh
bash scripts/start_testnet.sh
```

Port allocation:

| Node | P2P | RPC | REST API | gRPC |
|------|-----|-----|----------|------|
| node0 | 26656 | 26657 | 1317 | 9090 |
| node1 | 26756 | 26757 | 1417 | 9190 |
| node2 | 26856 | 26857 | 1517 | 9290 |
| node3 | 26956 | 26957 | 1617 | 9390 |

### Run Tests

```bash
make test   # 50+ tests across 7 suites
```

---

## For Agent Developers

### REST API (recommended)

```python
import requests

BASE = "http://localhost:1317"

# List all AI capabilities
caps = requests.get(f"{BASE}/oasyce/capability/v1/capabilities").json()

# Check account balance
bal = requests.get(f"{BASE}/cosmos/bank/v1beta1/balances/{address}").json()

# Query a data asset
asset = requests.get(f"{BASE}/oasyce/datarights/v1/data_asset/{asset_id}").json()

# Check reputation
rep = requests.get(f"{BASE}/oasyce/reputation/v1/reputation/{address}").json()
```

### CLI + JSON (for AI agent integration)

```bash
# All commands support --output json
oasyced query settlement escrow ESC001 --output json
oasyced query oasyce_capability list --output json
oasyced query datarights asset DATA_001 --output json
```

### gRPC (high performance)

```
localhost:9090
```

Full API reference: [llms.txt](llms.txt) | OpenAPI spec: [docs/openapi.yaml](docs/openapi.yaml)

---

## CLI Examples

```bash
# === Agent Registration (PoW self-register, no KYC) ===
oasyced tx onboarding register [nonce] --from agent1

# === Register AI Capability ===
oasyced tx oasyce_capability register \
  --name "Translation API" \
  --endpoint "https://api.example.com/translate" \
  --price 500000uoas \
  --tags "nlp,translation" \
  --from provider

# === Invoke Capability (auto escrow + settlement) ===
oasyced tx oasyce_capability invoke [cap-id] '{"text":"hello","target":"zh"}' --from consumer

# === Register Data Asset ===
oasyced tx datarights register \
  --name "Medical Imaging Dataset" \
  --content-hash "abc123..." \
  --tags "medical,imaging" \
  --from alice

# === Buy Data Shares (Bancor curve pricing) ===
oasyced tx datarights buy-shares [asset-id] 1000000uoas --from bob

# === Sell Shares (inverse curve, 5% protocol fee) ===
oasyced tx datarights sell-shares [asset-id] 100 --from bob

# === Submit Compute Task ===
oasyced tx work submit-task \
  --task-type "data-cleaning" \
  --input-hash [sha256] \
  --bounty 1000uoas \
  --from submitter

# === Query Reputation ===
oasyced query reputation show [address]
oasyced query reputation leaderboard
```

---

## Protocol Economics

| Parameter | Value |
|-----------|-------|
| Token | OAS (1 OAS = 1,000,000 uoas) |
| Bonding Curve | Bancor, CW = 0.5 |
| Escrow Release Fee Split | 90% provider, 5% protocol, 2% burn, 3% treasury |
| Sell Protocol Fee | 5% |
| Reserve Solvency Cap | 95% max payout on sell |
| Block Rewards | 4→2→1→0.5 OAS/block halving (every 10M blocks) |
| Block Time | ~5 seconds |
| Max Validators | 100 |
| Unbonding Period | 21 days |
| Jury Size | 5 jurors per dispute |
| Jury Threshold | 2/3 majority |

### Airdrop Halving Economics

| Registrations | Airdrop | PoW Difficulty |
|---------------|---------|----------------|
| 0 – 10,000 | 20 OAS | 16 bits |
| 10,001 – 50,000 | 10 OAS | 18 bits |
| 50,001 – 200,000 | 5 OAS | 20 bits |
| 200,001+ | 2.5 OAS | 22 bits |

---

## Architecture

```
                    +---------------------------+
                    |      oasyce-chain (Go)    |
                    |    Cosmos SDK v0.50.10    |
                    |   CometBFT Consensus     |
                    |   7 custom modules        |
                    |   gRPC :9090 / REST :1317 |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    |   oasyce (Python CLI)     |
                    |   Agent client + Dashboard|
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
| [oasyce-chain](https://github.com/Shangri-la-0428/oasyce-chain) (this repo) | L1 settlement chain | `make build` |
| [oasyce](https://github.com/Shangri-la-0428/oasyce-net) | Python agent client + CLI + Dashboard | `pip install oasyce` |
| [DataVault](https://github.com/Shangri-la-0428/DataVault) | AI agent data asset management skill | `pip install odv[oasyce]` |

---

## Core Mechanisms

- **Bancor Bonding Curve** — `tokens = supply * (sqrt(1 + payment/reserve) - 1)`. More buyers = higher price. No order book
- **Inverse Curve Sell** — `payout = reserve * (1 - (1 - tokens/supply)^2)`, 95% reserve cap
- **2% Deflationary Burn** — Every escrow release burns 2%
- **Access Level Gating** — >=0.1% equity -> L0, >=1% -> L1, >=5% -> L2, >=10% -> L3
- **Jury Voting** — `sha256(disputeID + nodeID) * log(1 + reputation)`, 5 jurors, 2/3 majority
- **Commit-Reveal PoUW** — `sha256(output_hash + salt + executor + unavailable)` anti-copying
- **Deterministic Task Assignment** — `sha256(taskID + blockHash + addr) / log(1 + reputation)`
- **PoW Self-Registration** — `sha256(address || nonce)` with N leading zero bits, no KYC, anti-Sybil

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md). Do NOT open public issues for security vulnerabilities.

## License

[Apache 2.0](LICENSE)

## Community

- Discord: [https://discord.gg/tfrCn54yZW](https://discord.gg/tfrCn54yZW)
