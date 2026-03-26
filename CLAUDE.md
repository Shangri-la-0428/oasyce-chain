# Oasyce L1 Chain

## Project
Cosmos SDK v0.50.10 chain at `/Users/wutongcheng/Desktop/oasyce-chain` with 7 custom modules: settlement, capability, reputation, datarights, work, onboarding, halving.

## Current Status

### Protobuf Migration — COMPLETE ✅
- All 5 modules fully protobuf-migrated (including x/work)
- app.go wired with real keepers, store keys, module manager

### REST/gRPC Integration — COMPLETE ✅
- REST API on :1317, gRPC on :9090
- gRPC-Gateway routes registered for all 6 custom modules
- Node starts, produces blocks, persists data, restarts cleanly
- All standard Cosmos queries (bank, auth, staking) work
- All custom module queries work

### CLI Commands — COMPLETE ✅
- All 6 modules have CLI tx + query commands (`x/*/cli/`)
- `oasyced tx datarights register|buy-shares|file-dispute|resolve-dispute`
- `oasyced tx settlement create-escrow|release-escrow|refund-escrow`
- `oasyced tx oasyce_capability register|invoke|complete-invocation|fail-invocation|claim-invocation|dispute-invocation`
- `oasyced tx reputation submit-feedback|report`
- `oasyced tx send` (bank transfers)
- Query: `oasyced query settlement escrow|escrows|bonding-curve-price|params`
- Query: `oasyced query datarights asset|list|shares|access-level|dispute|disputes|migration-path|migration-paths|children|params`
- Query: `oasyced query oasyce_capability get|list|by-provider|earnings|invocation|params`
- Query: `oasyced query reputation show|leaderboard|params`
- Query: `oasyced query work task|tasks-by-status|executor|executors|params|epoch`
- Query: `oasyced query onboarding registration|debt|params`

### Chain Upgrades — COMPLETE ✅
Five economic/governance upgrades implemented and tested:

1. **Bancor Bonding Curve** — Replaced tiered pricing with continuous curve
   - `tokens = supply × (√(1 + payment/reserve) − 1)`, CW=0.5
   - Bootstrap: `tokens = payment / INITIAL_PRICE` when reserve=0
   - Files: `x/settlement/keeper/bonding_curve.go`, `x/datarights/keeper/keeper.go`

2. **2% Token Burn** — Escrow release now splits: 90% provider, 5% protocol, 2% burn, 3% treasury
   - `BurnCoins()` added to BankKeeper interface
   - File: `x/settlement/keeper/keeper.go` (ReleaseEscrow)

3. **Sell Mechanism** — Inverse Bancor curve for selling shares back
   - `payout = reserve × (1 − (1 − tokens/supply)²)`, capped at 95% reserve
   - 5% protocol fee on sell payout
   - Files: `x/datarights/keeper/keeper.go` (SellShares), `x/datarights/types/msg_sell.go`

4. **Access Level Gating** — Equity-based tiered access (L0-L3)
   - ≥0.1%→L0, ≥1%→L1, ≥5%→L2, ≥10%→L3, capped by reputation score
   - File: `x/datarights/keeper/access_level.go`

5. **Jury Voting** — Decentralized dispute resolution
   - Deterministic jury selection: `sha256(disputeID+nodeID) × log(1+reputation)`
   - 5 jurors, 2/3 majority threshold, persisted membership
   - Files: `x/datarights/keeper/jury.go`

### End-to-End Verification — COMPLETE ✅
All 7 modules verified with real transactions:
- **datarights**: register asset, buy shares (Bancor curve), sell shares, access gating, jury voting
- **settlement**: create escrow (LOCKED), release escrow (RELEASED, 5% fee + 2% burn)
- **capability**: register capability, invoke (creates escrow + invocation), complete + challenge window + claim/dispute
- **reputation**: submit feedback (score=450), leaderboard populated
- **bank**: cross-account transfers work
- E2E test script: `scripts/e2e_test.sh`

### Provider Stake & Endpoint Liveness — COMPLETE ✅
- **Real stake locking**: `min_provider_stake` now actually locks funds in module account (was: balance-check only)
- **Stake return on deactivate**: `DeactivateCapability` returns locked stake to provider (errors are propagated, not silently swallowed)
- **CLI liveness check**: `register` and `update --endpoint` ping the endpoint before broadcasting tx
- **Bypass flag**: `--skip-liveness-check` for testing/special cases
- Module account registered in `maccPerms` for holding provider stakes
- Stake stored per capability at `ProviderStakePrefix + capID`

