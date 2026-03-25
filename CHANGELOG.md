# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.4.0] - 2026-03-25

### Changed (Economic Model Review)

- **Fee split unified** — Settlement and Work both now use 90/5/2/3 (provider 90%, protocol 5%, burn 2%, treasury 3%). Was 93/3/2/2 for settlement.
- **Slashing relaxed** — Downtime penalty 0.01% (was 1%, 100x too harsh), SignedBlocksWindow 10000 (was 100), MinSignedPerWindow 5% (was 50%)
- **Governance lowered** — MinDeposit 100 OAS (was 1000), Quorum 25% (was 40%) to encourage early participation
- **Reputation cooldown** — FeedbackCooldownSeconds 3600 (was 60) to prevent rating spam
- **Settlement ProtocolFeeRate** — 5% (was 3%), TreasuryRate 3% (was 2%), aligned with Work module
- **All tests updated** — 50+ tests passing with new parameters

## [0.3.0] - 2026-03-24

### Added

- **x/halving module** — Custom block reward halving (4→2→1→0.5 OAS per 10M blocks), replaces standard Cosmos inflation
- **x/onboarding module** — PoW self-registration with airdrop halving economics (4 epochs: 20→10→5→2.5 OAS)
- **Datarights lifecycle** — AssetStatus state machine (ACTIVE→SHUTTING_DOWN→SETTLED), MsgInitiateShutdown, MsgClaimSettlement
- **Datarights versioning** — parent_asset_id, version chain, any-address forking
- **Datarights migration** — MsgCreateMigrationPath, MsgMigrate (burn source → mint target at exchange rate), max_migrated_shares cap
- **Validator incentive docs** — docs/VALIDATOR_SETUP.md with 3 revenue streams, ROI examples, systemd/docker setup
- **Public testnet scripts** — scripts/init_public_testnet.sh (genesis generation, faucet account, parameter patching)
- **Faucet** — scripts/faucet.sh (rate limiting, address validation, configurable amounts)
- **Agent demo** — scripts/agent_demo.sh (full lifecycle: onboard→register→invoke→buy/sell→reputation)
- **Website** — website/index.html (landing page), website/docs.html (API documentation with curl/Python examples)
- **llms.txt** — LLM-readable protocol documentation for agent discoverability
- **OpenAPI spec** — docs/openapi.yaml covering all 7 modules
- **oasyce-sdk** — Python SDK published as separate package (github.com/Shangri-la-0428/oasyce-sdk)

### Changed

- **Fee split aligned to spec** — 90% provider, 5% protocol, 2% burn, 3% treasury (was 85/7/5/3)
- **Sell fee** — 3% protocol fee (was 5%), round-trip cost reduced from ~28% to ~12%
- **onboarding ConsensusVersion** bumped to 3 (halving economics migration)
- **datarights ConsensusVersion** bumped to 2 (lifecycle + versioning)

### Security

- Dead code cleanup: removed unused SettlementKeeper interface from datarights module
- Airdrop/difficulty scaling prevents late-stage Sybil attacks (higher PoW cost as network grows)

## [0.2.0] - 2026-03-22

### Added

- **x/work module** — Proof of Useful Work with commit-reveal verification
  - Task lifecycle: Submit→Assign→Commit→Reveal→Settle/Expire/Dispute
  - Deterministic assignment: sha256(taskID+blockHash+addr) / log(1+reputation)
  - Settlement: 90% executor, 5% protocol, 2% burn, 3% submitter rebate
  - Anti-DoS: bounty × deposit_rate held as deposit
  - BeginBlocker/EndBlocker for task expiry and assignment
  - 6 Msg types, 8 Query types, full CLI
  - 13 tests (executor, task CRUD, commit-reveal, assignment, settlement, minority penalty)

## [0.1.0] - 2026-03-20

### Added

- **Cosmos SDK v0.50.10 appchain** with CometBFT consensus
- **x/settlement** — Escrow lifecycle (create, release, refund, expire), Bancor bonding curve pricing
- **x/datarights** — Data asset registration, share trading (buy/sell via Bancor curve), dispute filing & resolution
- **x/capability** — AI capability registration & invocation with escrow-backed settlement
- **x/reputation** — Feedback-based reputation scoring with time decay
- **Bancor continuous bonding curve** — `tokens = supply * (sqrt(1 + payment/reserve) - 1)`, CW=0.5
- **Inverse Bancor sell mechanism** — `payout = reserve * (1 - (1 - tokens/supply)^2)`, 95% reserve solvency cap
- **2% token burn** on escrow release (90% provider, 5% protocol, 2% burn, 3% treasury)
- **Access level gating** — Equity-based tiered access (L0-L3) capped by reputation score
- **Jury voting** — Deterministic jury selection, 2/3 majority threshold, deposit return on upheld disputes
- **Protobuf-based gRPC/REST API** for all 4 modules
- **CLI commands** for all module operations (`oasyced tx/query`)
- **Docker support** — Multi-stage build, 4-node testnet via docker-compose
- **CI/CD** — GitHub Actions (build, test, lint, Docker image)
- **E2E test suite** — `scripts/e2e_test.sh`
- **Multi-validator testnet** — `scripts/init_multi_testnet.sh`

### Security

- Deterministic ID generation (sha256 counter+blockHash, no crypto/rand)
- Int64 overflow prevention in jury scoring (big.Float with 1e15 cap)
- Jury membership enforcement via KV store
- Minimum 1 uoas fee floor on small sells
- Dispute deposit return to plaintiff on upheld resolution
