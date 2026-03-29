#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

export START_NODE=1
exec bash "$SCRIPT_DIR/bootstrap_public_beta_node.sh" "$@"