### Claude AI Proxy (VPS) — DEPLOYED ✅
- **Script**: `scripts/claude_proxy.py` → deployed to VPS at `/home/oasyce/claude_proxy.py`
- **Service**: systemd `claude-proxy`, internal port 8090, external via nginx `/proxy/`
- **Endpoint**: `http://47.93.32.88/proxy/v1/chat` (POST)
- **Format**: Accepts `{"prompt":"..."}` or `{"messages":[...]}`, forwards as Anthropic Messages API
- **Response**: `{"text":"...", "model":"...", "usage":{"input_tokens":N,"output_tokens":N}, "raw":{...}}`
- **On-chain**: `CAP_0000000000000008` "Claude AI (Opus 4.6)", 0.5 OAS/call, provider=validator

### Known Genesis Param Issues
- `oasyce_capability.min_provider_stake`: defaults to 0 uoas — set to 5,000,000 (5 OAS) for testnet
- `datarights.dispute_deposit`: 10,000,000 uoas (10 OAS)
- Patch genesis.json after `init` before starting chain

### x/work Module (Proof of Useful Work) — COMPLETE ✅
- Task lifecycle: Submit → Assign → Commit → Reveal → Settle/Expire/Dispute
- **Commit-reveal** scheme prevents result copying (sha256(output_hash + salt + executor + unavailable))
- **Deterministic assignment**: sha256(taskID + blockHash + addr) / log(1 + reputation), creator excluded
- **Settlement**: 90% executor / 5% protocol / 2% burn / 3% submitter rebate
- **Anti-DoS**: bounty × deposit_rate held as deposit, forfeited if input unavailable
- **BeginBlocker**: expires timed-out tasks (max_tasks_per_block cap per block)
- **EndBlocker**: assigns executors to SUBMITTED tasks using current block hash
- 6 Msg types: RegisterExecutor, UpdateExecutor, SubmitTask, CommitResult, RevealResult, DisputeResult
- 8 Query types: Task, TasksByStatus, TasksByCreator, TasksByExecutor, ExecutorProfile, Executors, WorkParams, EpochStats
- CLI: `oasyced tx work submit-task|commit-result|reveal-result|register-executor|update-executor|dispute-result`
- Query: `oasyced query work task|tasks-by-status|executor|executors|params|epoch`
- Files: `x/work/`, `proto/oasyce/work/v1/`

### Capability Challenge Window — COMPLETE ✅
- **Two-phase completion**: Provider submits output (COMPLETED) → 100-block challenge window → Claim (SUCCESS) or Dispute (DISPUTED + refund)
- `CompleteInvocation` requires non-empty output_hash (≥32 chars), marks COMPLETED, records completed_height + optional usage_report, escrow stays LOCKED
- `ClaimInvocation` (new): provider claims after challenge window, releases escrow, marks SUCCESS
- `DisputeInvocation` (new): consumer disputes within window, refunds escrow, marks DISPUTED
- New proto messages: MsgCompleteInvocation, MsgFailInvocation, MsgClaimInvocation, MsgDisputeInvocation
- New statuses: INVOCATION_STATUS_COMPLETED, INVOCATION_STATUS_DISPUTED
- CLI: `oasyced tx oasyce_capability claim-invocation|dispute-invocation`
- ChallengeWindow constant: 100 blocks (~8 minutes at 5s/block)
- `usage_report` field (optional JSON string): on-chain token/resource usage tracking for transparency
  - Stored in Invocation (field 12) and submitted via MsgCompleteInvocation (field 4)
  - CLI: `--usage-report '{"prompt_tokens":150,"completion_tokens":80}'`
  - Provider agent auto-extracts from OpenAI-compatible upstream responses
- 5 new tests (complete flow, dispute flow, expired window, early claim rejection, short hash rejection)

### AccessLevel Query Endpoint — COMPLETE ✅
- REST: `GET /oasyce/datarights/v1/access_level/{asset_id}/{address}`
- Returns: access_level (L0-L3 or empty), equity_bps, shares, total_shares
- Off-chain gateways query this endpoint to decide data delivery tier
- Architecture: chain computes level, off-chain enforces gating

### Build & Test Status
```
go build ./...  ✅
go test ./...   ✅ (130+ tests across 10 suites)
  tests/integration     — 3 tests (full capability flow w/ challenge window, Bancor curve, escrow lifecycle)
  x/capability/keeper   — 11 tests (register, invoke, complete, claim, dispute, deactivate, auth, rate limit, tags)
  x/datarights/keeper   — 16 tests (Bancor buy/sell, access gating, jury voting, lifecycle, versioning, migration)
  x/reputation/keeper   — reputation tests
  x/settlement/keeper   — escrow + bonding curve tests
  x/work/keeper         — 13 tests (executor, task CRUD, commit-reveal, assignment, settlement, minority penalty)
  x/onboarding/keeper   — 4 tests (invite+claim, repay, cancel, default settlement)
  x/halving/keeper      — 13 tests (block reward boundaries, halving transitions, cumulative supply)
```

