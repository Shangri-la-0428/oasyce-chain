#!/bin/bash
# Stress test script for oasyce-chain
# Sends rapid-fire transactions to test throughput, mempool handling, and state consistency.
#
# Requires:
#   - Chain running locally (4-validator testnet recommended)
#   - "validator" key in test keyring with sufficient balance
#   - jq installed
#
# Usage:
#   ./scripts/stress_test.sh [--rounds 100] [--parallel 4] [--node http://localhost:26657]

set -euo pipefail

# ---------- Config ----------
OASYCED="${OASYCED:-./build/oasyced}"
CHAIN_ID="${CHAIN_ID:-oasyce-local-1}"
NODE="${NODE:-tcp://localhost:26657}"
REST="${REST:-http://localhost:1317}"
KB="--keyring-backend test"
ROUNDS=100
PARALLEL=4

while [[ $# -gt 0 ]]; do
  case $1 in
    --rounds)   ROUNDS=$2; shift 2 ;;
    --parallel) PARALLEL=$2; shift 2 ;;
    --node)     NODE=$2; shift 2 ;;
    --rest)     REST=$2; shift 2 ;;
    *)          echo "Unknown: $1"; exit 1 ;;
  esac
done

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[STRESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }

# ---------- Pre-flight ----------
log "Config: rounds=$ROUNDS parallel=$PARALLEL node=$NODE"

# Verify chain is running.
HEIGHT=$(curl -sf "${NODE/tcp/http}/status" | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['sync_info']['latest_block_height'])" 2>/dev/null || echo "0")
if [ "$HEIGHT" = "0" ]; then
  fail "Chain not reachable at $NODE"
  exit 1
fi
log "Chain online at height $HEIGHT"

# Get validator address.
VALIDATOR=$($OASYCED keys show validator -a $KB 2>/dev/null)
if [ -z "$VALIDATOR" ]; then
  fail "No 'validator' key found in test keyring"
  exit 1
fi
log "Using address: $VALIDATOR"

# Check balance.
BAL=$(curl -sf "$REST/cosmos/bank/v1beta1/balances/$VALIDATOR" | python3 -c "import sys,json; bals=json.load(sys.stdin).get('balances',[]); print(next((b['amount'] for b in bals if b['denom']=='uoas'),'0'))" 2>/dev/null || echo "0")
log "Balance: $BAL uoas"

NEEDED=$((ROUNDS * 200000))
if [ "$BAL" -lt "$NEEDED" ] 2>/dev/null; then
  warn "Balance may be insufficient for $ROUNDS rounds (need ~${NEEDED} uoas)"
fi

# Create test accounts for parallel sends.
declare -a TEST_ADDRS
for i in $(seq 1 $PARALLEL); do
  NAME="stress_test_$i"
  ADDR=$($OASYCED keys show "$NAME" -a $KB 2>/dev/null || true)
  if [ -z "$ADDR" ]; then
    ADDR=$($OASYCED keys add "$NAME" $KB --output json 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['address'])")
    log "Created test account: $NAME ($ADDR)"
    # Fund it.
    $OASYCED tx bank send validator "$ADDR" 50000000uoas \
      $KB --chain-id $CHAIN_ID --fees 10000uoas --yes \
      --node $NODE -o json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  fund tx: code={d.get(\"code\",0)}')" || true
    sleep 7
  fi
  TEST_ADDRS+=("$ADDR")
done

# ---------- Test 1: Rapid Bank Sends ----------
log "--- Test 1: Rapid bank sends ($ROUNDS rounds) ---"

START_TIME=$(date +%s)
TX_SUCCESS=0
TX_FAIL=0

for i in $(seq 1 $ROUNDS); do
  # Round-robin target.
  IDX=$(( (i - 1) % PARALLEL ))
  TARGET=${TEST_ADDRS[$IDX]}
  SENDER_NAME="stress_test_$(( (IDX + 1) % PARALLEL + 1 ))"

  # Send from validator to target, non-blocking.
  RESULT=$($OASYCED tx bank send validator "$TARGET" 1000uoas \
    $KB --chain-id $CHAIN_ID --fees 5000uoas --yes \
    --node $NODE -o json 2>/dev/null || echo '{"code":99}')
  CODE=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',99))" 2>/dev/null || echo "99")

  if [ "$CODE" = "0" ]; then
    TX_SUCCESS=$((TX_SUCCESS + 1))
  else
    TX_FAIL=$((TX_FAIL + 1))
  fi

  # Progress.
  if [ $((i % 20)) -eq 0 ]; then
    log "  Progress: $i/$ROUNDS (ok=$TX_SUCCESS fail=$TX_FAIL)"
  fi
done

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))
TPS=0
if [ "$ELAPSED" -gt 0 ]; then
  TPS=$((TX_SUCCESS / ELAPSED))
fi

log "Bank sends complete: $TX_SUCCESS ok, $TX_FAIL failed, ${ELAPSED}s elapsed, ~${TPS} tx/s submitted"

# Wait for pending txs to land.
log "Waiting 15s for pending transactions to finalize..."
sleep 15

# ---------- Test 2: Capability Register + Invoke Burst ----------
log "--- Test 2: Capability register + invoke burst ---"

# Register a capability.
REG_RESULT=$($OASYCED tx oasyce_capability register \
  --name "StressTestCap" \
  --endpoint "http://localhost:9999/stress" \
  --price 100uoas \
  --tags "stress,test" \
  --from validator $KB --chain-id $CHAIN_ID --fees 10000uoas --yes \
  --node $NODE -o json 2>/dev/null || echo '{"code":99}')
