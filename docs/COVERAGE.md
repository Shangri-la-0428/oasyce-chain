# Test Coverage Report

> Generated: 2026-03-20

## Summary

| Module | Package | Coverage |
|--------|---------|----------|
| x/datarights | keeper | **63.1%** |
| x/settlement | keeper | **62.2%** |
| x/reputation | keeper | **60.6%** |
| x/capability | keeper | **47.7%** |

All tests pass: `go test ./... -race` ✅

## Key Function Coverage

### x/datarights/keeper

| Function | Coverage |
|----------|----------|
| RegisterDataAsset | 80.0% |
| BuyShares | 81.6% |
| SellShares | 74.6% |
| DelistAsset | 80.0% |
| FileDispute | 80.8% |
| GetAccessLevel | 92.3% |
| SelectJury | 88.5% |
| SubmitJuryVote | 88.2% |
| TallyVotes | 100.0% |
| ResolveByJury | 69.8% |
| ResolveDispute (authority) | 35.4% |

### x/settlement/keeper

| Function | Coverage |
|----------|----------|
| CreateEscrow | 73.9% |
| ReleaseEscrow | 81.2% |
| RefundEscrow | 68.4% |
| ExpireStaleEscrows | 66.7% |
| BancorBuy | 80.0% |
| SpotPrice | 75.0% |
| BuyShares | 77.8% |

### x/reputation/keeper

| Function | Coverage |
|----------|----------|
| SubmitFeedback | 88.5% |
| UpdateScore | 90.3% |
| ReportMisbehavior | 70.0% |

### x/capability/keeper

| Function | Coverage |
|----------|----------|
| RegisterCapability | 56.2% |
| InvokeCapability | 82.4% |
| CompleteInvocation | 75.0% |
| FailInvocation | 72.7% |
| DeactivateCapability | 80.0% |

## Coverage Gaps (Priority)

1. **ResolveDispute (authority path)** — 35.4%: needs more edge case tests
2. **BancorSell (settlement)** — 0%: sell logic tested via datarights keeper, not directly
3. **ListCapabilities** — 0%: tag-based filtering untested
4. **Genesis init/export** — 0%: all modules lack genesis round-trip tests

## Running Coverage

```bash
# Generate coverage profile
go test ./... -coverprofile=coverage.out -covermode=atomic

# View in browser
go tool cover -html=coverage.out

# Print summary
go tool cover -func=coverage.out | tail -1
```
