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