### Datarights Lifecycle + Versioning + Migration — COMPLETE ✅
- **Lifecycle State Machine**: AssetStatus enum (ACTIVE → SHUTTING_DOWN → SETTLED)
  - `MsgInitiateShutdown` — owner triggers graceful shutdown with 7-day cooldown
  - `MsgClaimSettlement` — pro-rata reserve payout after cooldown, no fee
  - BuyShares blocked unless ACTIVE; SellShares blocked after cooldown
  - Dispute DELIST remedy triggers shutdown (not instant delist)
  - ConsensusVersion = 2
- **Versioning**: DataAsset fields `parent_asset_id`, `version`, `migration_enabled`
  - MsgRegisterDataAsset accepts `parent_asset_id`, auto-calculates version
  - Any address can fork (no same-owner requirement)
- **Migration**: MigrationPath as independent first-class object
  - `MsgCreateMigrationPath` — target owner creates, version chain validated
  - `MsgDisableMigration` — emergency disable
  - `MsgMigrate` — burns source shares, mints target at exchange rate
  - `max_migrated_shares` caps dilution; no reserve transfer (accepted dilution)
- CLI: `initiate-shutdown`, `claim-settlement`, `create-migration`, `disable-migration`, `migrate`
- Query: `migration-path`, `migration-paths`, `children` (asset version tree)
- Dead code cleanup: removed unused SettlementKeeper interface from datarights module

### x/onboarding Module (PoW Self-Registration) — COMPLETE ✅
- Proof-of-Work based self-registration: anyone can join by solving a hash puzzle
- **Flow**: Client solves sha256(address || nonce) with N leading zero bits → submits to chain → receives airdrop (minted as debt) → repays debt (burned)
- **Anti-sybil**: PoW cost (~2-5 min CPU per registration), one-registration-per-address
- **Debt**: Airdrop is a loan, repaid tokens are burned to maintain supply
- 2 Msg types: SelfRegister, RepayDebt
- 3 Query types: Registration, Debt, OnboardingParams
- CLI: `oasyced tx onboarding register <nonce>`, `oasyced tx onboarding repay <amount>`
- Query: `oasyced query onboarding registration|debt|params`
- Module account permissions: Minter + Burner
- Default params: airdrop=20 OAS, pow_difficulty=16 bits, deadline=90 days
- **Halving Economics** (keeper.go): Airdrop and difficulty scale with total registrations
  - Epoch 0 (0–10K): 20 OAS airdrop, 16-bit PoW
  - Epoch 1 (10K–50K): 10 OAS, 18-bit PoW
  - Epoch 2 (50K–200K): 5 OAS, 20-bit PoW
  - Epoch 3 (200K+): 2.5 OAS, 22-bit PoW
  - `total_registrations` counter stored at `TotalRegistrationsKey = 0x03`
  - Effective difficulty = max(params, halving); effective airdrop = min(params, halving)
- ConsensusVersion = 3

### Governance: MsgUpdateParams — COMPLETE ✅
All 6 custom modules support governance-gated parameter updates via `MsgUpdateParams`:
- **settlement**: `oasyced tx settlement update-params [params.json]`
- **capability**: `oasyced tx oasyce_capability update-params [params.json]`
- **reputation**: `oasyced tx reputation update-params [params.json]`
- **datarights**: `oasyced tx datarights update-params [params.json]`
- **work**: `oasyced tx work update-params [params.json]`
- **onboarding**: `oasyced tx onboarding update-params [params.json]`
- Each module's keeper has `Authority()` method, msg_server checks `msg.Authority == keeper.Authority()`
- Params JSON file contains full `Params` struct; broadcasted as standard Cosmos TX
- Files per module: `types/msg_update_params.go`, `keeper/msg_server.go`, `cli/tx.go`, `types/codec.go`

### Multi-Denom Support — COMPLETE ✅
Removed hardcoded `"uoas"` from production code:
- **Settlement bonding curve**: `BondingCurveState.ReserveDenom` field (proto field 6), set on first `BuyShares`, backward-compatible fallback to `"uoas"`
- **Datarights**: per-asset denom stored at `AssetReserveDenomPrefix + assetID`, set on first buy when reserve is zero
- **Capability earnings**: grouped by `PricePerCall.Denom` in Earnings query, falls back to `"uoas"` for legacy capabilities

