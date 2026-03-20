#!/bin/bash
# Start all 4 validators of the local testnet from a single terminal.
#
# Each node runs as a background process. Logs go to ~/.oasyce-localnet/nodeN.log.
# Press Ctrl+C to stop all nodes.
#
set -euo pipefail

NUM_VALIDATORS=4
BINARY="${BINARY:-oasyced}"
BASE_DIR="$HOME/.oasyce-localnet"

# Verify the testnet has been initialized.
if [ ! -d "$BASE_DIR/node0" ]; then
  echo "Error: Testnet not initialized. Run 'bash scripts/init_multi_testnet.sh' first."
  exit 1
fi

# Array of background PIDs so we can clean up on exit.
PIDS=()

cleanup() {
  echo ""
  echo "==> Stopping all nodes..."
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
  # Wait briefly for graceful shutdown, then force kill stragglers.
  sleep 2
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
  done
  echo "All nodes stopped."
}

trap cleanup EXIT INT TERM

echo "==> Starting ${NUM_VALIDATORS}-validator local testnet..."
echo ""

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="$BASE_DIR/node${i}"
  LOG_FILE="$BASE_DIR/node${i}.log"

  echo "    Starting node${i} (log: ${LOG_FILE})"
  $BINARY start --home "$NODE_HOME" > "$LOG_FILE" 2>&1 &
  PIDS+=($!)
done

echo ""
echo "All nodes started. PIDs: ${PIDS[*]}"
echo "Logs: $BASE_DIR/node*.log"
echo ""
echo "Press Ctrl+C to stop all nodes."
echo ""

# Wait for any child to exit (if a node crashes, we'll notice).
wait -n 2>/dev/null || true

# If we get here, at least one node exited. Show which ones are still running.
echo ""
echo "Warning: A node exited unexpectedly. Checking status..."
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  if kill -0 "${PIDS[$i]}" 2>/dev/null; then
    echo "    node${i} (PID ${PIDS[$i]}): running"
  else
    echo "    node${i} (PID ${PIDS[$i]}): STOPPED (check $BASE_DIR/node${i}.log)"
  fi
done

# Keep running until user hits Ctrl+C.
echo ""
echo "Remaining nodes still running. Press Ctrl+C to stop all."
wait 2>/dev/null || true
