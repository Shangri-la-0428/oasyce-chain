# Architecture Decisions

Honest documentation of design choices, trade-offs, and known limitations.

## Why Cosmos SDK (Not Substrate / Solidity / Pure Rust)

**Decision**: Cosmos SDK v0.50.10 (Go)

**Why**:
- Module system maps cleanly to our economic primitives (datarights, capability, settlement are natural Cosmos modules)
- IBC enables future cross-chain settlement without custom bridge code
- Mature tooling: protobuf code generation, gRPC-gateway, CLI scaffolding
- Go is readable enough for auditors who aren't blockchain-native

**Trade-off**: No smart contract flexibility. Every new economic primitive requires a module upgrade + chain restart. This is acceptable because our primitives are well-defined (property, contracts, arbitration) and shouldn't change frequently.

## Challenge Window vs TEE/ZK Verification

**Decision**: 100-block challenge window with consumer dispute rights. No TEE or zero-knowledge proofs.

**Why**:
- TEE (Intel SGX, ARM TrustZone) requires specific hardware and trusted manufacturers — antithetical to permissionless design
- ZK verification of arbitrary AI computation is not production-ready (circuit compilation is too expensive for real workloads)
- Challenge window + economic incentives (escrow at risk) provides practical trust without hardware assumptions

**How it works**:
1. Consumer invokes capability, funds locked in escrow
2. Provider submits output hash (must be >=32 chars, non-empty)
3. 100-block window (~8 minutes at 5s/block) begins
4. Consumer can dispute (full refund) or do nothing
5. After window, provider claims payment

**Known limitation**: A dishonest provider can submit a valid-looking hash for garbage output. The consumer must verify off-chain and dispute within the window. This is analogous to credit card chargebacks but faster and deterministic.

**Future direction**: Reputation penalties for disputed providers create economic disincentive over time. Repeated disputes tank success_rate and reputation score, making the provider less attractive in the marketplace.

## Bancor Bonding Curve (Not Order Book / AMM)

**Decision**: Bancor continuous token model with connector weight 0.5.

**Why**:
- No liquidity bootstrapping problem — the curve itself provides liquidity from token #1
- Deterministic pricing: `tokens = supply * (sqrt(1 + payment/reserve) - 1)`
- Price automatically rises with demand (no oracle needed)
- Inverse curve for sells with 95% reserve solvency cap prevents bank runs

**Trade-off**: CW=0.5 means aggressive price discovery — early buyers get significant advantage. This is intentional: it rewards early discovery of valuable data assets.

**Known limitation**: The bonding curve operates per-asset. There's no cross-asset arbitrage or price correlation. Each data asset is independently priced by its own demand.

## Access Level Gating (Chain-Side Compute, Off-Chain Enforcement)

**Decision**: Chain computes access levels (L0-L3 based on equity %), off-chain gateways enforce data delivery.

**Why**:
- Chain cannot and should not deliver data (size, latency, privacy)
- Chain IS the authoritative source for "who owns how much"
- Off-chain gateways query `GET /oasyce/datarights/v1/access_level/{asset_id}/{address}` and decide delivery
- This is the correct separation: chain = property registry, gateway = delivery

**Access levels**:
| Equity | Level | Access |
|--------|-------|--------|
| >= 0.1% | L0 | Metadata only |
| >= 1% | L1 | Preview/sample |
| >= 5% | L2 | Full read |
| >= 10% | L3 | Full delivery |

**Known limitation**: The gateway is trusted. A malicious gateway could serve wrong data or refuse service. This is mitigated by the data's content hash being on-chain — consumers can verify integrity after delivery.

## Jury Voting (Not Governance / Not DAO)

**Decision**: Deterministic jury selection with 5 jurors, 2/3 majority threshold.

**Selection formula**: `sha256(disputeID + nodeID) * log(1 + reputation)`

**Why**:
- Full governance voting is too slow for commercial disputes (days vs minutes)
- DAO voting suffers from voter apathy and whale dominance
- Small jury with reputation weighting is fast and resistant to collusion
- Deterministic selection means jury cannot be bribed in advance (unknown until dispute filed)

**Known limitation**: With only 5 jurors, a determined attacker who controls 3 high-reputation addresses can swing verdicts. This is acceptable at current scale. At larger scale, jury size should increase to 11 or 21.

## PoW Self-Registration (Not KYC / Not Staking)

**Decision**: Proof-of-Work anti-Sybil with halving economics.

**Why**:
- KYC is antithetical to autonomous agents (agents don't have passports)
- Staking-based registration creates chicken-and-egg (need tokens to register, but registration gives first tokens)
- PoW provides permissionless entry with tunable cost
- Halving schedule (20 OAS → 10 → 5 → 2.5) controls supply while maintaining accessibility

**Trade-off**: PoW is energy-intensive. At current scale (testnet), this is negligible. The difficulty (16-22 bits) is deliberately low — minutes on a modern CPU, not hours.

## 2% Deflationary Burn

**Decision**: Every escrow release burns 2% of the transaction value.

**Why**:
- Creates deflationary pressure that increases with economic activity
- Combined with halving block rewards (4 → 2 → 1 → 0.5 OAS/block), supply eventually peaks then contracts
- Burns are irreversible and on-chain verifiable — no trust required

**Fee split**: 90% provider, 5% protocol, 2% burn, 3% treasury.

## What We Don't Do (And Why)

- **No smart contracts**: Our economic primitives are fixed. Smart contract flexibility would add attack surface without clear benefit.
- **No cross-chain bridges**: IBC handles this natively. Custom bridges are the #1 source of DeFi exploits.
- **No oracle integration**: Bonding curves provide price discovery without external data. No oracle dependency = no oracle manipulation.
- **No governance token**: OAS is a utility token. Governance is through standard Cosmos SDK governance module with deposit + voting.
- **No privacy layer**: All transactions are public. Data privacy is handled off-chain by the delivery gateways, not on-chain.