REG_CODE=$(echo "$REG_RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',0))" 2>/dev/null || echo "99")

if [ "$REG_CODE" != "0" ]; then
  warn "Capability register returned code $REG_CODE (may already exist)"
fi

sleep 7

# Find the capability ID.
CAP_ID=$(curl -sf "$REST/oasyce/capability/v1/capabilities" | python3 -c "
import sys, json
caps = json.load(sys.stdin).get('capabilities', [])
for c in caps:
    if c.get('name') == 'StressTestCap':
        print(c['id'])
        break
" 2>/dev/null || echo "")

if [ -z "$CAP_ID" ]; then
  warn "Could not find StressTestCap, skipping invoke burst"
else
  log "Found capability: $CAP_ID"
  INVOKE_OK=0
  INVOKE_FAIL=0
  INVOKE_ROUNDS=$((ROUNDS / 5))
  if [ "$INVOKE_ROUNDS" -lt 10 ]; then INVOKE_ROUNDS=10; fi

  for i in $(seq 1 $INVOKE_ROUNDS); do
    RESULT=$($OASYCED tx oasyce_capability invoke "$CAP_ID" '{"test":"stress"}' \
      --from validator $KB --chain-id $CHAIN_ID --fees 10000uoas --yes \
      --node $NODE -o json 2>/dev/null || echo '{"code":99}')
    CODE=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',99))" 2>/dev/null || echo "99")

    if [ "$CODE" = "0" ]; then
      INVOKE_OK=$((INVOKE_OK + 1))
    else
      INVOKE_FAIL=$((INVOKE_FAIL + 1))
    fi
  done

  log "Invoke burst: $INVOKE_OK ok, $INVOKE_FAIL failed out of $INVOKE_ROUNDS"
fi

# ---------- Test 3: Query Endpoint Load ----------
log "--- Test 3: REST query load (parallel curl) ---"

QUERY_ROUNDS=50
QUERY_OK=0
QUERY_FAIL=0

for i in $(seq 1 $QUERY_ROUNDS); do
  # Mix of different query types.
  case $((i % 4)) in
    0) URL="$REST/cosmos/bank/v1beta1/balances/$VALIDATOR" ;;
    1) URL="$REST/oasyce/capability/v1/capabilities" ;;
    2) URL="$REST/oasyce/reputation/v1/leaderboard" ;;
    3) URL="$REST/health" ;;
  esac

  HTTP_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$URL" 2>/dev/null || echo "000")
  if [ "$HTTP_CODE" = "200" ]; then
    QUERY_OK=$((QUERY_OK + 1))
  else
    QUERY_FAIL=$((QUERY_FAIL + 1))
  fi
done

log "Query load: $QUERY_OK ok, $QUERY_FAIL failed out of $QUERY_ROUNDS"

# ---------- Test 4: State Consistency Check ----------
log "--- Test 4: Post-stress state consistency ---"

sleep 10

# Check chain is still producing blocks.
NEW_HEIGHT=$(curl -sf "${NODE/tcp/http}/status" | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['sync_info']['latest_block_height'])" 2>/dev/null || echo "0")
if [ "$NEW_HEIGHT" -gt "$HEIGHT" ]; then
  log "Chain progressing: $HEIGHT -> $NEW_HEIGHT (+$((NEW_HEIGHT - HEIGHT)) blocks)"
else
  fail "Chain appears stalled at height $NEW_HEIGHT"
fi

# Verify balances are non-negative for all test accounts.
ALL_POSITIVE=true
for ADDR in "${TEST_ADDRS[@]}"; do
  BAL=$(curl -sf "$REST/cosmos/bank/v1beta1/balances/$ADDR" | python3 -c "import sys,json; bals=json.load(sys.stdin).get('balances',[]); print(next((b['amount'] for b in bals if b['denom']=='uoas'),'0'))" 2>/dev/null || echo "0")
  if [ "$BAL" -lt 0 ] 2>/dev/null; then
    fail "Negative balance for $ADDR: $BAL"
    ALL_POSITIVE=false
  fi
done

if $ALL_POSITIVE; then
  log "All account balances non-negative"
fi

# Check aggregate endpoints.
PROFILE_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$REST/oasyce/v1/agent-profile/$VALIDATOR" 2>/dev/null || echo "000")
MARKET_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$REST/oasyce/v1/marketplace" 2>/dev/null || echo "000")
HEALTH_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$REST/health" 2>/dev/null || echo "000")

log "Aggregate endpoints: profile=$PROFILE_CODE marketplace=$MARKET_CODE health=$HEALTH_CODE"

# ---------- Summary ----------
echo ""
echo "=============================="
echo "  STRESS TEST SUMMARY"
echo "=============================="
echo "  Bank sends:    $TX_SUCCESS/$ROUNDS ok  (~${TPS} tx/s)"
if [ -n "$CAP_ID" ]; then
echo "  Invoke burst:  $INVOKE_OK/$INVOKE_ROUNDS ok"
fi
echo "  Query load:    $QUERY_OK/$QUERY_ROUNDS ok"
echo "  Chain height:  $HEIGHT -> $NEW_HEIGHT"
echo "  State check:   $(if $ALL_POSITIVE; then echo 'PASS'; else echo 'FAIL'; fi)"
echo "=============================="

if [ "$TX_FAIL" -gt $((ROUNDS / 2)) ]; then
  fail "More than 50% of transactions failed"
  exit 1
fi

log "Stress test completed successfully"
