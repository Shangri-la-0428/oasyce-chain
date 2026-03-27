# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.5.3] - 2026-03-28

### Added

- **DataAsset `service_url` field** — Buyers can now discover where to access data after purchasing shares. Set at registration via `--service-url` flag, updatable via `update-service-url` command (owner only). Content integrity still guaranteed by `content_hash`.
- **`MsgUpdateServiceUrl`** — New transaction type for updating data access endpoints. Owner-only, empty string clears the URL.
- **Consumer Agent** — `scripts/consumer_agent.py` autonomous consumer for cron-based economic cycles (faucet → discover → invoke → feedback)
- **Provider health degradation** — Provider agent returns HTTP 503 "degraded" when upstream is down; consumer pre-checks and skips cycle
- **Economic metrics in healthcheck** — Provider earnings tracking, call growth stall detection, consumer liveness monitoring, economic summary log

### Fixed

- **GetSigners for MsgUpdateServiceUrl** — Added explicit `GetSigners()` method since `Descriptor()` returns nil; without it, TX signer extraction would fail on live chain
- **Consumer state PermissionError** — Fallback delete+recreate when cron user differs from file owner
- **Gas auto sequence mismatch** — Changed `--gas auto` to fixed `--gas 200000` to avoid account sequence errors (code 19)

---

## [0.5.1] - 2026-03-26

### Added

- **REST params endpoints** — `GET /oasyce/reputation/v1/params` and `GET /oasyce/capability/v1/params` (proto + keeper + CLI)
- **7 missing CLI query commands** — settlement `params`/`bonding-curve-price`, datarights `params`/`shares`/`access-level`/`disputes`, capability `by-provider`
- **Params validation** — All 6 modules validate `Params` struct in `UpdateParams` handler before persisting
- **Cross-compile target** — `make build-linux` for VPS deployment with version ldflags + strip
- **Testnet reset script** — `scripts/reset_testnet.sh` for full VPS chain reset with key recovery

### Fixed

- **SuccessRate uint32 overflow** — Capability success rate calculation now uses uint64 arithmetic (prevents silent truncation after 4.3B calls)
- **Website version** — Download links updated from v0.4.0 to v0.5.0
- **Stale root llms.txt** — Removed root `llms.txt` (referenced `oasyce-localnet-1`); `docs/llms.txt` is the single source of truth
- **Endpoint count** — llms.txt query endpoint count corrected from 33 to 35
- **All docs updated** — Node ID, genesis SHA256, and network params synced across README, JOIN_TESTNET, VALIDATOR_SETUP, website

### Security

- **Governance params guard** — `UpdateParams` now calls `Params.Validate()` in all 6 modules, preventing invalid parameter values via governance proposals

---

## [0.5.0] - 2026-03-26

### Added

- **Challenge window mechanism** — `CompleteInvocation` starts 100-block challenge window; `ClaimInvocation` after expiry, `DisputeInvocation` within window (refunds escrow)
- **AccessLevel query endpoint** — `GET /oasyce/datarights/v1/access_level/{asset_id}/{address}` returns L0-L3 tier
- **Invocation query endpoint** — `GET /oasyce/capability/v1/invocation/{invocation_id}` for tracking challenge window progress
- **UpdateCapability and DeactivateCapability** — CLI commands for managing registered services
- **MsgUpdateParams for all 6 modules** — Governance-gated parameter updates with `Descriptor()` methods
- **AI-First agent interface** — `/llms.txt`, `/openapi.yaml`, `/.well-known/oasyce.json`, `/oasyce/v1/error-codes` served from node
- **PoW solver CLI** — `oasyced util solve-pow [address]` for agent self-onboarding
- **gRPC reflection** — `grpcurl -plaintext :9090 list` returns all services
- **AI auto-reporting** — Issue template `.github/ISSUE_TEMPLATE/ai_agent_report.md`, `report_issue` in discovery JSON
- **Tx codec integration tests** — 152 sub-tests covering all 38 message types (Descriptor, RegisterInterfaces, marshal roundtrip)
- **Proto descriptor patcher** — `tools/patch_descriptors` with `ensureSignerOptions()` and `validateModule()` for hand-written .pb.go
- **Agent workflows doc** — `docs/AGENT_WORKFLOWS.md` with 5 complete step-by-step flows
- **QA test suite** — `scripts/qa_full_test.sh` for comprehensive end-to-end protocol validation

### Fixed

- **Missing Descriptor() methods** — All hand-written protobuf types (20 across 6 modules) now have correct `Descriptor()` returning proper file descriptor index
- **Missing cosmos.msg.v1.signer options** — Patcher auto-injects signer extension bytes for all Msg types
- **E2E test timing** — `wait_tx` increased from 3s to 7s for 5s block time; PoW test uses built-in solver; user2 account pre-funded
- **Reputation query returns default zero** — Unknown address now returns zero-value response instead of HTTP 500
- **Documentation sync** — README, openapi.yaml, llms.txt, CONTRIBUTING.md updated to match actual 33 query endpoints

### Changed

- **Query count** — 33 query endpoints (was 32), 66+ CLI commands
- **Minimum fees updated** — Default fees in examples updated from `500uoas` to `10000uoas`
- **CONTRIBUTING.md** — Added proto descriptor requirements and patcher tool documentation

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
