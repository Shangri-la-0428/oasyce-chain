# Architecture

> Oasyce Chain — A rights settlement layer for AI agents

## Overview

Oasyce Chain is built on Cosmos SDK v0.50.10 + CometBFT consensus with 7 custom modules that together form a marketplace for AI capabilities and data rights.

```
┌─────────────────────────────────────────────────────┐
│                    Client Layer                      │
│  oasyce CLI (Python)  │  oasyce-sdk (Agent)         │
└──────────┬────────────┴───────────┬──────────────────┘
           │ gRPC / REST            │ CLI tx
┌──────────▼────────────────────────▼──────────────────┐
│                  oasyce-chain (Go)                    │
│                                                       │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐ │
│  │ x/datarights│  │x/capability │  │ x/reputation │ │
│  │ Assets      │  │ Endpoints   │  │ Feedback     │ │
│  │ Shares      │  │ Invocations │  │ Scores       │ │
│  │ Disputes    │  │ Challenge   │  │ Reports      │ │
│  │ Jury/Access │  │ Window      │  │ Cooldown     │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬───────┘ │
│         │                │                │          │
│  ┌──────┴──┐  ┌─────────┴─────┐  ┌───────┴────────┐ │
│  │ x/work  │  │x/onboarding   │  │ x/halving      │ │
│  │ PoUW    │  │PoW Self-Reg   │  │Block Rewards   │ │
│  │ Commit- │  │Airdrop Halving│  │4→2→1→0.5 OAS   │ │
│  │ Reveal  │  │Anti-Sybil     │  │Deflationary    │ │
│  └────┬────┘  └───────┬───────┘  └───────┬────────┘ │
│       │               │                  │           │
│  ┌────▼───────────────▼──────────────────▼────────┐  │
│  │              x/settlement                       │  │
│  │  Escrow Lifecycle  │  Bancor Bonding Curve      │  │
│  │  2% Burn           │  Protocol Fees             │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │        Cosmos SDK (bank, auth, staking, ...)    │  │
│  └─────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────┐  │
│  │               CometBFT Consensus                │  │
│  └─────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────┘
```

## Module Dependency Graph

```
x/capability  ──→ x/settlement (escrow for invocations)
x/capability  ──→ bank (stake validation)
x/datarights  ──→ x/settlement (bonding curve pricing)
x/datarights  ──→ bank (share payments, sell payouts)
x/reputation  ──→ x/capability (link feedback to invocations)
x/settlement  ──→ bank (escrow transfers, burns, fees)
x/work        ──→ x/settlement (task bounty escrow)
x/work        ──→ x/reputation (executor reputation for assignment)
x/onboarding  ──→ bank (mint airdrop, burn repayment)
x/halving     ──→ bank (mint block rewards → fee_collector)
```

## Module Details

### x/settlement — Escrow & Bonding Curve

The financial backbone. Handles:

- **Escrow lifecycle**: `LOCKED → RELEASED | REFUNDED | EXPIRED`
- **Fee split on release**: 90% provider, 5% protocol, 2% burn, 3% treasury
- **Bancor bonding curve**: continuous pricing with CW=0.5
- **Auto-expiry**: stale escrows refunded in EndBlock

Key files:
- `keeper/keeper.go` — CreateEscrow, ReleaseEscrow, RefundEscrow, ExpireStaleEscrows
- `keeper/bonding_curve.go` — BancorBuy, BancorSell, SpotPrice
- `types/types.go` — protocol constants (ReserveRatio, BurnRate, etc.)

### x/datarights — Data Asset Marketplace

Manages data asset registration, equity trading, and dispute resolution.

- **Asset registration**: fingerprint-based ID, rights types (0-3), co-creators
- **Share trading**: buy via Bancor curve, sell via inverse curve (95% solvency cap)
- **Access gating**: L0-L3 levels based on equity % + reputation score
- **Disputes**: file with deposit, jury selection, 2/3 majority voting
- **Delist**: owner or jury can deactivate an asset

Key files:
- `keeper/keeper.go` — RegisterDataAsset, BuyShares, SellShares, DelistAsset
- `keeper/access_level.go` — GetAccessLevel (equity thresholds + reputation caps)
- `keeper/jury.go` — SelectJury, SubmitJuryVote, TallyVotes, ResolveByJury

### x/capability — AI Endpoint Registry

Registers AI capabilities and manages invocation-to-payment flow.

