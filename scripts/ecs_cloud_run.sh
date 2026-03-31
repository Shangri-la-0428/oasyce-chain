#!/usr/bin/env bash
set -euo pipefail

ALIYUN_BIN="${ALIYUN_BIN:-/opt/homebrew/bin/aliyun}"
ALIYUN_PROFILE="${ALIYUN_PROFILE:-oasyce}"
ALIYUN_REGION="${ALIYUN_REGION:-cn-beijing}"
ALIYUN_INSTANCE_ID="${ALIYUN_INSTANCE_ID:-i-2ze3c737ux27bp7j38rq}"
POLL_INTERVAL="${POLL_INTERVAL:-2}"
POLL_MAX="${POLL_MAX:-60}"

usage() {
  cat <<'EOF'
Run a shell command on the Oasyce ECS instance through Alibaba Cloud Assistant.

Usage:
  scripts/ecs_cloud_run.sh 'hostname && whoami'
  echo 'journalctl -u ssh --since "10 min ago" | tail -50' | scripts/ecs_cloud_run.sh

Environment overrides:
  ALIYUN_BIN
  ALIYUN_PROFILE
  ALIYUN_REGION
  ALIYUN_INSTANCE_ID
  POLL_INTERVAL
  POLL_MAX
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd python3
require_cmd jq

if [[ ! -x "${ALIYUN_BIN}" ]]; then
  echo "Alibaba Cloud CLI not found or not executable: ${ALIYUN_BIN}" >&2
  exit 1
fi

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -gt 0 ]]; then
  COMMAND_CONTENT="$*"
else
  if [[ -t 0 ]]; then
    usage
    exit 1
  fi
  COMMAND_CONTENT="$(cat)"
fi

if [[ -z "${COMMAND_CONTENT}" ]]; then
  echo "Command content cannot be empty." >&2
  exit 1
fi

COMMAND_B64="$(printf '%s' "${COMMAND_CONTENT}" | base64 | tr -d '\n')"

run_aliyun() {
  "${ALIYUN_BIN}" --profile "${ALIYUN_PROFILE}" "$@"
}

RUN_JSON="$(run_aliyun ecs RunCommand \
  --RegionId "${ALIYUN_REGION}" \
  --Type RunShellScript \
  --ContentEncoding Base64 \
  --CommandContent "${COMMAND_B64}" \
  --InstanceId.1 "${ALIYUN_INSTANCE_ID}" \
  --Timeout 600 \
  --KeepCommand false)"

INVOKE_ID="$(printf '%s' "${RUN_JSON}" | jq -r '.InvokeId')"

if [[ -z "${INVOKE_ID}" || "${INVOKE_ID}" == "null" ]]; then
  echo "Failed to obtain InvokeId from RunCommand response." >&2
  printf '%s\n' "${RUN_JSON}" >&2
  exit 1
fi

echo "InvokeId: ${INVOKE_ID}" >&2

RESULT_JSON=""
for ((i=1; i<=POLL_MAX; i++)); do
  RESULT_JSON="$(run_aliyun ecs DescribeInvocationResults \
    --RegionId "${ALIYUN_REGION}" \
    --InvokeId "${INVOKE_ID}" \
    --IncludeHistory true)"

  STATUS="$(printf '%s' "${RESULT_JSON}" | jq -r '.Invocation.InvocationResults.InvocationResult[0].InvocationStatus // empty')"
  if [[ -n "${STATUS}" ]]; then
    echo "Status: ${STATUS}" >&2
  fi

  if [[ "${STATUS}" == "Success" || "${STATUS}" == "Failed" || "${STATUS}" == "Stopped" ]]; then
    break
  fi
  sleep "${POLL_INTERVAL}"
done

STATUS="$(printf '%s' "${RESULT_JSON}" | jq -r '.Invocation.InvocationResults.InvocationResult[0].InvocationStatus // empty')"
EXIT_CODE="$(printf '%s' "${RESULT_JSON}" | jq -r '.Invocation.InvocationResults.InvocationResult[0].ExitCode // 1')"
OUTPUT_B64="$(printf '%s' "${RESULT_JSON}" | jq -r '.Invocation.InvocationResults.InvocationResult[0].Output // empty')"
ERROR_INFO="$(printf '%s' "${RESULT_JSON}" | jq -r '.Invocation.InvocationResults.InvocationResult[0].ErrorInfo // empty')"

if [[ -n "${OUTPUT_B64}" ]]; then
  python3 - "${OUTPUT_B64}" <<'PY'
import base64
import sys

payload = sys.argv[1]
if payload:
    sys.stdout.write(base64.b64decode(payload).decode("utf-8", errors="replace"))
PY
fi

if [[ "${STATUS}" != "Success" ]]; then
  if [[ -n "${ERROR_INFO}" ]]; then
    echo "Cloud Assistant error: ${ERROR_INFO}" >&2
  fi
  exit "${EXIT_CODE}"
fi
