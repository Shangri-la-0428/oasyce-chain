#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: scripts/ecs_tail_unit.sh <systemd-unit>" >&2
  exit 1
fi

UNIT="$1"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${ROOT_DIR}/ecs_cloud_run.sh" "journalctl -u ${UNIT} --since '15 min ago' --no-pager | tail -200"
