# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-03-20

### Added

- **Cosmos SDK v0.50.10 appchain** with CometBFT consensus
- **x/settlement** — Escrow lifecycle (create, release, refund, expire), Bancor bonding curve pricing
- **x/datarights** — Data asset registration, share trading (buy/sell via Bancor curve), dispute filing & resolution
- **x/capability** — AI capability registration & invocation with escrow-backed settlement
- **x/reputation** — Feedback-based reputation scoring with time decay
- **Bancor continuous bonding curve** — `tokens = supply * (sqrt(1 + payment/reserve) - 1)`, CW=0.5
- **Inverse Bancor sell mechanism** — `payout = reserve * (1 - (1 - tokens/supply)^2)`, 95% reserve solvency cap
- **2% token burn** on escrow release (93% provider, 5% protocol fee, 2% burn)
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
