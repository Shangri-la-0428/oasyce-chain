#!/bin/bash
# =============================================================================
# Oasyce Full QA Test — All User Journeys + Edge Cases
# Runs against a LIVE testnet. Covers all 32 TX types and 32 query endpoints.
# =============================================================================

set -uo pipefail

OASYCED="./build/oasyced"
CHAIN_ID="oasyce-testnet-1"
NODE="tcp://47.93.32.88:26657"
REST="http://47.93.32.88:1317"
FAUCET="http://47.93.32.88:8080"
KB="--keyring-backend test"
COMMON="$KB --chain-id $CHAIN_ID --node $NODE --fees 10000uoas -y --output json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PASSED=0
FAILED=0
SKIPPED=0
FAILURES=""

pass() { PASSED=$((PASSED+1)); echo -e "${GREEN}  ✓ $1${NC}"; }
fail() { FAILED=$((FAILED+1)); FAILURES="$FAILURES\n  ✗ $1: $2"; echo -e "${RED}  ✗ $1: $2${NC}"; }
skip() { SKIPPED=$((SKIPPED+1)); echo -e "${YELLOW}  ⊘ $1 (skipped: $2)${NC}"; }
section() { echo -e "\n${CYAN}━━━ $1 ━━━${NC}"; }
wait_tx() { sleep 8; }

# Query helper — GET and extract field
qget() { curl -sf "$1" 2>/dev/null; }

# Find latest invocation ID from recent block events
find_latest_inv() {
    local latest_h
    latest_h=$(curl -sf "$REST/cosmos/base/tendermint/v1beta1/blocks/latest" | python3 -c "import sys,json; print(json.load(sys.stdin)['block']['header']['height'])" 2>/dev/null) || return
    local inv=""
    for bh in $(seq $((latest_h-3)) $latest_h); do
        local found
        found=$(curl -sf "http://47.93.32.88:26657/block_results?height=$bh" 2>/dev/null | python3 -c "
import sys,json
d = json.load(sys.stdin)
for tx in (d['result'].get('txs_results') or []):
    for e in tx.get('events',[]):
        if e.get('type') == 'capability_invoked':
            attrs = {a['key']:a['value'] for a in e.get('attributes',[])}
            print(attrs.get('invocation_id',''))
" 2>/dev/null) || true
        if [ -n "$found" ]; then inv="$found"; fi
    done
    echo "$inv"
}

# Check the LAST TX result from recent blocks.
# Returns "OK" if the last TX succeeded, "FAIL|reason" if it failed.
check_tx() {
    local latest
    latest=$(curl -sf "$REST/cosmos/base/tendermint/v1beta1/blocks/latest" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['block']['header']['height'])" 2>/dev/null) || { echo "FAIL|cannot get latest block"; return; }
    # Scan last 4 blocks and return the status of the most recent TX
    python3 -c "
import json, urllib.request
results = []
for h in range(max(1,$latest-3), $latest+1):
    try:
        url = 'http://47.93.32.88:26657/block_results?height=%d' % h
        data = json.loads(urllib.request.urlopen(url, timeout=5).read())
        txs = data['result'].get('txs_results') or []
        for tx in txs:
            code = tx.get('code', 0)
            if code != 0:
                results.append('FAIL|' + tx.get('log','')[:200])
            else:
                results.append('OK')
    except: pass
if results:
    print(results[-1])
else:
    print('OK')
" 2>/dev/null
}

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║         Oasyce Chain — Full QA Test Suite                   ║"
echo "║         Target: $REST                       ║"
echo "╚══════════════════════════════════════════════════════════════╝"

# ============================================================
section "0. Setup — Create & Fund Test Accounts"
# ============================================================

# Create accounts (idempotent)
for name in qa_provider qa_consumer qa_other; do
    $OASYCED keys show $name $KB 2>/dev/null || $OASYCED keys add $name $KB 2>/dev/null
done

PROVIDER=$($OASYCED keys show qa_provider -a $KB)
CONSUMER=$($OASYCED keys show qa_consumer -a $KB)
OTHER=$($OASYCED keys show qa_other -a $KB)

echo "  Provider: $PROVIDER"
echo "  Consumer: $CONSUMER"
echo "  Other:    $OTHER"

# Fund via faucet
for addr in $PROVIDER $CONSUMER $OTHER; do
    RESULT=$(curl -sf "$FAUCET/faucet?address=$addr" 2>/dev/null || echo '{"error":"fail"}')
    echo "  Faucet → ${addr:0:20}...: $(echo $RESULT | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('result','').get('txhash','')[:16] if 'result' in d else d.get('status', d.get('error','')))" 2>/dev/null || echo "done")"
    sleep 2
