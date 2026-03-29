#!/bin/bash
# Thin compatibility wrapper to keep a single healthcheck implementation.

SOURCE_PATH="${BASH_SOURCE[0]:-$0}"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$SOURCE_PATH")" && pwd)"
CANONICAL_SCRIPT="${SCRIPT_DIR}/../deploy/healthcheck.sh"

if [ ! -f "$CANONICAL_SCRIPT" ]; then
    echo "Canonical healthcheck script not found: $CANONICAL_SCRIPT" >&2
    exit 1
fi

if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    # shellcheck disable=SC1090
    . "$CANONICAL_SCRIPT"
else
    exec bash "$CANONICAL_SCRIPT" "$@"
fi
