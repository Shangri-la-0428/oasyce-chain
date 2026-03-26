#!/bin/bash
# Oasyce chain healthcheck — runs via crontab every 5 min
# Alerts via email (msmtp) when chain is down or faucet low

ALERT_EMAIL="${OASYCE_ALERT_EMAIL:-ptc0428@qq.com}"
STATE_FILE="/tmp/oasyce_health_state"
API="http://127.0.0.1:11317"
FAUCET_ADDR="oasyce1msmqqjw64k8m827w3apda97umxt9lgfxszr25d"
FAUCET_MIN_UOAS=200000000  # 200 OAS

send_alert() {
    local msg="$1"
    local ts=$(date "+%Y-%m-%d %H:%M:%S")
    echo "$ts: ALERT: $msg" >> /var/log/oasyce-alert.log
    printf "Subject: [Oasyce Alert] %s\nFrom: Oasyce Monitor <$(grep ^from /etc/msmtprc | awk '{print $2}')>\nTo: %s\nContent-Type: text/plain; charset=utf-8\n\n%s\n\nTime: %s\nNode: 47.93.32.88\n" \
        "$msg" "$ALERT_EMAIL" "$msg" "$ts" | msmtp "$ALERT_EMAIL" 2>> /var/log/oasyce-alert.log
}

# 1. Check chain health
HEALTH=$(curl -s --max-time 5 "$API/health" 2>/dev/null)
STATUS=$(echo "$HEALTH" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null)
HEIGHT=$(echo "$HEALTH" | python3 -c "import sys,json; print(json.load(sys.stdin).get('block_height',0))" 2>/dev/null)

if [ "$STATUS" != "ok" ]; then
    PREV_FAIL=$(cat "$STATE_FILE" 2>/dev/null)
    if [ "$PREV_FAIL" = "fail" ]; then
        send_alert "Chain DOWN — health check failed twice"
    fi
    echo "fail" > "$STATE_FILE"
    exit 1
fi

# 2. Check block progression
PREV_HEIGHT=$(cat "${STATE_FILE}_height" 2>/dev/null || echo 0)
if [ "$HEIGHT" -le "$PREV_HEIGHT" ] && [ "$PREV_HEIGHT" -gt 0 ]; then
    send_alert "Chain STALLED — height stuck at $HEIGHT for 5+ min"
fi
echo "$HEIGHT" > "${STATE_FILE}_height"
echo "ok" > "$STATE_FILE"

# 3. Check faucet balance
BAL=$(curl -s --max-time 5 "$API/cosmos/bank/v1beta1/balances/$FAUCET_ADDR" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['balances'][0]['amount'] if d.get('balances') else 0)" 2>/dev/null)
if [ -n "$BAL" ] && [ "$BAL" -lt "$FAUCET_MIN_UOAS" ]; then
    OAS=$(python3 -c "print($BAL / 1000000)")
    send_alert "Faucet LOW — only ${OAS} OAS remaining"
fi
