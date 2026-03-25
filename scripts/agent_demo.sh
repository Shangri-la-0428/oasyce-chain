#!/bin/bash
# =============================================================================
# Oasyce Agent Economy Demo
#
# Demonstrates the full agent-to-agent transaction lifecycle:
#   1. Agent self-registers (PoW onboarding)
#   2. Provider registers an AI capability
#   3. Consumer invokes the capability (auto escrow + settlement)
#   4. Provider registers a data asset
#   5. Consumer buys data shares (Bancor bonding curve)
#   6. Consumer sells shares back (inverse curve + 3% protocol fee)
#   7. Reputation feedback after invocation
#
# Prerequisites: chain running locally with REST API enabled.
# Usage: bash scripts/agent_demo.sh
# =============================================================================
set -e

OASYCED="${OASYCED:-./build/oasyced}"
CHAIN_ID="${CHAIN_ID:-oasyce-local-1}"
NODE="tcp://localhost:26657"
REST="http://localhost:1317"
KB="--keyring-backend test"
FEES="--fees 10000uoas"
COMMON="$KB --chain-id $CHAIN_ID $FEES --yes"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

banner() { echo -e "\n${CYAN}${BOLD}=== $1 ===${NC}"; }
step()   { echo -e "${GREEN}>>> $1${NC}"; }
info()   { echo -e "${DIM}    $1${NC}"; }
result() { echo -e "${YELLOW}    $1${NC}"; }
wait_tx() { sleep 3; }

query_balance() {
  local addr=$1
  curl -s "$REST/cosmos/bank/v1beta1/balances/$addr" | \
    python3 -c "import sys,json; d=json.load(sys.stdin); b=[x for x in d.get('balances',[]) if x['denom']=='uoas']; print(b[0]['amount'] if b else '0')" 2>/dev/null
}

format_oas() {
  python3 -c "print(f'{int(\"$1\") / 1_000_000:.2f} OAS')"
}

# =============================================================================
echo -e "${BOLD}"
echo "    ____                              "
echo "   / __ \____ ________  ________ "
echo "  / / / / __ \`/ ___/ / / / ___/ _ \\"
echo " / /_/ / /_/ (__  ) /_/ / /__/  __/"
echo " \____/\__,_/____/\__, /\___/\___/ "
echo "                 /____/             "
echo ""
echo "  Where agents pay agents."
echo -e "${NC}"
echo -e "${DIM}  Agent Economy Demo — Full Transaction Lifecycle${NC}"
echo ""

# ---------------------------------------------------------------------------
banner "SETUP: Prepare Two Agent Accounts"
# ---------------------------------------------------------------------------

# Ensure provider and consumer keys exist
$OASYCED keys show provider $KB 2>/dev/null || $OASYCED keys add provider $KB 2>/dev/null
$OASYCED keys show consumer $KB 2>/dev/null || $OASYCED keys add consumer $KB 2>/dev/null

PROVIDER=$($OASYCED keys show provider -a $KB 2>/dev/null)
CONSUMER=$($OASYCED keys show consumer -a $KB 2>/dev/null)
VALIDATOR=$($OASYCED keys show validator -a $KB 2>/dev/null || $OASYCED keys show val0 -a $KB 2>/dev/null)

info "Provider: $PROVIDER"
info "Consumer: $CONSUMER"
info "Funder:   $VALIDATOR"

# Fund agents if needed
PROVIDER_BAL=$(query_balance "$PROVIDER")
CONSUMER_BAL=$(query_balance "$CONSUMER")

if [ "$PROVIDER_BAL" = "0" ] || [ "$CONSUMER_BAL" = "0" ]; then
  step "Funding agent accounts (100 OAS each)..."
  if [ "$PROVIDER_BAL" = "0" ]; then
    $OASYCED tx send "$VALIDATOR" "$PROVIDER" 100000000uoas --from validator $COMMON 2>/dev/null || \
    $OASYCED tx send "$VALIDATOR" "$PROVIDER" 100000000uoas --from val0 $COMMON 2>/dev/null
    wait_tx
  fi
  if [ "$CONSUMER_BAL" = "0" ]; then
    $OASYCED tx send "$VALIDATOR" "$CONSUMER" 100000000uoas --from validator $COMMON 2>/dev/null || \
    $OASYCED tx send "$VALIDATOR" "$CONSUMER" 100000000uoas --from val0 $COMMON 2>/dev/null
    wait_tx
  fi
