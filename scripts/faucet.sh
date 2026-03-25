#!/bin/bash
# =============================================================================
# Oasyce Testnet Faucet
#
# Sends test OAS tokens to a given address. For public testnet use.
#
# Usage:
#   bash scripts/faucet.sh <oasyce-address>
#   bash scripts/faucet.sh oasyce1abc... 50   # custom amount (OAS)
#
# Environment:
#   FAUCET_KEY    — key name for faucet account (default: faucet)
#   FAUCET_AMOUNT — default amount in OAS (default: 100)
#   CHAIN_ID      — chain ID (default: oasyce-testnet-1)
#   NODE          — RPC endpoint (default: tcp://localhost:26657)
# =============================================================================
set -e

OASYCED="${OASYCED:-oasyced}"
FAUCET_KEY="${FAUCET_KEY:-faucet}"
CHAIN_ID="${CHAIN_ID:-oasyce-testnet-1}"
NODE="${NODE:-tcp://localhost:26657}"
KB="--keyring-backend test"
DEFAULT_AMOUNT="${FAUCET_AMOUNT:-100}"

# Rate limiting: max 1 request per address per hour
RATE_LIMIT_DIR="/tmp/oasyce-faucet"
RATE_LIMIT_SECONDS=3600

RECIPIENT="${1:-}"
AMOUNT_OAS="${2:-$DEFAULT_AMOUNT}"

if [ -z "$RECIPIENT" ]; then
  echo "Oasyce Testnet Faucet"
  echo ""
  echo "Usage: $0 <oasyce-address> [amount-in-OAS]"
  echo ""
  echo "Examples:"
  echo "  $0 oasyce1abc...def        # sends ${DEFAULT_AMOUNT} OAS"
  echo "  $0 oasyce1abc...def 50     # sends 50 OAS"
  exit 1
fi

# Validate address prefix
if [[ ! "$RECIPIENT" =~ ^oasyce1[a-z0-9]{38}$ ]]; then
  echo "Error: Invalid address format. Expected: oasyce1<38 chars>"
  exit 1
fi

# Rate limiting
mkdir -p "$RATE_LIMIT_DIR"
RATE_FILE="$RATE_LIMIT_DIR/$RECIPIENT"
if [ -f "$RATE_FILE" ]; then
  LAST_REQUEST=$(cat "$RATE_FILE")
  NOW=$(date +%s)
  ELAPSED=$((NOW - LAST_REQUEST))
  if [ "$ELAPSED" -lt "$RATE_LIMIT_SECONDS" ]; then
    REMAINING=$(( (RATE_LIMIT_SECONDS - ELAPSED) / 60 ))
    echo "Error: Rate limited. Try again in ${REMAINING} minutes."
    exit 1
  fi
fi

# Convert OAS to uoas (1 OAS = 1,000,000 uoas)
AMOUNT_UOAS=$((AMOUNT_OAS * 1000000))

# Check faucet balance
FAUCET_ADDR=$($OASYCED keys show "$FAUCET_KEY" -a $KB 2>/dev/null)
if [ -z "$FAUCET_ADDR" ]; then
  echo "Error: Faucet key '$FAUCET_KEY' not found. Create it with:"
  echo "  $OASYCED keys add $FAUCET_KEY $KB"
  exit 1
fi

echo "Oasyce Testnet Faucet"
echo "  From:   $FAUCET_ADDR ($FAUCET_KEY)"
echo "  To:     $RECIPIENT"
echo "  Amount: $AMOUNT_OAS OAS ($AMOUNT_UOAS uoas)"
echo ""

# Send tokens
$OASYCED tx send "$FAUCET_ADDR" "$RECIPIENT" "${AMOUNT_UOAS}uoas" \
  --from "$FAUCET_KEY" \
  $KB \
  --chain-id "$CHAIN_ID" \
  --node "$NODE" \
  --fees 10000uoas \
  --yes 2>/dev/null

# Record rate limit
date +%s > "$RATE_FILE"

echo ""
echo "Sent $AMOUNT_OAS OAS to $RECIPIENT"
echo "Transaction submitted. Confirm in ~5 seconds."
