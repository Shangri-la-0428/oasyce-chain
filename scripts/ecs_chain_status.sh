#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${ROOT_DIR}/ecs_cloud_run.sh" '
echo "=== Services ==="
systemctl is-active oasyced oasyce-faucet oasyce-provider
echo
echo "=== Block ==="
STATUS_JSON="$(curl -fsS http://127.0.0.1:26667/status || curl -fsS http://127.0.0.1:26657/status)"
printf "%s" "$STATUS_JSON" | python3 -c "import json,sys; d=json.load(sys.stdin)[\"result\"][\"sync_info\"]; print(\"height:\", d[\"latest_block_height\"]); print(\"catching_up:\", d[\"catching_up\"])"
echo
echo "=== REST health ==="
curl -s http://127.0.0.1:11317/health || true
'
