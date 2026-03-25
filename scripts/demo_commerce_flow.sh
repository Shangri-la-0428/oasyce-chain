#!/bin/bash
# =============================================================================
# Oasyce Commerce Flow Demo
# Demonstrates the full economic lifecycle: registration → asset → trading →
# service contract → settlement → reputation
# =============================================================================

set -e

# Colors
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

BINARY="${BINARY:-oasyced}"
CHAIN_ID="${CHAIN_ID:-oasyce-localnet-1}"
KEYRING="--keyring-backend test"
COMMON="--chain-id $CHAIN_ID $KEYRING --gas auto --gas-adjustment 1.5 --fees 10000uoas -y --output json"

section() { echo -e "\n${CYAN}═══════════════════════════════════════════════════════════════${NC}"; echo -e "${GREEN}  $1${NC}"; echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}\n"; }
info()    { echo -e "${YELLOW}→ $1${NC}"; }
wait_tx() { sleep 6; }

PROVIDER=$($BINARY keys show provider -a $KEYRING 2>/dev/null || echo "")
CONSUMER=$($BINARY keys show consumer -a $KEYRING 2>/dev/null || echo "")

if [ -z "$PROVIDER" ] || [ -z "$CONSUMER" ]; then
    echo "Error: 'provider' and 'consumer' keys must exist in the test keyring."
    echo "Create them with:"
    echo "  $BINARY keys add provider $KEYRING"
    echo "  $BINARY keys add consumer $KEYRING"
    echo "Then fund them via faucet or genesis."
    exit 1
fi

echo -e "${GREEN}Provider: $PROVIDER${NC}"
echo -e "${GREEN}Consumer: $CONSUMER${NC}"

# ═══════════════════════════════════════════════════════════════
# Step 1: Asset Securitization — Data is a financial instrument
# ═══════════════════════════════════════════════════════════════
section "STEP 1: Data Securitization"
info "Registering a data asset with bonding curve pricing."
info "Data is not a file — it's a financial instrument. Price rises with demand."

TX=$($BINARY tx datarights register \
    "NLP Training Set v2" \
    "100K labeled sentences for NLP fine-tuning" \
    "sha256:abc123def456789012345678901234567890abcdef" \
    --tags "nlp,training,english" \
    --from provider $COMMON 2>&1)
