# v0.8.0 Proposal-Ready Checklist

`v0.8.0` is the Template-Height software-upgrade package for the `x/sigil` v1 -> v2 migration.

This package is proposal-ready, not release-ready:

- do render and dry-run in-repo
- do not lock a real upgrade height here
- do not attach final checksums or rollback steps here

## What This Upgrade Does

- upgrade `x/sigil` from consensus version `1` to `2`
- rebuild active liveness indexes from `LastActiveHeight` semantics to `MaxPulseHeight()` semantics
- keep the migration state-only
- add no new module stores

## Architecture Note — Rehearsal Scope

Because this upgrade is state-only (see above), the interesting rehearsal
question is whether `Migrate1to2` behaves correctly on **real state shape**,
not whether cosmos-sdk's governance path still works.

**In scope for local rehearsal**:

- `Migrate1to2` correctness on a testnet-derived v0.7.0 state export, not
  the synthetic two-sigil fixture in `x/sigil/keeper/migration_test.go`
- Post-migration invariants: `active_count` preserved, every sigil lives in
  exactly one bucket at the correct `MaxPulseHeight`, no orphan index
  entries, no dropped records

**Out of scope for local rehearsal**:

- Governance proposal voting — no new signal over cosmos-sdk's own tests
- Binary swap / cosmovisor auto-restart — no new store keys, standard path
- Any patch to `voting_period`, `quorum`, `threshold`, or `min_deposit` in
  genesis.json. Do not reshape gov params to make local pass faster; that
  would test a system that does not exist in production, and the rehearsal
  loses its meaning.

**Elegant migration rehearsal flow**:

1. `oasyced export` on the testnet seed node → `v070-state.json`
2. Drop the export into a fresh local `~/.oasyced/config/genesis.json`
3. Add `TestUpgradeV080_WithRealSeedFixture` in `app/upgrades_test.go` that
   loads the real fixture, calls `app.UpgradeKeeper.ApplyUpgrade`, and
   asserts the invariants above
4. No chain start, no gov proposal, no binary swap

**Governance exercise runs once, in its natural environment**: the real
v0.8.0 upgrade on the testnet seed node, using production gov parameters.
If you want an expedited path, use `--expedited` with an
`expedited_voting_period` that matches what would land in mainnet genesis.
That is a configuration decision belonging in the mainnet genesis file, not
a rehearsal hack.

## Preflight Before Filling Height

```bash
curl -s http://<node>:1317/health | jq
oasyced tx sigil --help
python3 scripts/check_pulse_compat.py --sdk-mode source
python3 scripts/live_gate_local.py
```

Expected:

- `/health` reachable
- `oasyced tx sigil pulse` present
- Pulse compatibility returns `chain-ready`, `thronglets-ready`, `sdk-ready`
- `live_gate_local.py` returns `status: ok`

## Render Proposal Artifacts

```bash
python3 scripts/upgrade_proposal_v080.py render \
  --height <candidate-height> \
  --proposal-output /tmp/v080-proposal.json \
  --metadata-output /tmp/v080-metadata.json
```

## Validate Rendered Proposal

```bash
python3 scripts/upgrade_proposal_v080.py validate /tmp/v080-proposal.json
```

## Dry-Run on Tempnet

```bash
python3 scripts/upgrade_proposal_v080.py dry-run \
  --height <candidate-height> \
  --network tempnet
```

Expected:

- rendered proposal passes local validation
- `oasyced tx gov submit-proposal ... --dry-run` succeeds
- on-chain `metadata` stays a short reference string; the full metadata body lives in `/tmp/v080-metadata.json`

## Real Submission Command Shape

After choosing a real height and completing preflight:

```bash
oasyced tx gov submit-proposal /tmp/v080-proposal.json \
  --from <signer> \
  --chain-id <chain-id> \
  --fees 20000uoas \
  --yes \
  --output json
```

## Post-Upgrade Checks

```bash
curl -s http://<node>:1317/health | jq '.module_versions'
oasyced tx sigil --help
python3 scripts/check_pulse_compat.py --sdk-mode source
python3 scripts/live_gate_local.py
```

Expected:

- `/health` shows `sigil == 2`
- `/health` continues to show `anchor` / `delegate` / `sigil`
- `oasyced tx sigil pulse` still exists
- `live_gate_local.py` still returns `status: ok`
