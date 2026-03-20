# Roadmap

## Phase A — Foundation (Complete ✅)

- [x] Cosmos SDK v0.50.10 chain initialization
- [x] 4 custom modules: settlement, capability, reputation, datarights
- [x] Protobuf migration (16 proto files, gRPC + REST)
- [x] CLI commands for all modules (tx + query)
- [x] Bancor continuous bonding curve (CW=0.5)
- [x] 2% deflationary token burn on settlement
- [x] Sell mechanism with inverse Bancor curve
- [x] Equity-based access gating (L0-L3)
- [x] Jury voting system (5 jurors, 2/3 majority)
- [x] Owner delist + slippage protection
- [x] E2E verification with real transactions
- [x] CI/CD (GitHub Actions), Docker, 4-node testnet
- [x] Open-source infrastructure (README, LICENSE, CONTRIBUTING, etc.)
- [x] 40+ tests across 5 suites

## Phase B — Production Readiness (In Progress)

- [x] Full `proto-gen` pipeline (MsgSellShares, MsgDelistAsset → protobuf wire format)
- [x] IBC integration — ibc-go v8.8.0, cross-chain OAS transfers, Tendermint light client
- [x] Governance module — fully wired (7d voting, 40% quorum, param change proposals)
- [x] Mainnet genesis configuration — all modules with default params
- [x] Swagger/OpenAPI documentation — REST API spec for all 4 custom modules + IBC
- [x] IBC guide — channel setup, cross-chain transfer walkthrough
- [ ] Security audit (external)
- [ ] Validator incentive program
- [ ] Testnet launch with external validators

## Phase C — Proof of Useful Work (PoUW) (Complete ✅)

- [x] `x/work` module — reward validators for executing AI compute tasks
- [x] Task lifecycle state machine (Submit→Assign→Commit→Reveal→Settle)
- [x] Commit-reveal scheme (anti copy-attack)
- [x] Deterministic executor assignment (sha256 + reputation-weighted)
- [x] Redundant execution + 2/3 majority consensus settlement
- [x] Economics: 90% executor / 5% protocol / 2% burn / 3% rebate
- [x] Anti-gaming: self-assignment prevention, deposit-based DoS protection
- [x] Executor registration and capability declaration
- [x] Input unavailability handling (2/3 threshold)
- [x] Epoch statistics tracking
- [x] Full CLI (6 tx commands + 8 query commands)
- [x] Protobuf (4 proto files, gRPC + REST)
- [x] 13 unit tests, zero regression

## Phase D — Ecosystem Growth

- [ ] SDK for third-party data providers
- [ ] Cross-chain data rights (IBC data attestations)
- [ ] Privacy-preserving compute (TEE / ZK integration)
- [ ] Mobile wallet support
- [ ] Fiat on-ramp partnerships
- [ ] Multi-language SDK (JS/TS, Rust, Python)

## Phase E — Decentralized AI Marketplace

- [ ] Agent-to-agent capability discovery
- [ ] Federated learning coordination via x/capability
- [ ] Data DAO governance (community-owned datasets)
- [ ] Revenue sharing for co-created AI models
- [ ] Reputation portability across chains (IBC reputation)
