# Roadmap

## Phase A — Foundation (Complete ✅)

- [x] Cosmos SDK v0.50.10 chain initialization
- [x] 8 custom modules: settlement, capability, reputation, datarights, work, onboarding, halving, anchor
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
- [x] 152+ tests across 10 suites

## Phase B — Production Readiness (Complete ✅)

- [x] Full `proto-gen` pipeline (MsgSellShares, MsgDelistAsset → protobuf wire format)
- [x] IBC integration — ibc-go v8.8.0, cross-chain OAS transfers, Tendermint light client
- [x] Governance module — fully wired (7d voting, 25% quorum, param change proposals)
- [x] Mainnet genesis configuration — all modules with default params
- [x] Swagger/OpenAPI documentation — REST API spec for all 8 custom modules + IBC
- [x] IBC guide — channel setup, cross-chain transfer walkthrough
- [x] AI-first agent interface: llms.txt, AGENTS.md, openapi.yaml, error-codes, PoW solver
- [x] x/anchor module — Thronglets trace anchoring (content-addressed, batch support)

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

## Phase D — Public Testnet (Complete ✅)

- [x] VPS seed node deployed (47.93.32.88)
- [x] Faucet, Provider Agent, nginx rate limiting
- [x] Python SDK v0.5.0 — native Cosmos signing, zero Go dependency
- [x] 25 MCP tools (11 read + 14 write) + 18 LangChain tools
- [x] Autonomous agents: provider, consumer, data registration
- [x] Healthcheck + logrotate + monitoring
- [x] Website + docs.html API reference
- [x] Public beta announced (2026-03-27)

## Phase E — Ecosystem Growth (In Progress)

- [x] Python SDK with native signing (`pip install oasyce-sdk`)
- [ ] 3+ external validators on testnet
- [ ] Real data asset trading on-chain (Bancor curve demo)
- [ ] Security audit (external)
- [ ] Cross-chain data rights (IBC data attestations)
- [ ] Privacy-preserving compute (TEE / ZK integration)

## Phase F — Mainnet

- [ ] Security audit complete (no critical/high findings)
- [ ] 3+ validators stable for 14 days
- [ ] Mainnet genesis (no faucet, formal token distribution)
- [ ] SDK v1.0.0 + CLI mainnet release
- [ ] Mobile wallet support

## Future

- [ ] Agent-to-agent capability discovery (AHRP multi-hop routing)
- [ ] Federated learning coordination via x/capability
- [ ] Data DAO governance (community-owned datasets)
- [ ] Revenue sharing for co-created AI models
- [ ] Reputation portability across chains (IBC reputation)
- [ ] Thronglets P2P shared memory integration
- [ ] Lens hardware PoPC protocol