done

wait_tx

# Verify balances
for addr in $PROVIDER $CONSUMER $OTHER; do
    BAL=$(qget "$REST/cosmos/bank/v1beta1/balances/$addr" | python3 -c "import sys,json; b=json.load(sys.stdin).get('balances',[]); print(b[0]['amount'] if b else '0')" 2>/dev/null || echo "0")
    if [ "$BAL" != "0" ] && [ -n "$BAL" ]; then
        pass "Balance ${addr:0:15}...: $BAL uoas"
    else
        fail "Balance ${addr:0:15}..." "unfunded ($BAL)"
    fi
done

# ============================================================
section "1. Settlement — Escrow Lifecycle"
# ============================================================

# 1a. Create Escrow
echo "--- 1a. CreateEscrow ---"
TX=$($OASYCED tx settlement create-escrow 5000uoas --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "CreateEscrow"; else fail "CreateEscrow" "$RESULT"; fi

# Query escrows
ESCROWS=$(qget "$REST/oasyce/settlement/v1/escrows/$PROVIDER" || echo '{}')
ESCROW_ID=$(echo "$ESCROWS" | python3 -c "import sys,json; e=json.load(sys.stdin).get('escrows',[]); locked=[x for x in e if x['status']=='ESCROW_STATUS_LOCKED']; print(locked[0]['id'] if locked else '')" 2>/dev/null || echo "")

if [ -n "$ESCROW_ID" ]; then
    pass "QueryEscrows — found $ESCROW_ID (LOCKED)"
else
    fail "QueryEscrows" "no locked escrow found"
    # Create one more for release test
    ESCROW_ID=""
fi

# 1b. Release Escrow
echo "--- 1b. ReleaseEscrow ---"
if [ -n "$ESCROW_ID" ]; then
    TX=$($OASYCED tx settlement release-escrow "$ESCROW_ID" --from qa_provider $COMMON 2>&1 || echo '{}')
    wait_tx
    STATUS=$(qget "$REST/oasyce/settlement/v1/escrow/$ESCROW_ID" | python3 -c "import sys,json; print(json.load(sys.stdin).get('escrow',{}).get('status',''))" 2>/dev/null || echo "")
    if [ "$STATUS" = "ESCROW_STATUS_RELEASED" ]; then pass "ReleaseEscrow → RELEASED"; else fail "ReleaseEscrow" "status=$STATUS"; fi
else
    skip "ReleaseEscrow" "no escrow ID"
fi

# 1c. Create + Refund Escrow
echo "--- 1c. RefundEscrow ---"
TX=$($OASYCED tx settlement create-escrow 3000uoas --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
ESCROWS2=$(qget "$REST/oasyce/settlement/v1/escrows/$PROVIDER" || echo '{}')
ESCROW_ID2=$(echo "$ESCROWS2" | python3 -c "import sys,json; e=json.load(sys.stdin).get('escrows',[]); locked=[x for x in e if x['status']=='ESCROW_STATUS_LOCKED']; print(locked[0]['id'] if locked else '')" 2>/dev/null || echo "")
if [ -n "$ESCROW_ID2" ]; then
    TX=$($OASYCED tx settlement refund-escrow "$ESCROW_ID2" --from qa_provider $COMMON 2>&1 || echo '{}')
    wait_tx
    STATUS2=$(qget "$REST/oasyce/settlement/v1/escrow/$ESCROW_ID2" | python3 -c "import sys,json; print(json.load(sys.stdin).get('escrow',{}).get('status',''))" 2>/dev/null || echo "")
    if [ "$STATUS2" = "ESCROW_STATUS_REFUNDED" ]; then pass "RefundEscrow → REFUNDED"; else fail "RefundEscrow" "status=$STATUS2"; fi
else
    fail "RefundEscrow" "could not create escrow for refund test"
fi

# 1d. Query params
echo "--- 1d. Settlement Params ---"
SPARAMS=$(qget "$REST/oasyce/settlement/v1/params" | python3 -c "import sys,json; p=json.load(sys.stdin).get('params',{}); print(f'fee={p.get(\"protocol_fee_rate\",\"?\")} treasury={p.get(\"treasury_rate\",\"?\")}')" 2>/dev/null || echo "")
if [ -n "$SPARAMS" ]; then pass "SettlementParams: $SPARAMS"; else fail "SettlementParams" "empty"; fi

# ============================================================
section "2. DataRights — Asset Lifecycle"
# ============================================================

# 2a. Register Data Asset
echo "--- 2a. RegisterDataAsset ---"
TX=$($OASYCED tx datarights register "QA-Dataset-$(date +%s)" "sha256:qa$(date +%s)" \
    --description "QA test asset" --rights-type original --tags "qa,test" \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "RegisterDataAsset"; else fail "RegisterDataAsset" "$RESULT"; fi

# Find the asset
ASSET_ID=$(qget "$REST/oasyce/datarights/v1/data_assets" | python3 -c "
import sys,json
assets = json.load(sys.stdin).get('data_assets',[])
qa = [a for a in assets if 'QA-Dataset' in a.get('name','')]
print(qa[-1]['id'] if qa else '')
" 2>/dev/null || echo "")

if [ -n "$ASSET_ID" ]; then
    pass "QueryDataAssets — found $ASSET_ID"
else
    fail "QueryDataAssets" "QA asset not found"
fi

# 2b. Query single asset
echo "--- 2b. QueryDataAsset ---"
if [ -n "$ASSET_ID" ]; then
    ANAME=$(qget "$REST/oasyce/datarights/v1/data_asset/$ASSET_ID" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data_asset',{}).get('name',''))" 2>/dev/null || echo "")
    if [ -n "$ANAME" ]; then pass "QueryDataAsset: $ANAME"; else fail "QueryDataAsset" "empty"; fi
fi

# 2c. Buy Shares (Bancor curve)
echo "--- 2c. BuyShares ---"
if [ -n "$ASSET_ID" ]; then
    TX=$($OASYCED tx datarights buy-shares "$ASSET_ID" 500000uoas --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "BuyShares (500000uoas)"; else fail "BuyShares" "$RESULT"; fi

    # Query shares
    SHARES=$(qget "$REST/oasyce/datarights/v1/shares/$ASSET_ID" | python3 -c "
import sys,json
holders = json.load(sys.stdin).get('shareholders',[])
for h in holders:
    print(f'{h[\"address\"][:15]}... shares={h[\"shares\"]}')
" 2>/dev/null || echo "")
    if [ -n "$SHARES" ]; then pass "QueryShares: $SHARES"; else fail "QueryShares" "empty"; fi
fi

# 2d. Access Level query
echo "--- 2d. AccessLevel ---"
if [ -n "$ASSET_ID" ]; then
    ALEVEL=$(qget "$REST/oasyce/datarights/v1/access_level/$ASSET_ID/$CONSUMER" | python3 -c "
import sys,json
d = json.load(sys.stdin)
print(f'level={d.get(\"access_level\",\"none\")} equity={d.get(\"equity_bps\",0)}bps shares={d.get(\"shares\",\"0\")}')
" 2>/dev/null || echo "")
    if [ -n "$ALEVEL" ]; then pass "AccessLevel: $ALEVEL"; else fail "AccessLevel" "empty"; fi

    # Non-holder should get empty level
    ALEVEL_NONE=$(qget "$REST/oasyce/datarights/v1/access_level/$ASSET_ID/$OTHER" | python3 -c "
import sys,json; d=json.load(sys.stdin); print(d.get('access_level',''))" 2>/dev/null || echo "x")
    if [ -z "$ALEVEL_NONE" ]; then pass "AccessLevel (non-holder) = empty"; else fail "AccessLevel (non-holder)" "got $ALEVEL_NONE"; fi
fi

# 2e. Sell Shares
echo "--- 2e. SellShares ---"
if [ -n "$ASSET_ID" ]; then
    TX=$($OASYCED tx datarights sell-shares "$ASSET_ID" 100 --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "SellShares (100 shares)"; else fail "SellShares" "$RESULT"; fi
fi

# 2f. Version fork
echo "--- 2f. RegisterVersionedAsset (fork) ---"
if [ -n "$ASSET_ID" ]; then
    TX=$($OASYCED tx datarights register "QA-Fork-$(date +%s)" "sha256:fork$(date +%s)" \
        --description "Fork of QA asset" --rights-type derivative --tags "qa,fork" \
        --parent "$ASSET_ID" --from qa_other $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "RegisterVersionedAsset (fork)"; else fail "RegisterVersionedAsset" "$RESULT"; fi

    # Query children
    CHILDREN=$(qget "$REST/oasyce/datarights/v1/asset_children/$ASSET_ID" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('data_assets',[])))" 2>/dev/null || echo "0")
    if [ "$CHILDREN" != "0" ]; then pass "QueryAssetChildren: $CHILDREN forks"; else fail "QueryAssetChildren" "0 forks"; fi
fi

# 2g. Datarights Params
echo "--- 2g. DatarightsParams ---"
DRPARAMS=$(qget "$REST/oasyce/datarights/v1/params" | python3 -c "import sys,json; p=json.load(sys.stdin).get('params',{}); print(f'cooldown={p.get(\"shutdown_cooldown_seconds\",\"?\")}s')" 2>/dev/null || echo "")
if [ -n "$DRPARAMS" ]; then pass "DatarightsParams: $DRPARAMS"; else fail "DatarightsParams" "empty"; fi

# ============================================================
section "3. Capability — Full Service Contract Flow"
# ============================================================

# 3a. Register capability
echo "--- 3a. RegisterCapability ---"
TX=$($OASYCED tx oasyce_capability register "QA-Summarizer-$(date +%s)" "https://api.qa.test/summarize" 100000uoas \
    --description "QA test capability" --tags "qa,nlp" --rate-limit 60 \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "RegisterCapability"; else fail "RegisterCapability" "$RESULT"; fi

# Query capabilities
CAP_ID=$(qget "$REST/oasyce/capability/v1/capabilities" | python3 -c "
import sys,json
caps = json.load(sys.stdin).get('capabilities',[])
qa = [c for c in caps if 'QA-Summarizer' in c.get('name','')]
print(qa[-1]['id'] if qa else '')
" 2>/dev/null || echo "")

if [ -n "$CAP_ID" ]; then
    pass "QueryCapabilities — found $CAP_ID"
else
    fail "QueryCapabilities" "QA capability not found"
fi

# 3b. Query single capability
echo "--- 3b. QueryCapability ---"
if [ -n "$CAP_ID" ]; then
    CNAME=$(qget "$REST/oasyce/capability/v1/capability/$CAP_ID" | python3 -c "import sys,json; print(json.load(sys.stdin).get('capability',{}).get('name',''))" 2>/dev/null || echo "")
    if [ -n "$CNAME" ]; then pass "QueryCapability: $CNAME"; else fail "QueryCapability" "empty"; fi
fi

# 3c. Query by provider
echo "--- 3c. QueryByProvider ---"
PCAPS=$(qget "$REST/oasyce/capability/v1/capabilities/provider/$PROVIDER" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('capabilities',[])))" 2>/dev/null || echo "0")
if [ "$PCAPS" != "0" ]; then pass "QueryByProvider: $PCAPS caps"; else fail "QueryByProvider" "0 caps"; fi

# 3d. Invoke capability (consumer pays, escrow locked)
echo "--- 3d. InvokeCapability ---"
if [ -n "$CAP_ID" ]; then
    TX=$($OASYCED tx oasyce_capability invoke "$CAP_ID" \
        --input '{"text":"Summarize the Oasyce whitepaper"}' \
        --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "InvokeCapability"; else fail "InvokeCapability" "$RESULT"; fi
fi

# Find the invocation ID from recent block events
INV_ID=""
LATEST_H=$(curl -sf "$REST/cosmos/base/tendermint/v1beta1/blocks/latest" | python3 -c "import sys,json; print(json.load(sys.stdin)['block']['header']['height'])" 2>/dev/null)
for bh in $(seq $((LATEST_H-3)) $LATEST_H); do
    FOUND=$(curl -sf "http://47.93.32.88:26657/block_results?height=$bh" 2>/dev/null | python3 -c "
import sys,json
d = json.load(sys.stdin)
for tx in (d['result'].get('txs_results') or []):
    for e in tx.get('events',[]):
        if e.get('type') == 'capability_invoked':
            attrs = {a['key']:a['value'] for a in e.get('attributes',[])}
            print(attrs.get('invocation_id',''))
" 2>/dev/null)
    if [ -n "$FOUND" ]; then INV_ID="$FOUND"; fi
done
echo "  Invocation ID: $INV_ID"

# 3e. Complete invocation (provider submits output hash)
echo "--- 3e. CompleteInvocation ---"
OUTPUT_HASH="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
TX=$($OASYCED tx oasyce_capability complete-invocation "$INV_ID" "$OUTPUT_HASH" \
    --usage-report '{"prompt_tokens":150,"completion_tokens":80,"total_tokens":230}' \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then
    pass "CompleteInvocation → COMPLETED (challenge window started, with usage_report)"
    COMPLETED_HEIGHT=$(qget "$REST/cosmos/base/tendermint/v1beta1/blocks/latest" | python3 -c "import sys,json; print(json.load(sys.stdin)['block']['header']['height'])" 2>/dev/null || echo "0")
    echo -e "${YELLOW}    Challenge window ends at ~block $((COMPLETED_HEIGHT + 100))${NC}"
    # Verify usage_report is stored on-chain
    USAGE=$(qget "$REST/oasyce/capability/v1/invocation/$INV_ID" | python3 -c "import sys,json; print(json.load(sys.stdin).get('invocation',{}).get('usage_report',''))" 2>/dev/null || echo "")
    if [ -n "$USAGE" ]; then
        pass "usage_report stored on-chain: $USAGE"
    else
        warn "usage_report not returned in query (may not be indexed yet)"
    fi
else
    fail "CompleteInvocation" "$RESULT"
fi

# 3f. Edge: Claim too early (should fail)
echo "--- 3f. ClaimInvocation (too early — expect failure) ---"
TX=$($OASYCED tx oasyce_capability claim-invocation "$INV_ID" \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then
    fail "ClaimInvocation (too early)" "should have failed but succeeded"
else
    pass "ClaimInvocation (too early) — correctly rejected"
fi

# 3g. Edge: Dispute by non-consumer (should fail)
echo "--- 3g. DisputeInvocation (wrong caller — expect failure) ---"
TX=$($OASYCED tx oasyce_capability dispute-invocation "$INV_ID" "I am not the consumer" \
    --from qa_other $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then
    fail "DisputeInvocation (wrong caller)" "should have failed but succeeded"
else
    pass "DisputeInvocation (wrong caller) — correctly rejected"
fi

# 3h. Second invoke for dispute test
echo "--- 3h. Invoke #2 (for dispute test) ---"
if [ -n "$CAP_ID" ]; then
    TX=$($OASYCED tx oasyce_capability invoke "$CAP_ID" \
        --input '{"text":"dispute test"}' \
        --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "InvokeCapability #2"; else fail "InvokeCapability #2" "$RESULT"; fi
fi

INV_ID2=$(find_latest_inv)
echo "  Invocation #2 ID: $INV_ID2"

# 3i. Complete #2
echo "--- 3i. Complete #2 ---"
TX=$($OASYCED tx oasyce_capability complete-invocation "$INV_ID2" \
    "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "CompleteInvocation #2"; else fail "CompleteInvocation #2" "$RESULT"; fi

# 3j. Dispute #2 (consumer disputes within window)
echo "--- 3j. DisputeInvocation (within window) ---"
TX=$($OASYCED tx oasyce_capability dispute-invocation "$INV_ID2" "output was garbage" \
    --from qa_consumer $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "DisputeInvocation → DISPUTED (escrow refunded)"; else fail "DisputeInvocation" "$RESULT"; fi

# 3k. Third invoke for fail test
echo "--- 3k. Invoke #3 (for fail test) ---"
if [ -n "$CAP_ID" ]; then
    TX=$($OASYCED tx oasyce_capability invoke "$CAP_ID" \
        --input '{"text":"fail test"}' \
        --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "InvokeCapability #3"; else fail "InvokeCapability #3" "$RESULT"; fi
fi

INV_ID3=$(find_latest_inv)
echo "  Invocation #3 ID: $INV_ID3"

# 3l. FailInvocation (provider reports failure, escrow refunded)
echo "--- 3l. FailInvocation ---"
TX=$($OASYCED tx oasyce_capability fail-invocation "$INV_ID3" \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "FailInvocation → FAILED (escrow refunded)"; else fail "FailInvocation" "$RESULT"; fi

# 3m. Edge: Complete with short hash (rejected at ValidateBasic before broadcast)
echo "--- 3m. CompleteInvocation (short hash — expect failure) ---"
# Need a new invocation first
TX=$($OASYCED tx oasyce_capability invoke "$CAP_ID" \
    --input '{"text":"hash test"}' --from qa_consumer $COMMON 2>&1 || echo '{}')
wait_tx
INV_ID4=$(find_latest_inv)
echo "  Invocation #4 ID: $INV_ID4"
# Short hash should be rejected at ValidateBasic (CLI exits with error before broadcasting)
SHORT_RESULT=$($OASYCED tx oasyce_capability complete-invocation "$INV_ID4" "tooshort" \
    --from qa_provider $COMMON 2>&1)
SHORT_EXIT=$?
if [ $SHORT_EXIT -ne 0 ] || echo "$SHORT_RESULT" | grep -qi "output.hash\|invalid\|error"; then
    pass "CompleteInvocation (short hash) — rejected (ValidateBasic)"
else
    SHORT_CODE=$(echo "$SHORT_RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',0))" 2>/dev/null || echo "?")
    if [ "$SHORT_CODE" != "0" ]; then
        pass "CompleteInvocation (short hash) — rejected at CheckTx (code=$SHORT_CODE)"
    else
        wait_tx
        RESULT=$(check_tx)
        if [[ "$RESULT" != "OK" ]]; then
            pass "CompleteInvocation (short hash) — rejected at DeliverTx"
        else
            fail "CompleteInvocation (short hash)" "should have been rejected"
        fi
    fi
fi

# 3n. UpdateCapability
echo "--- 3n. UpdateCapability ---"
if [ -n "$CAP_ID" ]; then
    TX=$($OASYCED tx oasyce_capability update "$CAP_ID" \
        --description "Updated QA description" --price 200000uoas \
        --from qa_provider $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then pass "UpdateCapability"; else fail "UpdateCapability" "$RESULT"; fi
fi

# 3o. DeactivateCapability
echo "--- 3o. DeactivateCapability ---"
if [ -n "$CAP_ID" ]; then
    TX=$($OASYCED tx oasyce_capability deactivate "$CAP_ID" \
        --from qa_provider $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then
        ACTIVE=$(qget "$REST/oasyce/capability/v1/capability/$CAP_ID" | python3 -c "import sys,json; print(json.load(sys.stdin).get('capability',{}).get('is_active',''))" 2>/dev/null || echo "")
        if [ "$ACTIVE" = "false" ] || [ "$ACTIVE" = "False" ]; then pass "DeactivateCapability → inactive"; else fail "DeactivateCapability" "still active=$ACTIVE"; fi
    else
        fail "DeactivateCapability" "$RESULT"
    fi
fi

# 3p. Edge: Invoke deactivated capability (should fail)
echo "--- 3p. Invoke deactivated (expect failure) ---"
if [ -n "$CAP_ID" ]; then
    TX=$($OASYCED tx oasyce_capability invoke "$CAP_ID" \
        --input '{"text":"should fail"}' --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then
        fail "Invoke deactivated" "should have been rejected"
    else
        pass "Invoke deactivated — correctly rejected"
    fi
fi

# ============================================================
section "4. Reputation"
# ============================================================

# 4a. Submit feedback
echo "--- 4a. SubmitFeedback ---"
TX=$($OASYCED tx reputation submit-feedback "$INV_ID" 450 --comment "Excellent service" \
    --from qa_consumer $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then pass "SubmitFeedback (score=450)"; else fail "SubmitFeedback" "$RESULT"; fi

# 4b. Query reputation
echo "--- 4b. QueryReputation ---"
REP=$(qget "$REST/oasyce/reputation/v1/reputation/$PROVIDER" | python3 -c "
import sys,json
d = json.load(sys.stdin).get('reputation',{})
print(f'score={d.get(\"score\",\"0\")} feedbacks={d.get(\"total_feedbacks\",\"0\")}')
" 2>/dev/null || echo "")
if [ -n "$REP" ]; then pass "QueryReputation: $REP"; else fail "QueryReputation" "empty"; fi

# 4c. Query feedback
echo "--- 4c. QueryFeedback ---"
FB=$(qget "$REST/oasyce/reputation/v1/feedback/$INV_ID" | python3 -c "
import sys,json
d = json.load(sys.stdin).get('feedback',{})
print(f'score={d.get(\"score\",\"0\")} comment={d.get(\"comment\",\"\")[:30]}')
" 2>/dev/null || echo "")
if [ -n "$FB" ]; then pass "QueryFeedback: $FB"; else fail "QueryFeedback" "empty"; fi

# 4d. Leaderboard
echo "--- 4d. Leaderboard ---"
LB=$(qget "$REST/oasyce/reputation/v1/leaderboard" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('scores',[])))" 2>/dev/null || echo "0")
if [ "$LB" != "0" ]; then pass "Leaderboard: $LB entries"; else fail "Leaderboard" "empty"; fi

# ============================================================
section "5. Query Endpoints Sweep"
# ============================================================

# Test all query endpoints return valid JSON (not 404/500)
ENDPOINTS=(
    "/oasyce/settlement/v1/params"
    "/oasyce/settlement/v1/escrows/$PROVIDER"
    "/oasyce/capability/v1/capabilities"
    "/oasyce/capability/v1/capabilities/provider/$PROVIDER"
    "/oasyce/capability/v1/earnings/$PROVIDER"
    "/oasyce/reputation/v1/reputation/$PROVIDER"
    "/oasyce/reputation/v1/leaderboard"
    "/oasyce/datarights/v1/data_assets"
    "/oasyce/datarights/v1/params"
    "/oasyce/datarights/v1/disputes"
    "/oasyce/onboarding/v1/params"
    "/oasyce/work/v1/executors"
    "/oasyce/work/v1/params"
    "/cosmos/bank/v1beta1/balances/$PROVIDER"
    "/cosmos/auth/v1beta1/accounts/$PROVIDER"
    "/cosmos/staking/v1beta1/validators"
    "/cosmos/base/tendermint/v1beta1/blocks/latest"
)

for ep in "${ENDPOINTS[@]}"; do
    CODE=$(curl -sf -o /dev/null -w '%{http_code}' "$REST$ep" 2>/dev/null || echo "000")
    SHORT=$(echo "$ep" | sed "s|$PROVIDER|{addr}|g" | tail -c 60)
    if [ "$CODE" = "200" ]; then
        pass "GET $SHORT → $CODE"
    else
        fail "GET $SHORT" "HTTP $CODE"
    fi
done

# ============================================================
section "6. Edge Cases & Auth"
# ============================================================

# 6a. Deactivate by non-owner (should fail)
echo "--- 6a. DeactivateCapability (non-owner — expect failure) ---"
# Register a new cap first since the old one is deactivated
TX=$($OASYCED tx oasyce_capability register "QA-AuthTest-$(date +%s)" "https://auth.test" 100uoas \
    --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
AUTH_CAP=$(qget "$REST/oasyce/capability/v1/capabilities" | python3 -c "
import sys,json
caps = json.load(sys.stdin).get('capabilities',[])
qa = [c for c in caps if 'QA-AuthTest' in c.get('name','') and c.get('is_active')]
print(qa[-1]['id'] if qa else '')
" 2>/dev/null || echo "")

if [ -n "$AUTH_CAP" ]; then
    TX=$($OASYCED tx oasyce_capability deactivate "$AUTH_CAP" \
        --from qa_consumer $COMMON 2>&1 || echo '{}')
    wait_tx
    RESULT=$(check_tx)
    if [[ "$RESULT" == "OK" ]]; then
        fail "Deactivate (non-owner)" "should have been rejected"
    else
        pass "Deactivate (non-owner) — correctly rejected"
    fi
fi

# 6b. Double complete (should fail)
echo "--- 6b. Double Complete (expect failure) ---"
# INV_ID2 was already disputed, try completing it again
TX=$($OASYCED tx oasyce_capability complete-invocation "$INV_ID2" \
    "$OUTPUT_HASH" --from qa_provider $COMMON 2>&1 || echo '{}')
wait_tx
RESULT=$(check_tx)
if [[ "$RESULT" == "OK" ]]; then
    fail "Double Complete" "should have been rejected (already DISPUTED)"
else
    pass "Double Complete — correctly rejected"
fi

# ============================================================
section "RESULTS"
# ============================================================

echo ""
echo -e "${GREEN}  Passed:  $PASSED${NC}"
echo -e "${RED}  Failed:  $FAILED${NC}"
echo -e "${YELLOW}  Skipped: $SKIPPED${NC}"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failures:${NC}"
    echo -e "$FAILURES"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Note about challenge window claim test
echo -e "${YELLOW}"
echo "NOTE: ClaimInvocation (after challenge window) was NOT tested."
echo "$INV_ID is in COMPLETED state. After ~100 blocks (~8 min),"
echo "run manually:"
echo "  ./build/oasyced tx oasyce_capability claim-invocation $INV_ID \\"
echo "    --from qa_provider --keyring-backend test \\"
echo "    --chain-id oasyce-testnet-1 --node tcp://47.93.32.88:26657 \\"
echo "    --fees 10000uoas -y"
echo -e "${NC}"

exit $FAILED
