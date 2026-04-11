# v0.8.0 Upgrade Checklist

`v0.8.0` carries the `x/sigil` v1 -> v2 migration:

- rebuild active liveness indexing from `LastActiveHeight` semantics to `MaxPulseHeight()` semantics
- rebuild the dormant `0x09` bucket so Phase 1 dissolve stays `O(expiring)`
- keep the upgrade **state migration only**
- add **no new store keys**

This checklist is split into the two execution contexts that matter:

1. **Local fixture migration replay** using a copied pre-upgrade VPS node home
2. **VPS real upgrade execution** using the actual seed node

Do not reshape governance params locally just to make rehearsal faster. Local rehearsal is about proving the migration on real state shape, not about re-testing Cosmos governance.

## Local Fixture Migration Replay

### Fixture rules

- input must be a copied **node home**, not exported genesis
- keep the copied fixture outside the live VPS and outside `~/.oasyced`
- the replay tool copies the fixture again into a repo-local temp dir before applying `UpgradeV080`

### Preflight

- confirm the fixture really comes from a **pre-upgrade** node
- confirm the copied home contains `data/application.db`
- confirm the fixture has legacy `sigil` module version `0` or `1`

### Run

```bash
go run ./tools/v080_fixture_audit audit-home \
  --home /path/to/copied-vps-home \
  --output ./tmp/reports/v080-audit-before.json

go run ./tools/v080_fixture_audit replay-v080 \
  --source-home /path/to/copied-vps-home \
  --working-home ./tmp/v080-replay \
  --output ./tmp/reports/v080-replay-report.json
```

### Pass criteria

- `before.active_count == after.active_count`
- every `active` sigil appears exactly once in `0x05`
- every `active` sigil is indexed at `MaxPulseHeight(s)`
- every `dormant` sigil appears exactly once in `0x09`
- every `dormant` sigil is indexed at its frozen `MaxPulseHeight(s)`
- no `dissolved` sigil appears in `0x05` or `0x09`
- no orphan index entries
- no sigil appears in both liveness buckets
- post-replay `module_version == 2`

### Record here

- fixture source: `root@47.93.32.88:/home/oasyce/.oasyced/data/application.db` copied via VPS temp fixture `/tmp/oasyce-pre-v080-fixture`
- fixture copy path: `./tmp/vps-pre-v080-home-full`
- before report path: `./tmp/reports/v080-audit-before.json`
- replay report path: `./tmp/reports/v080-replay-report.json`
- replay status: `ok`
- invariant errors: `[]`

## VPS Real Upgrade Execution

### Binary preparation

Build locally and copy to the VPS. Do **not** fetch build dependencies on the China VPS.

```bash
GOOS=linux GOARCH=amd64 go build -o ./tmp/oasyced-v0.8.0 ./cmd/oasyced
shasum -a 256 ./tmp/oasyced-v0.8.0
scp -P 29222 ./tmp/oasyced-v0.8.0 root@47.93.32.88:/tmp/oasyced-v0.8.0
```

### Pre-upgrade checks on VPS

- `/health` reachable
- current binary path, service user, and `--home` confirmed
- proposal JSON and metadata reviewed before submission

Recommended checks:

```bash
systemctl show oasyced -p User,ExecStart
curl -s http://127.0.0.1:11317/health | jq
```

### Governance execution

- submit the real software-upgrade proposal
- vote with the intended validator/operator set
- `--expedited` is allowed **only if** you intend to use the existing chain-side expedited governance path
- do not patch `min_deposit`, `quorum`, `voting_period`, or other gov params just for this upgrade

### Post-upgrade checks on VPS

- `/health` returns `module_versions.sigil == 2`
- `anchor`, `delegate`, `sigil` versions remain visible
- `oasyced tx sigil pulse` still exists
- at least one replay-selected canary sigil has transitioned from `active` to `dormant` on the upgraded chain
- treat this P0 smoke's **Phase 1** as `active -> dormant`; final `dormant -> dissolved` remains a longer soak check
- no orphan liveness index entries found in audit follow-up

Recommended checks:

```bash
curl -s http://127.0.0.1:11317/health | jq '.module_versions'
/usr/local/bin/oasyced tx sigil pulse --help
```

### Record here

- binary SHA256: `27aa3bf87c0e7af3ba4d73f5c4b17f18c5b8951fb992d43581646cf5753b615e` (`./tmp/oasyced-v0.8.0`); previous live binary SHA: `f16b15429209f7407a751a1d12e470631da7851d67b3fa320fbf3dbe327be583`
- binary path on VPS: `/usr/local/bin/oasyced` (live path), staged from `/tmp/oasyced-v0.8.0`
- proposal id: `1`
- expedited used (`yes/no`): `yes`
- upgrade height: `192884`
- pre-upgrade `/health` snapshot path: `./tmp/reports/health-pre-v080.json`
- current live status snapshots: `./tmp/reports/health-current.json`, `./tmp/reports/proposal-1-current.json`, `./tmp/reports/proposal-1-votes-current.json`, `./tmp/reports/status-current.json`, `./tmp/reports/tx-sigil-help-current.txt`
- post-upgrade `/health` snapshot path: `pending (proposal #1 still voting as of 2026-04-12 07:18 CST; current height=168495; voting_end=2026-04-12T22:31:40.795581177Z, then wait for height 192884)`
- `sigil == 2` confirmed (`yes/no`): `pending`
- Phase 1 `active -> dormant` observed at height: `pending`
- notes / anomalies:
  - real fixture replay exposed legacy `sigil` module version map = `0`; repo tooling/handler updated to treat live store as v1 before running `1 -> 2`
  - live node initially could not parse `/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade`, and `/usr/local/bin/oasyced tx sigil` had no `pulse`; on this single-validator testnet, `v0.8.0` binary was deployed to `/usr/local/bin/oasyced` before proposal submission so the chain could accept the governance proposal
  - proposal submit txhash: `04ACEAE4BE92BBC6A419B9167F8C617FF500CA030E2B96069D531AB65C4A2816`
  - vote txhash: `FC3A10A33FCB62CA2D45246D94E02B451747090F6B139CE39DE07DE2EF9EF3B2`
  - proposal state snapshots: initial=`./tmp/reports/proposal-1.json`, latest=`./tmp/reports/proposal-1-current.json`; votes: initial=`./tmp/reports/proposal-1-votes.json`, latest=`./tmp/reports/proposal-1-votes-current.json`
  - current live chain snapshots: `./tmp/reports/health-current.json`, `./tmp/reports/status-current.json`; current live CLI snapshot: `./tmp/reports/tx-sigil-help-current.txt`
  - immediate post-upgrade dormant canaries from replay: `SIG_2388db3b706e395f4c2439e07883e8ab`, `SIG_27f9b72f64ead2bf91d143aea5857c96`, `SIG_34131bf6cdb30c240fa1fcc46faf42f9` (all `blocks_until_dormant_after_upgrade = 0`)
