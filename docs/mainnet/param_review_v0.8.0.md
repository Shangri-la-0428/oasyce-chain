# v0.8.0 Mainnet Parameter Review

Each line records the current value, the proposed mainnet value, and the reason. `no change` is explicit when the current setting still matches the intended ecology after the `MaxPulseHeight()` migration.

| Parameter | Current | Proposed | Reason |
| --- | --- | --- | --- |
| `sigil.dormant_threshold` | `100000` blocks | `no change` | After `v0.8.0`, the threshold now measures inactivity against `MaxPulseHeight()`. Keeping roughly six days preserves the intended grace window while real pulse cadence is still being observed. |
| `sigil.dissolve_threshold` | `1000000` blocks | `no change` | The post-dormant pruning window remains intentionally long so pruning stays meaningful rather than becoming an operator footgun during early mainnet. |
| `capability.min_provider_stake` | `0uoas` | `5000000uoas` | Mainnet should not allow zero-cost provider spam once real OAS is at risk. A 5 OAS floor is still low enough for early providers while reintroducing economic friction. |
| `datarights.dispute_deposit` | `10000000uoas` | `no change` | A 10 OAS dispute bond is already large enough to discourage low-quality disputes without freezing legitimate complainants out of the process. |
| `gov.min_deposit` | `100000000uoas` | `no change` | 100 OAS is still a reasonable proposal threshold for an early-stage validator set; raising it now would reduce governance reach before participation hardens. |
| `gov.quorum` | `25%` | `no change` | The validator set is still expected to be small during the first mainnet era, so the current quorum remains more realistic than a stricter threshold. |
| `slashing.signed_blocks_window` | `10000` blocks | `no change` | Keeping a longer observation window still makes sense for a geographically distributed, low-validator-count network where brief outages should not dominate slash outcomes. |
| `slashing.min_signed_per_window` | `5%` | `50%` | `5%` is too forgiving for public mainnet because a validator can miss nearly the whole window and remain unslashed. Raising to `50%` materially tightens liveness guarantees without jumping straight to mature-network severity. |
