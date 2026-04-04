# Oasyce Chain

[![CI](https://github.com/Shangri-la-0428/oasyce-chain/actions/workflows/ci.yml/badge.svg)](https://github.com/Shangri-la-0428/oasyce-chain/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.50.10-blue)](https://github.com/cosmos/cosmos-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> Chinese version: [README.md](README.md) | LLM-optimized docs: [docs/llms.txt](docs/llms.txt)

**The public lifecycle ledger, authorization truth layer, and settlement finality layer of the Sigil stack.**

When AI loops begin collaborating across devices, delegates, and markets, the problem is no longer just "how to pay" or "how to call an API" — it is: **what continues to exist, who is authorized to act, which commitments need public finality, and which exchanges need settlement?**

Stripe / x402 / Tempo solve "how to pay." Oasyce solves "why the payment is justified."

---

## Stack Role

- `Sigil` defines continuity and lifecycle grammar
- `oasyce-sdk` instantiates local delegate bodies and binding / signer access
- `Thronglets` carries shared environment, trace, signal, and presence
- `Psyche` carries subjective continuity and self-state
- `Oasyce Chain` records lifecycle events, authorization truth, commitments, settlement, and public finality

So the chain is not the high-frequency runtime, and not the whole product's front door. It only handles facts that must be public, durable, auditable, and final.

## Independent Adoption

`Oasyce Chain` must remain independently usable.

- you can use it directly as a public lifecycle / authorization / settlement ledger
- you can consume it through `CLI / REST / gRPC`
- you do not need `Psyche`
- you do not need `Thronglets`
- you do not need `oasyce-sdk`

`oasyce-sdk` is only the bridge for local delegate runtime + chain flows. It is not a prerequisite for the chain to exist.

---

## Beyond Payments and Beyond Marketplaces

| Problem | Payment Rails (Stripe, x402, Tempo) | Oasyce |
|---------|-------------------------------------|--------|
| **Subject lifecycle** | Not addressed | `x/sigil` records GENESIS / BOND / FORK / MERGE / DISSOLVE |
| **Authorization truth** | Platform ACLs / private config | `x/delegate` + chain state define verifiable execution boundaries |
| **Data ownership** | Not addressed | Data securitization — bonding curve pricing, share trading, version migration |
| **Fair pricing** | Fixed price / off-chain negotiation | Bancor continuous curve — price rises with demand |
| **Service delivery** | Pay and hope | On-chain escrow + challenge window + dispute mechanism |
| **Trust** | None / platform reputation | On-chain credit scores (time-decaying, verifiable feedback) |
| **Disputes** | Chargebacks or nothing | On-chain jury voting, deterministic outcome |
| **Access** | KYC + corporate entity | PoW self-registration, permissionless |

---

## Module Tiers

### Tier 1: Public Primitives

| Module | Role | TX | Queries |
|--------|------|-----|---------|
| **x/sigil** | Lifecycle ledger — GENESIS / BOND / FORK / MERGE / DISSOLVE | 7 | 6 |
| **x/anchor** | Evidence bridge — anchor sparse durable traces as public proof | 2 | 4 |
| **x/onboarding** | Permissionless GENESIS path — PoW anti-sybil + airdrop halving | 2 | 3 |

### Tier 2: Authorization and Economic Infrastructure

| Module | Role | TX | Queries |
|--------|------|-----|---------|
| **x/delegate** | Execution authorization — principal budgets and message boundaries for delegates | 4 | 4 |
| **x/settlement** | Settlement backbone — atomic escrow, pricing, fee routing, burn | 3 | 4 |
| **x/datarights** | Economic layer for assets / shares / access / disputes / migration | 11 | 10 |
| **x/halving** | Scarcity schedule — block rewards and halving cadence | 0 | 2 |

### Tier 3: Higher-Level Surfaces Composed From Primitives

| Module | Role | TX | Queries |
|--------|------|-----|---------|
| **x/capability** | Service invocation surface — register / invoke / challenge window / auto-settlement | 8 | 5 |
| **x/reputation** | Feedback residue — time-decayed reputation for pricing and arbitration context | 2 | 3 |
| **x/work** | Verifiable work surface — commit-reveal and multi-executor consensus | 6 | 8 |

Full workflows and interfaces live in [docs/llms.txt](docs/llms.txt).

---

<!-- BEGIN GENERATED:PUBLIC_BETA_EN -->
## Public Beta

The **single chain-side onboarding guide** for the public beta is [docs/PUBLIC_BETA.md](/Users/wutongcheng/Desktop/Net/oasyce-chain/docs/PUBLIC_BETA.md).

Complete the chain-side onboarding first. Add `oas`, `oasyce-agent`, or `oasyce-sdk` later only when you want richer local workflows, scanning, or Python automation.

- Public beta guide: [docs/PUBLIC_BETA.md](https://github.com/Shangri-la-0428/oasyce-chain/blob/main/docs/PUBLIC_BETA.md)
- Install CLI: `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/install_oasyced.sh)`
- Windows PowerShell install: `Invoke-WebRequest .../install_oasyced.ps1 -OutFile install_oasyced.ps1` then `powershell -ExecutionPolicy Bypass -File ./install_oasyced.ps1`
- Create account + request faucet funds: `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_account.sh)`
- Windows PowerShell account setup: `Invoke-WebRequest .../bootstrap_public_beta_account.ps1 -OutFile bootstrap_public_beta_account.ps1` then `powershell -ExecutionPolicy Bypass -File ./bootstrap_public_beta_account.ps1`
- Prepare local node: `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/bootstrap_public_beta_node.sh)`
- Prepare and start node now: `bash <(curl -fsSL https://raw.githubusercontent.com/Shangri-la-0428/oasyce-chain/main/scripts/run_public_beta_node.sh)`
- Product-side guide: [oasyce-net/docs/public-testnet-guide.md](https://github.com/Shangri-la-0428/oasyce-net/blob/main/docs/public-testnet-guide.md)
- Dashboard: `pip install oasyce && oas bootstrap && oas start`
- Data ingress: [oasyce-sdk README](https://github.com/Shangri-la-0428/oasyce-sdk/blob/main/README.md)
- Python SDK (NativeSigner): `pip install -U "oasyce-sdk>=0.5.0"`
- API reference: [chain.oasyce.com/docs.html](https://chain.oasyce.com/docs.html)
- Validator guide: [docs/VALIDATOR_SETUP.md](https://github.com/Shangri-la-0428/oasyce-chain/blob/main/docs/VALIDATOR_SETUP.md)
- Latest release: [releases/latest](https://github.com/Shangri-la-0428/oasyce-chain/releases/latest)
<!-- END GENERATED:PUBLIC_BETA_EN -->

---

## Quick Start

**Fastest path for the economic path:**

```bash
pip install oasyce-sdk
oasyce-agent start
```

```python
from oasyce_sdk.crypto import Wallet, NativeSigner
from oasyce_sdk import OasyceClient

wallet = Wallet.auto()  # reuse local binding; first device can run oasyce-agent start first
client = OasyceClient("http://47.93.32.88:1317")
signer = NativeSigner(wallet, client, chain_id="oasyce-testnet-1")
```

You can also skip the SDK entirely and use the chain directly through CLI / REST / gRPC.

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

BASE = "http://<node>:1317"

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

Full API reference: [docs/llms.txt](docs/llms.txt) | OpenAPI spec: [docs/openapi.yaml](docs/openapi.yaml)

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
oasyced tx oasyce_capability invoke [cap-id] --input '{"text":"hello","target":"zh"}' --from consumer

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
                    |   oasyce-sdk (Agent SDK)  |
                    | binding/signer/data ingress|
                    |   MCP Server + LangChain  |
                    |   pip install oasyce-sdk  |
                    +---------------------------+
```

### Ecosystem

| Component | Role | Install |
|-----------|------|---------|
| [oasyce-chain](https://github.com/Shangri-la-0428/oasyce-chain) (this repo) | L1 settlement chain | `make build` |
| [oasyce](https://github.com/Shangri-la-0428/oasyce-net) | Python agent client + CLI + Dashboard | `pip install oasyce && oas bootstrap` |
| [oasyce-sdk](https://github.com/Shangri-la-0428/oasyce-sdk) | Python Agent SDK (binding/signer/MCP/LangChain) | `pip install oasyce-sdk` |

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