### AI-First Agent Interface — COMPLETE ✅
Chain nodes now serve AI-discoverable endpoints:
- `GET /llms.txt` — Agent playbook (embedded, 620+ lines) with step-by-step workflows
- `GET /openapi.yaml` — OpenAPI 3.0 spec (embedded)
- `GET /.well-known/oasyce.json` — Service discovery metadata (chain_id, modules, denom, report_issue)
- `GET /oasyce/v1/error-codes` — 60+ error codes with recovery actions (JSON)
- `oasyced util solve-pow [address]` — Built-in PoW solver for agent self-onboarding
- gRPC reflection enabled by Cosmos SDK v0.50.10 (`grpcurl -plaintext :9090 list`)
- **AI auto-reporting**: Agents can file GitHub issues/PRs when encountering problems
  - Issue template: `.github/ISSUE_TEMPLATE/ai_agent_report.md`, label: `ai-reported`
  - `/.well-known/oasyce.json` includes `report_issue` field with API URL
  - `llms.txt` §10 has exact curl/gh commands for filing issues and PRs
- Files: `docs/embed.go`, `docs/error_codes.json`, `cmd/oasyced/cmd/pow.go`, `app/app.go` (RegisterAPIRoutes)
- Docs: `AGENTS.md` (integration guide), `docs/AGENT_WORKFLOWS.md` (5 workflows)

### Proto Descriptor Requirements
Hand-written protobuf message types (msg_update_params.go, msgs_challenge_pb.go) MUST have:
1. `Descriptor()` method returning `(fileDescriptor_xxx, []int{N})` where N = message index in file descriptor
2. `cosmos.msg.v1.signer` option in file descriptor (set by `tools/patch_descriptors`)
3. `proto.RegisterType` in `init()` function
Without Descriptor(): SDK error "does not have a Descriptor() method: tx parse error"
Without signer option: SDK error "no cosmos.msg.v1.signer option found"
Tool: `go run ./tools/patch_descriptors` — adds missing RPCs, messages, and signer options to file descriptors

### Protocol Constants (x/settlement/types/types.go)
```go
ReserveRatio       = 0.5   // Bancor connector weight
InitialPrice       = 1.0   // 1 uoas per token at bootstrap
ReserveSolvencyCap = 0.95  // Max 95% reserve payout on sell
BurnRate           = 0.02  // 2% burn on escrow release
TreasuryRate       = 0.03  // 3% treasury on escrow release
ProtocolFeeRate    = 0.05  // 5% protocol fee (DefaultParams)
```
Fee split on escrow release: 90% provider, 5% protocol, 2% burn, 3% treasury.
Sell fee: 5% protocol fee deducted from bonding curve payout.

### Validator Incentives (app/genesis.go)

Validators earn from three sources:

1. **Block Rewards** — Custom x/halving module with height-based halving
   - Standard mint module disabled (inflation = 0%)
   - Halving mints fixed reward → `fee_collector` → distribution module → validators + delegators
   - Schedule: 4 OAS/block (0–10M) → 2 (10M–20M) → 1 (20M–30M) → 0.5 (30M+)

2. **Transaction Fees** — Gas fees from all chain transactions
   - Collected in `fee_collector`, distributed proportionally to stake

3. **Protocol Fees** — Custom module fees routed to `fee_collector`
   - Settlement escrow release: 5% protocol + 3% treasury → fee_collector
   - Datarights sell: 5% protocol fee → fee_collector
   - Work task settlement: 5% protocol share → fee_collector

**Staking**: `BondDenom = "uoas"`, `MaxValidators = 100`, `UnbondingTime = 21 days`

**Slashing**: `SignedBlocksWindow = 10000`, `MinSignedPerWindow = 5%`
- Downtime: 0.01% slash
- Double-sign: 5% slash

**Governance**: `MinDeposit = 100 OAS`, `VotingPeriod = 7 days`, `Quorum = 25%`, `Threshold = 66.7%`

### Critical: goleveldb replace directive
The go.mod MUST include this replace (same as SDK v0.50.10):
```
replace github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
```
Without it, the newer goleveldb silently loses data — IAVL stores appear to work in memory but nothing persists to disk.

### Running a local node
```bash
build/oasyced init test-node --chain-id oasyce-local-1
build/oasyced keys add validator --keyring-backend test
build/oasyced genesis add-genesis-account validator 1000000000uoas --keyring-backend test
build/oasyced genesis gentx validator 500000000uoas --chain-id oasyce-local-1 --keyring-backend test
build/oasyced genesis collect-gentxs
# Patch genesis: lower min_provider_stake to 0
python3 -c "import json; g=json.load(open('~/.oasyced/config/genesis.json')); g['app_state']['oasyce_capability']['params']['min_provider_stake']={'denom':'uoas','amount':'0'}; json.dump(g,open('~/.oasyced/config/genesis.json','w'),indent=2)"
# Enable API: sed -i 's/enable = false/enable = true/' ~/.oasyced/config/app.toml
build/oasyced start --minimum-gas-prices 0uoas
```

