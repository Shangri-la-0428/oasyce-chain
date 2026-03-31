#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${ROOT_DIR}/ecs_cloud_run.sh" '
echo "=== alert log ==="
tail -n 50 /var/log/oasyce-alert.log 2>/dev/null || true
echo
echo "=== healthcheck state ==="
find /var/lib/oasyce-healthcheck -maxdepth 2 -type f -print -exec cat {} \; 2>/dev/null || true
echo
echo "=== consumer state ==="
cat /var/lib/oasyce-consumer/state.json 2>/dev/null || true
'
