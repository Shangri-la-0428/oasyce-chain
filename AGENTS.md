# Oasyce Chain — AI Agent Integration Guide

> This file is the source of truth for AI tool integration. It is read automatically by Claude Code (CLAUDE.md), Cursor (.cursorrules), Windsurf (.windsurfrules), and any AI tool that supports project-level instructions.

Oasyce Chain is the L1 Cosmos SDK appchain. For the Python client and full product experience, install `pip install oasyce` and see the [Plugin Engine AGENTS.md](https://github.com/Shangri-la-0428/Oasyce_Claw_Plugin_Engine/blob/main/AGENTS.md).

## Chain CLI (`oasyced`)

```bash
# Data rights
oasyced tx datarights register --name "Dataset" --content-hash abc123 --rights-type 1 --from alice
oasyced tx datarights buy-shares --asset-id DATA_xxxx --amount 1000000uoas --from bob
oasyced tx datarights sell-shares --asset-id DATA_xxxx --shares 100 --from bob
oasyced tx datarights file-dispute --asset-id DATA_xxxx --reason "..." --from bob
oasyced tx datarights resolve-dispute --asset-id DATA_xxxx --verdict upheld --from alice

# Settlement
oasyced tx settlement create-escrow --provider cosmos1xxx --amount 1000000uoas --from alice
oasyced tx settlement release-escrow --escrow-id ESC_xxxx --from alice
oasyced tx settlement refund-escrow --escrow-id ESC_xxxx --from alice

# AI Capabilities
oasyced tx oasyce_capability register --name "Translation API" --endpoint "https://..." --price 500000uoas --from provider
oasyced tx oasyce_capability invoke --capability-id CAP_xxxx --from consumer

# Reputation
oasyced tx reputation submit-feedback --target cosmos1xxx --score 5 --from alice
oasyced query reputation show cosmos1xxx

# Proof of Useful Work (x/work)
oasyced tx work register-executor --task-types "ml_training,inference" --from validator
oasyced tx work submit-task --task-type ml_training --input-hash abc123 --bounty 1000000uoas --from alice
oasyced tx work commit-result --task-id TASK_xxxx --commit-hash <sha256> --from executor
oasyced tx work reveal-result --task-id TASK_xxxx --output-hash abc123 --salt mysalt --from executor
oasyced tx work dispute-result --task-id TASK_xxxx --reason "..." --from disputer

# Governance
oasyced tx gov submit-proposal --title "..." --description "..." --deposit 10000000uoas --from alice
oasyced tx gov vote 1 yes --from alice
oasyced query gov proposals
```

## Build & Run

```bash
make build
./scripts/init_testnet.sh
./scripts/start_testnet.sh
# RPC :26657 | REST :1317 | gRPC :9090
make test
```

## Modules

| Module | Purpose |
|--------|---------|
| x/datarights | Data asset registry, Bancor curve, jury voting |
| x/settlement | Escrow lifecycle, 2% burn, fee distribution |
| x/capability | AI capability endpoints, invocation via escrow |
| x/reputation | Feedback scoring, time decay, leaderboard |
| x/work | PoUW: AI compute tasks, commit-reveal, redundant execution |

## For Development

See [CLAUDE.md](CLAUDE.md) for detailed keeper patterns, proto mappings, build instructions, and known issues.