### E2E Testing
```bash
# With chain running:
./scripts/e2e_test.sh
```

## Key Patterns

### Proto field name mapping
- `ID` → `Id`, `AssetID` → `AssetId`, `CapabilityID` → `CapabilityId`
- `EscrowID` → `EscrowId`, `EndpointURL` → `EndpointUrl`
- `InvocationID` → `InvocationId`, `DisputeID` → `DisputeId`

### Keeper marshal pattern
```go
bz, err := k.cdc.Marshal(&obj)
store.Set(key, bz)
```

### Module interface pattern
```go
// types/codec.go — RegisterInterfaces with msgservice
// keeper/msg_server.go — implements types.MsgServer
// keeper/query_server.go — implements types.QueryServer
// module.go — RegisterServices wires both servers
// cli/tx.go — GetTxCmd(), cli/query.go — GetQueryCmd()
```

### CLI wiring
ModuleBasics.AddTxCommands panics (distr/staking need AddressCodec). Add custom module CLI commands individually in `cmd/oasyced/cmd/root.go`.

### Context conventions
- BankKeeper interfaces use `context.Context` (Cosmos SDK v0.50 convention)
- Internal keeper methods use `sdk.Context`
- Settlement/capability keeper interfaces that cross module boundaries use `sdk.Context`

### TX broadcast: CheckTx vs DeliverTx
`oasyced tx ... --yes` returns CheckTx result (code 0 = accepted into mempool). Use `curl localhost:26657/block_results?height=N` to check DeliverTx code. `query tx <hash>` fails due to type URL resolution bug — use block_results instead.

## DONE: Economic Model Review (2026-03-25)

Comprehensive economic model audit and parameter adjustments:
- Fee split unified: 90% provider, 5% protocol, 2% burn, 3% treasury (settlement + work aligned) ✅
- Slashing relaxed: 0.01% downtime (was 1%), 10000 block window (was 100) ✅
- Governance lowered: 100 OAS deposit (was 1000), 25% quorum (was 40%) ✅
- Reputation cooldown: 3600s (was 60s) ✅
- ProtocolFeeRate = 0.05 (was 0.03), TreasuryRate = 0.03 (was 0.02) ✅
- Bonding curve slippage protection already exists (min_shares_out / min_payout_out) ✅
- All 50+ tests updated and passing ✅

## DONE: Airdrop Halving Economics (2026-03-24)

Airdrop and PoW difficulty now scale with total registrations (x/onboarding/keeper/keeper.go):

| Epoch | Cumulative Registrations | Airdrop | PoW Difficulty |
|-------|--------------------------|---------|----------------|
| 0 | 0 – 10,000 | 20 OAS | 16 bits |
| 1 | 10,001 – 50,000 | 10 OAS | 18 bits |
| 2 | 50,001 – 200,000 | 5 OAS | 20 bits |
| 3 | 200,001+ | 2.5 OAS | 22 bits |

- `total_registrations` counter at `TotalRegistrationsKey = 0x03`
- Effective difficulty = max(params, HalvingDifficulty(epoch))
- Effective airdrop = min(params, HalvingAirdrop(epoch))
- ConsensusVersion bumped to 3

## DONE: Block Reward Halving (2026-03-24)

Replaced standard Cosmos SDK 5% inflation with custom x/halving module. Standard mint module inflation set to 0%.

| Block Range | Reward | Cumulative Supply |
|-------------|--------|-------------------|
| 0 – 10,000,000 | 4 OAS/block | 40M OAS |
| 10,000,001 – 20,000,000 | 2 OAS/block | 60M OAS |
| 20,000,001 – 30,000,000 | 1 OAS/block | 70M OAS |
| 30,000,001+ | 0.5 OAS/block | +~3.15M/year |

Combined with the 2% burn rate, this creates a supply curve that peaks and then contracts.

- Module: `x/halving/` (no proto, no store, no CLI — purely deterministic from block height)
- `keeper.BlockReward(height)` returns uoas reward for any block height
- BeginBlocker: mint → halving module account → fee_collector → distribution → validators
- Runs after standard mint (which mints 0) and before distribution module
- Module account has `Minter` permission
- 13 tests (boundary conditions, halving transitions, cumulative supply)
