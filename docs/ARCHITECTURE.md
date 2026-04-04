# Architecture

> Oasyce Chain — the public lifecycle ledger, authorization truth, commitments, settlement, and finality layer for the Sigil stack

## Stack Role

Oasyce Chain is not the runtime body, not the shared high-frequency memory layer, and not the whole product.

Its job is narrower and stronger:

- record Sigil lifecycle events as ordered public history
- act as the final authorization source of truth
- hold durable commitments and settlement state
- provide public finality when off-chain execution must become auditable fact

In the broader stack:

- `Oasyce-Sigil` defines the constitutional grammar
- `oasyce-sdk` instantiates local delegates and signer access
- `Thronglets` handles spaces, traces, signals, presence, and delegate continuity
- `Psyche` handles subjective continuity
- `Oasyce Chain` records the durable public layer

## Independent Adoption

Oasyce Chain must remain independently consumable.

- any client may use it directly via CLI / REST / gRPC
- Psyche is optional
- Thronglets is optional
- oasyce-sdk is optional

The chain is a public truth layer, not an install-time dependency hub.

## Overview

Oasyce Chain is built on Cosmos SDK v0.50.10 + CometBFT consensus with 10 custom modules organized in three tiers.

```
Client / Runtime Layer
  oasyce-sdk  |  oasyced CLI  |  MCP / agents
       |
       v
Tier 1 — Axiom-derived public primitives
  x/sigil       lifecycle ledger
  x/anchor      evidence bridge
  x/onboarding  permissionless genesis path

Tier 2 — Economic and authorization infrastructure
  x/settlement  escrow + pricing + fee routing
  x/delegate    policy-bounded delegated execution
  x/datarights  asset/equity/access/dispute layer
  x/halving     emissions and scarcity schedule

Tier 3 — Higher-level convenience surfaces
  x/capability  service invocation surface
  x/reputation  feedback residue and scoring
  x/work        useful work coordination surface

Base Layer
  Cosmos SDK + CometBFT
```

## Module Dependency Graph

```
x/capability  ──→ x/settlement (escrow for invocations)
x/capability  ──→ bank (stake validation)
x/anchor      ──→ x/sigil (optional sigil linkage for traces)
x/datarights  ──→ x/settlement (bonding curve pricing)
x/datarights  ──→ bank (share payments, sell payouts)
x/delegate    ──→ bank (policy-bounded spending)
x/onboarding  ──→ x/sigil (self-registration emits GENESIS)
x/reputation  ──→ x/capability (link feedback to invocations)
x/settlement  ──→ bank (escrow transfers, burns, fees)
x/work        ──→ x/settlement (task bounty escrow)
x/work        ──→ x/reputation (executor reputation for assignment)
x/onboarding  ──→ bank (mint airdrop, burn repayment)
x/halving     ──→ bank (mint block rewards → fee_collector)
```

## Module Details

### x/sigil — Lifecycle Ledger

The constitutional core of the chain.

- **Lifecycle**: `GENESIS`, `DISSOLVE`, `BOND`, `UNBOND`, `FORK`, `MERGE`
- **Continuity history**: ordered, public, immutable event stream for continuing digital actors
- **Liveness decay**: active → dormant → dissolved via chain time
- **Lineage**: fork inheritance and merge absorption recorded as first-class history

Key files:
- `x/sigil/keeper/keeper.go` — lifecycle state transitions and indexes
- `x/sigil/keeper/msg_server.go` — lifecycle message handlers
- `x/sigil/keeper/begin_blocker.go` — liveness decay

### x/anchor — Evidence Bridge

Optional bridge from high-frequency off-chain traces into public proof.

- **Anchor trace hashes**: content-addressed immutable record
- **Attach signer evidence**: ed25519 node signature + optional Sigil ID
- **Batch mode**: amortize public anchoring for sparse durable evidence

Key files:
- `x/anchor/keeper/keeper.go` — anchor storage and indexes
- `x/anchor/keeper/msg_server.go` — single + batch anchoring
- `x/anchor/keeper/query_server.go` — by trace / capability / node / sigil

### x/delegate — Authorization Surface

Policy-bounded delegated execution on behalf of a principal.

- **Policy**: one on-chain budget + message allowlist per principal
- **Enroll**: delegates self-enroll with a shared token
- **Exec**: delegates submit bounded msgs under principal policy
- **Spend windows**: gross outflow tracking prevents masking via round trips

Key files:
- `x/delegate/keeper/keeper.go` — policy, delegate, and spend window state
- `x/delegate/keeper/msg_server.go` — set-policy, enroll, revoke, exec
- `x/delegate/keeper/query_server.go` — policy, delegates, spend, principal lookup

### x/settlement — Escrow & Bonding Curve

The economic finality backbone. Handles:

- **Escrow lifecycle**: `LOCKED → RELEASED | REFUNDED | EXPIRED`
- **Fee split on release**: 90% provider, 5% protocol, 2% burn, 3% treasury
- **Bancor bonding curve**: continuous pricing with CW=0.5
- **Auto-expiry**: stale escrows refunded in EndBlock

Key files:
- `keeper/keeper.go` — CreateEscrow, ReleaseEscrow, RefundEscrow, ExpireStaleEscrows
- `keeper/bonding_curve.go` — BancorBuy, BancorSell, SpotPrice
- `types/types.go` — protocol constants (ReserveRatio, BurnRate, etc.)

### x/datarights — Asset, Equity, and Access Surface

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

### x/capability — Capability Invocation Surface

Registers AI capabilities and manages invocation-to-payment flow.
This is a convenience surface, not the constitutional center of the chain.

- **Register**: provider stakes, publishes endpoint URL + price + tags
- **Invoke**: consumer triggers invocation → auto-creates escrow
- **Complete/Fail**: provider completes → escrow released; failure → refunded
- **Stats**: tracks success rate, total calls, total earned per capability

Key files:
- `keeper/keeper.go` — RegisterCapability, InvokeCapability, CompleteInvocation, FailInvocation

### x/reputation — Feedback Residue

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

Permissionless genesis path with anti-sybil PoW.

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
| 26657 | RPC (CometBFT) |
| 1317  | REST (gRPC-GW)   |
| 9090  | gRPC             |

## Build & CI

- **Build**: `make build` → `build/oasyced`
- **Test**: `make test` → `go test ./... -v -race`
- **Lint**: `make lint` → golangci-lint
- **Docker**: `make docker-build` → multi-stage Alpine image
- **Testnet**: `docker-compose up` → 4-node local testnet
- **CI**: GitHub Actions — build + test + lint + Docker build on push/PR to main
