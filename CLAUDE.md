# Oasyce L1 Chain

## Project
Cosmos SDK v0.50.10 chain at `/Users/wutongcheng/Desktop/oasyce-chain` with 5 custom modules: settlement, capability, reputation, datarights, work.

## Current Status

### Protobuf Migration — COMPLETE ✅
- All 5 modules fully protobuf-migrated (including x/work)
- app.go wired with real keepers, store keys, module manager

### REST/gRPC Integration — COMPLETE ✅
- REST API on :1317, gRPC on :9090
- gRPC-Gateway routes registered for all 4 custom modules
- Node starts, produces blocks, persists data, restarts cleanly
- All standard Cosmos queries (bank, auth, staking) work
- All custom module queries work

### CLI Commands — COMPLETE ✅
- All 4 modules have CLI tx + query commands (`x/*/cli/`)
- `oasyced tx datarights register|buy-shares|file-dispute|resolve-dispute`
- `oasyced tx settlement create-escrow|release-escrow|refund-escrow`
- `oasyced tx oasyce_capability register|invoke`
- `oasyced tx reputation submit-feedback|report`
- `oasyced tx send` (bank transfers)
- Query commands: `oasyced query <module> <subcommand>`

### Chain Upgrades — COMPLETE ✅
Five economic/governance upgrades implemented and tested:

1. **Bancor Bonding Curve** — Replaced tiered pricing with continuous curve
   - `tokens = supply × (√(1 + payment/reserve) − 1)`, CW=0.5
   - Bootstrap: `tokens = payment / INITIAL_PRICE` when reserve=0
   - Files: `x/settlement/keeper/bonding_curve.go`, `x/datarights/keeper/keeper.go`

2. **2% Token Burn** — Escrow release now splits: 93% provider, 5% protocol fee, 2% burn
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
All 4 modules verified with real transactions:
- **datarights**: register asset, buy shares (Bancor curve), sell shares, access gating, jury voting
- **settlement**: create escrow (LOCKED), release escrow (RELEASED, 5% fee + 2% burn)
- **capability**: register capability, invoke (creates escrow + invocation)
- **reputation**: submit feedback (score=450), leaderboard populated
- **bank**: cross-account transfers work
- E2E test script: `scripts/e2e_test.sh`

### Known Genesis Param Issues
- `oasyce_capability.min_provider_stake`: defaults to 10B uoas (10000 OAS) — set to 0 for testnet
- `datarights.dispute_deposit`: 1B uoas (1000 OAS) — high for testing
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

### Build & Test Status
```
go build ./...  ✅
go test ./...   ✅ (40+ tests across 6 suites)
  tests/integration     — 3 tests (full capability flow, Bancor curve, escrow lifecycle)
  x/capability/keeper   — capability tests
  x/datarights/keeper   — 12 tests (Bancor buy/sell, access gating, jury voting)
  x/reputation/keeper   — reputation tests
  x/settlement/keeper   — escrow + bonding curve tests
  x/work/keeper         — 13 tests (executor, task CRUD, commit-reveal, assignment, settlement, minority penalty)
```

### Protocol Constants (x/settlement/types/types.go)
```go
ReserveRatio       = 0.5   // Bancor connector weight
InitialPrice       = 1.0   // 1 uoas per token at bootstrap
ReserveSolvencyCap = 0.95  // Max 95% reserve payout on sell
BurnRate           = 0.02  // 2% burn on escrow release
```

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
