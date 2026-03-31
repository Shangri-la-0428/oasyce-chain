#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: scripts/ecs_service_restart.sh <systemd-unit>" >&2
  exit 1
fi

UNIT="$1"
case "${UNIT}" in
  oasyced|oasyce-faucet|oasyce-provider|claude-proxy|thronglets|nginx|ssh)
    ;;
  *)
    echo "Refusing to restart unexpected unit: ${UNIT}" >&2
    exit 1
    ;;
esac

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${ROOT_DIR}/ecs_cloud_run.sh" "systemctl restart ${UNIT} && sleep 3 && systemctl status ${UNIT} --no-pager | tail -30"