fi

PROVIDER_BAL=$(query_balance "$PROVIDER")
CONSUMER_BAL=$(query_balance "$CONSUMER")
result "Provider balance: $(format_oas $PROVIDER_BAL)"
result "Consumer balance: $(format_oas $CONSUMER_BAL)"

# ---------------------------------------------------------------------------
banner "STEP 1: Provider Registers AI Capability"
# ---------------------------------------------------------------------------
step "Registering 'Translation API' — price 0.5 OAS/call..."

$OASYCED tx oasyce_capability register "Translation API" "https://api.agent.ai/translate" 500000uoas \
  --description "Neural machine translation, 100+ languages" \
  --tags "nlp,translation,agent-api" \
  --from provider $COMMON 2>/dev/null
wait_tx

CAP_ID=$($OASYCED query oasyce_capability list --node $NODE --output json 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); caps=[c for c in d.get('capabilities',[]) if c['name']=='Translation API']; print(caps[0]['id'] if caps else '')" 2>/dev/null)

result "Capability registered: $CAP_ID"
info "Price: 0.5 OAS/call | Tags: nlp, translation, agent-api"

# ---------------------------------------------------------------------------
banner "STEP 2: Consumer Invokes Capability"
# ---------------------------------------------------------------------------
step "Consumer calls Translation API (auto escrow + settlement)..."
info "Input: {\"text\": \"Where agents pay agents\", \"target\": \"zh\"}"

CONSUMER_BEFORE=$(query_balance "$CONSUMER")

$OASYCED tx oasyce_capability invoke "$CAP_ID" \
  --input '{"text":"Where agents pay agents","target":"zh"}' \
  --from consumer $COMMON 2>/dev/null
wait_tx

CONSUMER_AFTER=$(query_balance "$CONSUMER")
COST=$(python3 -c "print(int('$CONSUMER_BEFORE') - int('$CONSUMER_AFTER'))")

result "Invocation complete!"
info "Consumer paid: $(format_oas $COST) (0.5 OAS + gas)"
info "Flow: Consumer -> Escrow -> 93% Provider + 3% Validators + 2% Burn + 2% Treasury"

# ---------------------------------------------------------------------------
banner "STEP 3: Provider Registers Data Asset"
# ---------------------------------------------------------------------------
step "Registering 'Medical Imaging Dataset v2'..."

$OASYCED tx datarights register "Medical Imaging Dataset v2" "sha256:a1b2c3d4e5f6" \
  --description "50K annotated CT scans, DICOM format" \
  --rights-type original --tags "medical,imaging,ai-training" \
  --from provider $COMMON 2>/dev/null
wait_tx

ASSET_ID=$($OASYCED query datarights list --node $NODE --output json 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); a=[x for x in d.get('data_assets',[]) if 'Medical' in x.get('name','')]; print(a[0]['id'] if a else '')" 2>/dev/null)

result "Data asset registered: $ASSET_ID"

# ---------------------------------------------------------------------------
banner "STEP 4: Consumer Buys Data Shares (Bancor Bonding Curve)"
# ---------------------------------------------------------------------------
step "Buying shares with 10 OAS..."
info "Pricing: tokens = supply * (sqrt(1 + payment/reserve) - 1), CW=0.5"

CONSUMER_BEFORE=$(query_balance "$CONSUMER")

$OASYCED tx datarights buy-shares "$ASSET_ID" 10000000uoas --from consumer $COMMON 2>/dev/null
wait_tx