- **Register**: provider stakes, publishes endpoint URL + price + tags
- **Invoke**: consumer triggers invocation → auto-creates escrow
- **Complete/Fail**: provider completes → escrow released; failure → refunded
- **Stats**: tracks success rate, total calls, total earned per capability

Key files:
- `keeper/keeper.go` — RegisterCapability, InvokeCapability, CompleteInvocation, FailInvocation

### x/reputation — Trust & Feedback

Time-decayed reputation scoring based on invocation feedback.

- **Feedback**: 0-500 rating linked to invocation, with verified weight (2x)
- **Decay**: exponential, half-life = 30 days
- **Score**: weighted average → 0-500 range
- **Reports**: misbehavior evidence submission for governance review
- **Cooldown**: 1 hour between same submitter→target feedback

Key files:
- `keeper/keeper.go` — SubmitFeedback, UpdateScore, ReportMisbehavior, GetReputation

### x/work — Proof of Useful Work

Verifiable off-chain computation with commit-reveal scheme.

- **Task lifecycle**: Submit → Assign → Commit → Reveal → Settle/Expire/Dispute
- **Commit-reveal**: `sha256(output_hash + salt + executor + unavailable)` prevents result copying
- **Deterministic assignment**: `sha256(taskID + blockHash + addr) / log(1 + reputation)`
- **BeginBlocker**: expires timed-out tasks and reveal windows
- **Settlement**: 90% executor, 5% protocol, 2% burn, 3% submitter rebate

Key files:
- `keeper/task.go` — SubmitTask, AssignTask, CommitResult, RevealResult
- `keeper/msg_server.go` — all 6 Msg handlers
- `keeper/begin_blocker.go` — ExpireTimedOutTasks, ExpireRevealWindows

### x/onboarding — PoW Self-Registration

Permissionless identity registration with anti-sybil PoW.

- **PoW**: `sha256(address || nonce)` with N leading zero bits
- **Airdrop**: minted as repayable debt (20 OAS, halves with registrations)
- **Halving economics**: difficulty and airdrop scale with total registrations (4 epochs)

Key files:
- `keeper/keeper.go` — SelfRegister, RepayDebt, HalvingEpoch

### x/halving — Block Reward Halving

Custom block rewards replacing standard Cosmos SDK inflation.

- **Schedule**: 4→2→1→0.5 OAS/block, halving every 10M blocks
- **BeginBlocker**: mint → halving module → fee_collector → distribution → validators
- **Standard mint disabled**: inflation = 0%

Key files:
- `keeper/keeper.go` — BlockReward, BeginBlocker

## Data Flow Examples

### Buy Shares

```
Consumer                  x/datarights              x/settlement           bank
   │ BuyShares(asset,amt)     │                         │                    │
   │─────────────────────────▶│ Bancor formula           │                    │
   │                          │ tokens = f(payment)      │                    │
   │                          │──────────────────────────▶│ SendCoins          │
   │                          │                           │──────────────────▶│
   │                          │ update shares + reserve   │                    │
   │◀─────────────────────────│ emit event                │                    │
```

### Capability Invocation

```
Consumer          x/capability        x/settlement         Provider
   │ Invoke()          │                    │                  │
   │──────────────────▶│ CreateEscrow()     │                  │
   │                   │──────────────────▶│ lock funds        │
   │                   │                    │                  │
   │                   │  (off-chain: consumer calls endpoint) │
   │                   │                    │                  │
   │                   │ CompleteInvocation()│                  │
   │                   │──────────────────▶│ ReleaseEscrow()   │
   │                   │                    │ 90% → provider   │
   │                   │                    │  5% → fee_collector│
   │                   │                    │  2% → burn 🔥     │
   │                   │                    │  3% → treasury    │
```

## Network Ports

| Port  | Service         |
|-------|-----------------|
| 26656 | P2P (CometBFT)  |
| 26657 | RPC (Tendermint) |
| 1317  | REST (gRPC-GW)   |
| 9090  | gRPC             |

## Build & CI

- **Build**: `make build` → `build/oasyced`
- **Test**: `make test` → `go test ./... -v -race`
- **Lint**: `make lint` → golangci-lint
- **Docker**: `make docker-build` → multi-stage Alpine image
- **Testnet**: `docker-compose up` → 4-node local testnet
- **CI**: GitHub Actions — build + test + lint + Docker build on push/PR to main