echo "$TX" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  TX Hash: {d.get(\"txhash\",\"N/A\")}')" 2>/dev/null || echo "  TX submitted"
wait_tx

info "Querying all data assets..."
$BINARY query datarights assets --output json 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
assets = data.get('data_assets', [])
if assets:
    a = assets[-1]
    print(f'  Asset ID:     {a[\"id\"]}')
    print(f'  Name:         {a[\"name\"]}')
    print(f'  Total Shares: {a[\"total_shares\"]}')
    print(f'  Reserve:      {a[\"reserve_balance\"]}')
    print(f'  Status:       {a.get(\"status\", \"ACTIVE\")}')
" 2>/dev/null || echo "  (query datarights assets)"

# Get the asset ID for next steps
ASSET_ID=$($BINARY query datarights assets --output json 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
assets = data.get('data_assets', [])
print(assets[-1]['id'] if assets else '')
" 2>/dev/null)

# ═══════════════════════════════════════════════════════════════
# Step 2: Property Trading — Bancor curve, price rises with demand
# ═══════════════════════════════════════════════════════════════
section "STEP 2: Property Trading (Bancor Bonding Curve)"
info "Consumer buys shares. Bancor curve: more buyers → higher price."
info "Holding ≥1% equity → L1 access (sample data). ≥10% → L3 (full delivery)."

if [ -n "$ASSET_ID" ]; then
    TX=$($BINARY tx datarights buy-shares $ASSET_ID 500000uoas \
        --from consumer $COMMON 2>&1)
    echo "$TX" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  TX Hash: {d.get(\"txhash\",\"N/A\")}')" 2>/dev/null || echo "  TX submitted"
    wait_tx

    info "Checking access level..."
    $BINARY query datarights shares $ASSET_ID --output json 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
holders = data.get('shareholders', [])
for h in holders:
    print(f'  Holder:  {h[\"address\"]}')
    print(f'  Shares:  {h[\"shares\"]}')
" 2>/dev/null || echo "  (query datarights shares)"
else
    info "Skipping — no asset ID found"
fi

# ═══════════════════════════════════════════════════════════════
# Step 3: Service Contract — Register AI capability
# ═══════════════════════════════════════════════════════════════
section "STEP 3: Service Contract — Register Capability"
info "Provider registers an AI service endpoint."
info "This is not an API listing — it's a chain-enforced service contract."

TX=$($BINARY tx oasyce_capability register \
    "GPT-4 Summarizer" \
    "https://api.example.com/summarize" \
    100000uoas \
    --description "Summarizes text using GPT-4 with structured output" \
    --tags "nlp,summarization,gpt4" \
    --rate-limit 60 \
    --from provider $COMMON 2>&1)
echo "$TX" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  TX Hash: {d.get(\"txhash\",\"N/A\")}')" 2>/dev/null || echo "  TX submitted"
wait_tx

info "Querying registered capabilities..."
$BINARY query oasyce_capability list --output json 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
caps = data.get('capabilities', [])
if caps:
    c = caps[-1]
    print(f'  Capability ID: {c[\"id\"]}')
    print(f'  Name:          {c[\"name\"]}')
    print(f'  Price/Call:    {c[\"price_per_call\"]}')
    print(f'  Active:        {c[\"is_active\"]}')
" 2>/dev/null || echo "  (query oasyce_capability list)"

CAP_ID=$($BINARY query oasyce_capability list --output json 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
caps = data.get('capabilities', [])
print(caps[-1]['id'] if caps else '')
" 2>/dev/null)

# ═══════════════════════════════════════════════════════════════
# Step 4: Invoke — Escrow + Challenge Window
# ═══════════════════════════════════════════════════════════════
section "STEP 4: Invoke Capability (Auto-Escrow)"
info "Consumer invokes the capability. Funds are locked in escrow."
info "NOT 'pay and hope' — funds are locked until service is verified."

if [ -n "$CAP_ID" ]; then
    TX=$($BINARY tx oasyce_capability invoke $CAP_ID \
        --input '{"text":"Summarize the Oasyce whitepaper in 3 sentences"}' \
        --from consumer $COMMON 2>&1)
    echo "$TX" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  TX Hash: {d.get(\"txhash\",\"N/A\")}')" 2>/dev/null || echo "  TX submitted"
    wait_tx
    info "Invocation created. Escrow is LOCKED."
    info "Provider must submit output → 100-block challenge window → then claim payment."
    info "Consumer can dispute within the window for a full refund."
else
    info "Skipping — no capability ID found"
fi

# ═══════════════════════════════════════════════════════════════
# Step 5: Settlement — Fee Split
# ═══════════════════════════════════════════════════════════════
section "STEP 5: Settlement Economics"
info "On successful claim after challenge window:"
info "  90% → Provider (earned revenue)"
info "   5% → Protocol (sustainability)"
info "   2% → Burned (deflationary pressure)"
info "   3% → Treasury (ecosystem fund)"
info ""
info "This is automatic, deterministic, and irreversible."
info "No chargebacks. No human intervention. Math, not trust."

# ═══════════════════════════════════════════════════════════════
# Step 6: Reputation — Credit Scoring
# ═══════════════════════════════════════════════════════════════
section "STEP 6: Reputation (Credit Scoring)"
info "Reputation is not decoration — it's an economic factor."
info "Time-decaying (30-day half-life), influences:"
info "  • Jury eligibility for dispute resolution"
info "  • Task assignment priority (x/work module)"
info "  • Access level caps (reputation gates data access)"

info "Querying reputation for provider..."
$BINARY query reputation show $PROVIDER --output json 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
rep = data.get('reputation', {})
print(f'  Address:   {rep.get(\"address\", \"N/A\")}')
print(f'  Score:     {rep.get(\"score\", \"0\")}')
print(f'  Feedbacks: {rep.get(\"total_feedbacks\", \"0\")}')
" 2>/dev/null || echo "  (query reputation show)"

# ═══════════════════════════════════════════════════════════════
section "DEMO COMPLETE"
echo -e "${GREEN}This demo showed a complete AI commerce lifecycle:${NC}"
echo ""
echo "  1. Data Securitization  — Data as financial instrument (bonding curve)"
echo "  2. Property Trading     — Equity-based access (Bancor curve pricing)"
echo "  3. Service Contracts    — Chain-enforced capability registration"
echo "  4. Escrowed Invocation  — Locked funds + challenge window"
echo "  5. Automatic Settlement — 90/5/2/3 fee split, no human intervention"
echo "  6. Credit Scoring       — Time-decaying reputation as economic factor"
echo ""
echo -e "${YELLOW}Stripe/x402/Tempo solve 'how to pay.'${NC}"
echo -e "${GREEN}Oasyce solves 'why the payment is justified.'${NC}"
