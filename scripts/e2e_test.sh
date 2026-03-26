#!/bin/bash
# End-to-end test script for oasyce-chain
# Requires: chain running locally, validator key in test keyring
# Usage: ./scripts/e2e_test.sh

set -e

OASYCED="${OASYCED:-./build/oasyced}"
CHAIN_ID="oasyce-local-1"
NODE="tcp://localhost:26657"
KB="--keyring-backend test"
FEES="--fees 10000uoas"
COMMON="$KB --chain-id $CHAIN_ID $FEES --yes"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass() { echo -e "${GREEN}  ✓ $1${NC}"; }
fail() { echo -e "${RED}  ✗ $1: $2${NC}"; }

wait_tx() { sleep 7; }

check_latest_tx() {
    local latest=$(curl -s http://localhost:26657/block | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['block']['header']['height'])")
    for h in $(seq $((latest-2)) $latest); do
        local result=$(curl -s "http://localhost:26657/block_results?height=$h" 2>/dev/null)
        echo "$result" | python3 -c "
import sys,json
try:
    d = json.load(sys.stdin)
    txs = d['result'].get('txs_results') or []
    for tx in txs:
        code = tx.get('code', 0)
        log = tx.get('log', '')
        print(f'{code}|{log[:200]}')
except: pass
" 2>/dev/null
    done | tail -1
}

ADDR=$($OASYCED keys show validator -a $KB 2>/dev/null)
echo "====== Oasyce Chain E2E Tests ======"
echo "Validator: $ADDR"
echo ""

# Ensure user1 exists
$OASYCED keys show user1 $KB 2>/dev/null || $OASYCED keys add user1 $KB 2>/dev/null
USER1=$($OASYCED keys show user1 -a $KB 2>/dev/null)
echo "User1: $USER1"
echo ""

# Fund user1 if needed
USER1_BAL=$(curl -s "http://localhost:1317/cosmos/bank/v1beta1/balances/$USER1" | python3 -c "import sys,json; d=json.load(sys.stdin); b=d.get('balances',[]); print(b[0]['amount'] if b else '0')" 2>/dev/null)
if [ "$USER1_BAL" = "0" ]; then
    echo "Funding user1..."
    $OASYCED tx send "$ADDR" "$USER1" 100000000uoas --from validator $COMMON 2>/dev/null
    wait_tx
fi

# --- Test 1: Register Data Asset ---
echo "--- Test 1: Register Data Asset ---"
$OASYCED tx datarights register "e2e-test-asset" "sha256:e2e123" \
    --description "E2E test" --rights-type original --tags "e2e,test" \
    --from validator $COMMON 2>/dev/null
wait_tx
ASSET_ID=$($OASYCED query datarights list --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); assets=[a for a in d.get('data_assets',[]) if a['name']=='e2e-test-asset']; print(assets[0]['id'] if assets else '')" 2>/dev/null)
if [ -n "$ASSET_ID" ]; then pass "RegisterDataAsset ($ASSET_ID)"; else fail "RegisterDataAsset" "not found"; fi

# --- Test 2: Buy Shares ---
echo "--- Test 2: Buy Shares ---"
$OASYCED tx datarights buy-shares "$ASSET_ID" 1000uoas --from user1 $COMMON 2>/dev/null
wait_tx
SHARES=$($OASYCED query datarights asset "$ASSET_ID" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data_asset',{}).get('total_shares','0'))" 2>/dev/null)
if [ "$SHARES" != "0" ]; then pass "BuyShares (shares=$SHARES)"; else fail "BuyShares" "shares=0"; fi

# --- Test 3: Create Escrow ---
echo "--- Test 3: Create Escrow ---"
$OASYCED tx settlement create-escrow 5000uoas --asset-id "$ASSET_ID" --from validator $COMMON 2>/dev/null
wait_tx
ESCROW_ID=$($OASYCED query settlement escrows "$ADDR" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); e=[x for x in d.get('escrows',[]) if x['status']=='ESCROW_STATUS_LOCKED']; print(e[0]['id'] if e else '')" 2>/dev/null)
if [ -n "$ESCROW_ID" ]; then pass "CreateEscrow ($ESCROW_ID)"; else fail "CreateEscrow" "not found"; fi

# --- Test 4: Release Escrow ---
echo "--- Test 4: Release Escrow ---"
$OASYCED tx settlement release-escrow "$ESCROW_ID" --from validator $COMMON 2>/dev/null
wait_tx
STATUS=$($OASYCED query settlement escrow "$ESCROW_ID" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('escrow',{}).get('status',''))" 2>/dev/null)
if [ "$STATUS" = "ESCROW_STATUS_RELEASED" ]; then pass "ReleaseEscrow"; else fail "ReleaseEscrow" "status=$STATUS"; fi

# --- Test 5: Register Capability ---
echo "--- Test 5: Register Capability ---"
$OASYCED tx oasyce_capability register "E2E-API" "https://api.example.com/e2e" 500uoas \
    --description "E2E test capability" --tags "e2e" \
    --from validator $COMMON 2>/dev/null
wait_tx
CAP_ID=$($OASYCED query oasyce_capability list --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); caps=[c for c in d.get('capabilities',[]) if c['name']=='E2E-API']; print(caps[0]['id'] if caps else '')" 2>/dev/null)
if [ -n "$CAP_ID" ]; then pass "RegisterCapability ($CAP_ID)"; else fail "RegisterCapability" "not found"; fi

# --- Test 6: Invoke Capability (from user1) ---
echo "--- Test 6: Invoke Capability ---"
$OASYCED tx oasyce_capability invoke "$CAP_ID" --input '{"test":true}' --from user1 $COMMON 2>/dev/null
wait_tx
RESULT=$(check_latest_tx)
CODE=$(echo "$RESULT" | cut -d'|' -f1)
if [ "$CODE" = "0" ]; then pass "InvokeCapability"; else fail "InvokeCapability" "$RESULT"; fi

# Find invocation ID dynamically (query latest invocation for this capability)
INV_ID=$($OASYCED query oasyce_capability get "$CAP_ID" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); invs=d.get('capability',{}).get('invocations',[]); print(invs[-1] if invs else '')" 2>/dev/null)
if [ -z "$INV_ID" ]; then
  # Fallback: scan invocations by querying all capabilities
  INV_ID=$($OASYCED query oasyce_capability invocation "INV_0000000000000001" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('invocation',{}).get('id',''))" 2>/dev/null)
fi
if [ -z "$INV_ID" ]; then INV_ID="INV_0000000000000001"; fi  # last resort fallback

# --- Test 6b: Complete Invocation (provider submits output hash) ---
echo "--- Test 6b: Complete Invocation (challenge window starts) ---"
OUTPUT_HASH="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
$OASYCED tx oasyce_capability complete-invocation "$INV_ID" "$OUTPUT_HASH" --from validator $COMMON 2>/dev/null
wait_tx
RESULT=$(check_latest_tx)
CODE=$(echo "$RESULT" | cut -d'|' -f1)
if [ "$CODE" = "0" ]; then pass "CompleteInvocation (challenge window started)"; else fail "CompleteInvocation" "$RESULT"; fi

# --- Test 6c: Claim Invocation (provider claims after challenge window) ---
# NOTE: On a live chain, must wait 100 blocks (~8 min). For e2e testing this will fail
# unless the chain has advanced 100+ blocks. We test it anyway to verify the TX format.
echo "--- Test 6c: Claim Invocation (will fail if <100 blocks since complete) ---"
$OASYCED tx oasyce_capability claim-invocation "$INV_ID" --from validator $COMMON 2>/dev/null
wait_tx
RESULT=$(check_latest_tx)
CODE=$(echo "$RESULT" | cut -d'|' -f1)
if [ "$CODE" = "0" ]; then pass "ClaimInvocation"; else fail "ClaimInvocation (expected — challenge window)" "$RESULT"; fi

# --- Test 7: Submit Feedback (from user1 about validator's capability) ---
echo "--- Test 7: Submit Feedback ---"
$OASYCED tx reputation submit-feedback "$INV_ID" 450 --comment "Great" --from user1 $COMMON 2>/dev/null
wait_tx
RESULT=$(check_latest_tx)
CODE=$(echo "$RESULT" | cut -d'|' -f1)
if [ "$CODE" = "0" ]; then pass "SubmitFeedback"; else fail "SubmitFeedback" "$RESULT"; fi

# --- Test 8: Reputation Leaderboard ---
echo "--- Test 8: Reputation Leaderboard ---"
SCORES=$($OASYCED query reputation leaderboard --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('scores',[])))" 2>/dev/null)
if [ "$SCORES" != "0" ]; then pass "Leaderboard ($SCORES entries)"; else fail "Leaderboard" "empty"; fi

# --- Test 9: Sell Shares ---
echo "--- Test 9: Sell Shares ---"
$OASYCED tx datarights sell-shares "$ASSET_ID" 500 --from user1 $COMMON 2>/dev/null
wait_tx
SHARES_AFTER=$($OASYCED query datarights asset "$ASSET_ID" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data_asset',{}).get('total_shares','0'))" 2>/dev/null)
if [ "$SHARES_AFTER" != "$SHARES" ]; then pass "SellShares (shares=$SHARES_AFTER)"; else fail "SellShares" "shares unchanged"; fi

# --- Test 10: Register Versioned Asset (fork) ---
echo "--- Test 10: Register Versioned Asset ---"
$OASYCED tx datarights register "e2e-fork-asset" "sha256:fork123" \
    --description "Fork of e2e-test-asset" --rights-type derivative --tags "e2e,fork" \
    --parent "$ASSET_ID" --from user1 $COMMON 2>/dev/null
wait_tx
FORK_ID=$($OASYCED query datarights list --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); assets=[a for a in d.get('data_assets',[]) if a['name']=='e2e-fork-asset']; print(assets[0]['id'] if assets else '')" 2>/dev/null)
if [ -n "$FORK_ID" ]; then pass "RegisterVersionedAsset ($FORK_ID)"; else fail "RegisterVersionedAsset" "not found"; fi

# --- Test 11: Query Asset Children ---
echo "--- Test 11: Query Asset Children ---"
CHILDREN=$($OASYCED query datarights children "$ASSET_ID" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('data_assets',[])))" 2>/dev/null)
if [ "$CHILDREN" != "0" ]; then pass "AssetChildren ($CHILDREN forks)"; else fail "AssetChildren" "empty"; fi

# --- Test 12: Onboarding — PoW Self-Register ---
echo "--- Test 12: PoW Self-Register ---"
$OASYCED keys show user2 $KB 2>/dev/null || $OASYCED keys add user2 $KB 2>/dev/null
USER2=$($OASYCED keys show user2 -a $KB 2>/dev/null)
# Fund user2 with minimal amount (creates account on chain, needed for tx signing)
$OASYCED tx send "$ADDR" "$USER2" 1uoas --from validator $COMMON 2>/dev/null
wait_tx
# Use built-in PoW solver to find valid nonce
NONCE=$($OASYCED util solve-pow "$USER2" --difficulty 16 --output json 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['nonce'])")
echo "  Solved PoW nonce: $NONCE"
$OASYCED tx onboarding register "$NONCE" --from user2 $KB --chain-id $CHAIN_ID --fees 0uoas --yes 2>/dev/null
wait_tx
RESULT=$(check_latest_tx)
CODE=$(echo "$RESULT" | cut -d'|' -f1)
if [ "$CODE" = "0" ]; then pass "SelfRegister"; else fail "SelfRegister" "$RESULT"; fi

# --- Test 13: Onboarding — Repay ---
echo "--- Test 13: Onboarding Repay ---"
# Repay partial debt (airdrop=20M uoas, keep some for fees)
$OASYCED tx onboarding repay 10000000 --from user2 $KB --chain-id $CHAIN_ID --fees 0uoas --yes 2>/dev/null
wait_tx
RESULT=$(check_latest_tx)
CODE=$(echo "$RESULT" | cut -d'|' -f1)
if [ "$CODE" = "0" ]; then pass "RepayDebt"; else fail "RepayDebt" "$RESULT"; fi

# --- Test 15: Initiate Shutdown ---
echo "--- Test 15: Initiate Shutdown ---"
$OASYCED tx datarights initiate-shutdown "$ASSET_ID" --from validator $COMMON 2>/dev/null
wait_tx
ASSET_STATUS=$($OASYCED query datarights asset "$ASSET_ID" --node $NODE --output json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data_asset',{}).get('status',''))" 2>/dev/null)
if [ "$ASSET_STATUS" = "ASSET_STATUS_SHUTTING_DOWN" ]; then pass "InitiateShutdown"; else fail "InitiateShutdown" "status=$ASSET_STATUS"; fi

echo ""
echo "====== E2E Complete ======"