SHARES=$($OASYCED query datarights asset "$ASSET_ID" --node $NODE --output json 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data_asset',{}).get('total_shares','0'))" 2>/dev/null)
CONSUMER_AFTER=$(query_balance "$CONSUMER")

result "Shares purchased: $SHARES tokens"
result "Consumer spent: $(format_oas $(python3 -c "print(int('$CONSUMER_BEFORE') - int('$CONSUMER_AFTER'))"))"
info "Bonding curve: more buyers = higher price. No order book needed."

# ---------------------------------------------------------------------------
banner "STEP 5: Consumer Sells Shares (Inverse Curve)"
# ---------------------------------------------------------------------------
SELL_AMOUNT=$((SHARES / 2))
step "Selling $SELL_AMOUNT shares back..."
info "Payout: reserve * (1 - (1 - tokens/supply)^2), 95% reserve cap, 3% protocol fee"

CONSUMER_BEFORE=$(query_balance "$CONSUMER")

$OASYCED tx datarights sell-shares "$ASSET_ID" "$SELL_AMOUNT" --from consumer $COMMON 2>/dev/null
wait_tx

CONSUMER_AFTER=$(query_balance "$CONSUMER")
PAYOUT=$(python3 -c "print(int('$CONSUMER_AFTER') - int('$CONSUMER_BEFORE'))")

result "Sold $SELL_AMOUNT shares"
result "Received: $(format_oas $PAYOUT) (after 3% protocol fee)"

# ---------------------------------------------------------------------------
banner "STEP 6: Reputation Feedback"
# ---------------------------------------------------------------------------
step "Consumer rates the Translation API invocation..."

INV_COUNT=$($OASYCED query oasyce_capability list --node $NODE --output json 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); caps=[c for c in d.get('capabilities',[]) if c['name']=='Translation API']; print(caps[0].get('total_calls','0') if caps else '0')" 2>/dev/null)
INV_ID=$(printf "INV_%016x" $((INV_COUNT)))

$OASYCED tx reputation submit-feedback "$INV_ID" 450 \
  --comment "Fast and accurate translation" \
  --from consumer $COMMON 2>/dev/null
wait_tx

REP=$($OASYCED query reputation show "$PROVIDER" --node $NODE --output json 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); s=d.get('reputation_score',d.get('score',{})); print(s.get('score','N/A'))" 2>/dev/null)

result "Feedback submitted: 450/500"
result "Provider reputation score: $REP"
info "Score decays with 30-day half-life — stay active to maintain ranking."

# ---------------------------------------------------------------------------
banner "SUMMARY"
# ---------------------------------------------------------------------------
PROVIDER_FINAL=$(query_balance "$PROVIDER")
CONSUMER_FINAL=$(query_balance "$CONSUMER")

echo ""
echo -e "${BOLD}  Agent Economy Transaction Lifecycle — Complete!${NC}"
echo ""
echo -e "  ${CYAN}Provider${NC} (AI service + data owner)"
echo -e "    Balance: $(format_oas $PROVIDER_FINAL)"
echo -e "    Earned from: capability invocations + data share sales"
echo -e "    Reputation: $REP / 500"
echo ""
echo -e "  ${CYAN}Consumer${NC} (AI agent purchasing services)"
echo -e "    Balance: $(format_oas $CONSUMER_FINAL)"
echo -e "    Purchased: Translation API call + data shares"
echo ""
echo -e "  ${CYAN}Validators${NC}"
echo -e "    Earned: 3% of escrow releases + block rewards + gas fees"
echo -e "    Block reward: 4 OAS/block (halves every 10M blocks)"
echo ""
echo -e "  ${CYAN}Protocol${NC}"
echo -e "    Burned: 2% of every escrow release (deflationary)"
echo -e "    Treasury: 2% of every escrow release"
echo ""
echo -e "${DIM}  All transactions settled on-chain in ~5 seconds.${NC}"
echo -e "${DIM}  No KYC. No credit cards. No human approval.${NC}"
echo -e "${DIM}  Just agents paying agents.${NC}"
echo ""
