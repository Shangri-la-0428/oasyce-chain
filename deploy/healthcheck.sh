#!/bin/bash
# Oasyce chain healthcheck — runs via crontab every 5 min
# Self-heals first, then alerts via email (msmtp)

ALERT_EMAIL="${OASYCE_ALERT_EMAIL:-ptc0428@qq.com}"
STATE_FILE="/tmp/oasyce_health_state"
API="http://127.0.0.1:11317"
FAUCET_ADDR="oasyce1msmqqjw64k8m827w3apda97umxt9lgfxszr25d"
FAUCET_MIN_UOAS=200000000  # 200 OAS

send_alert() {
    local msg="$1"
    local ts
    ts=$(date "+%Y-%m-%d %H:%M:%S")
    echo "$ts: ALERT: $msg" >> /var/log/oasyce-alert.log
    printf "Subject: [Oasyce Alert] %s\nFrom: Oasyce Monitor <ptc0428@qq.com>\nTo: %s\nContent-Type: text/plain; charset=utf-8\n\n%s\n\nTime: %s\nNode: 47.93.32.88\n" \
        "$msg" "$ALERT_EMAIL" "$msg" "$ts" | msmtp "$ALERT_EMAIL" 2>> /var/log/oasyce-alert.log
}

# 0. Self-heal: check and restart dead services
OASYCED_ACTIVE=$(systemctl is-active oasyced 2>/dev/null)
FAUCET_ACTIVE=$(systemctl is-active oasyce-faucet 2>/dev/null)

if [ "$OASYCED_ACTIVE" != "active" ]; then
    echo "$(date): oasyced down, restarting..." >> /var/log/oasyce-alert.log
    systemctl restart oasyced
    sleep 5
    if [ "$(systemctl is-active oasyced)" = "active" ]; then
        send_alert "oasyced was DOWN — auto-restarted successfully"
    else
        send_alert "oasyced DOWN — auto-restart FAILED, manual intervention needed"
        exit 1
    fi
fi

if [ "$FAUCET_ACTIVE" != "active" ]; then
    echo "$(date): faucet down, restarting..." >> /var/log/oasyce-alert.log
    systemctl restart oasyce-faucet
    sleep 2
    if [ "$(systemctl is-active oasyce-faucet)" = "active" ]; then
        send_alert "oasyce-faucet was DOWN — auto-restarted successfully"
    else
        send_alert "oasyce-faucet DOWN — auto-restart FAILED"
    fi
fi

# 0b. Self-heal provider + claude-proxy
for SVC in oasyce-provider claude-proxy; do
    if [ "$(systemctl is-active $SVC 2>/dev/null)" != "active" ]; then
        echo "$(date): $SVC down, restarting..." >> /var/log/oasyce-alert.log
        systemctl restart $SVC
        sleep 2
        if [ "$(systemctl is-active $SVC)" = "active" ]; then
            send_alert "$SVC was DOWN — auto-restarted successfully"
        else
            send_alert "$SVC DOWN — auto-restart FAILED"
        fi
    fi
done

# 1. Check chain health
HEALTH=$(curl -s --max-time 5 "$API/health" 2>/dev/null)
STATUS=$(echo "$HEALTH" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("status",""))' 2>/dev/null)
HEIGHT=$(echo "$HEALTH" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("block_height",0))' 2>/dev/null)

if [ "$STATUS" != "ok" ]; then
    PREV_FAIL=$(cat "$STATE_FILE" 2>/dev/null)
    if [ "$PREV_FAIL" = "fail" ]; then
        send_alert "Chain DOWN — health check failed twice (services running but API unresponsive)"
    fi
    echo "fail" > "$STATE_FILE"
    exit 1
fi

# 2. Check block progression
PREV_HEIGHT=$(cat "${STATE_FILE}_height" 2>/dev/null || echo 0)
if [ "$HEIGHT" -le "$PREV_HEIGHT" ] 2>/dev/null && [ "$PREV_HEIGHT" -gt 0 ]; then
    send_alert "Chain STALLED — height stuck at $HEIGHT for 5+ min"
fi
echo "$HEIGHT" > "${STATE_FILE}_height"
echo "ok" > "$STATE_FILE"

# 3. Check faucet balance
BAL=$(curl -s --max-time 5 "$API/cosmos/bank/v1beta1/balances/$FAUCET_ADDR" 2>/dev/null | \
    python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["balances"][0]["amount"] if d.get("balances") else 0)' 2>/dev/null)
if [ -n "$BAL" ] && [ "$BAL" -lt "$FAUCET_MIN_UOAS" ] 2>/dev/null; then
    OAS=$(python3 -c "print($BAL / 1000000)")
    send_alert "Faucet LOW — only ${OAS} OAS remaining"
fi

# 4. Economic metrics — provider earnings + settlement rate
PROVIDER_ADDR="oasyce1a57fdrtq2wu65tjeyx9jyg4cku4evr8en4gyv5"
ECON=$(curl -s --max-time 5 "$API/oasyce/capability/v1/earnings/$PROVIDER_ADDR" 2>/dev/null)
TOTAL_CALLS=$(echo "$ECON" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("total_calls","0"))' 2>/dev/null)
TOTAL_EARNED=$(echo "$ECON" | python3 -c 'import sys,json; e=json.load(sys.stdin).get("total_earned",[]); print(e[0]["amount"] if e else "0")' 2>/dev/null)

# Track call growth — alert if zero new calls in 2 hours (4 checks)
PREV_CALLS=$(cat "${STATE_FILE}_calls" 2>/dev/null || echo 0)
STALE_COUNT=$(cat "${STATE_FILE}_stale" 2>/dev/null || echo 0)
if [ -n "$TOTAL_CALLS" ] && [ "$TOTAL_CALLS" -gt 0 ] 2>/dev/null; then
    if [ "$TOTAL_CALLS" -le "$PREV_CALLS" ] 2>/dev/null; then
        STALE_COUNT=$((STALE_COUNT + 1))
        if [ "$STALE_COUNT" -ge 4 ]; then
            send_alert "Economy STALE — no new invocations in 2+ hours (total_calls=$TOTAL_CALLS)"
            STALE_COUNT=0
        fi
    else
        STALE_COUNT=0
    fi
    echo "$TOTAL_CALLS" > "${STATE_FILE}_calls"
    echo "$STALE_COUNT" > "${STATE_FILE}_stale"
fi

# 5. Consumer agent liveness — alert if last_run > 90 min ago
CONSUMER_STATE="/tmp/consumer_agent_state.json"
if [ -f "$CONSUMER_STATE" ]; then
    LAST_RUN=$(python3 -c 'import sys,json; print(json.load(open("'"$CONSUMER_STATE"'")).get("last_run",""))' 2>/dev/null)
    if [ -n "$LAST_RUN" ]; then
        AGE_MIN=$(python3 -c 'from datetime import datetime; d=datetime.strptime("'"$LAST_RUN"'","%Y-%m-%d %H:%M:%S"); print(int((datetime.now()-d).total_seconds()/60))' 2>/dev/null)
        if [ -n "$AGE_MIN" ] && [ "$AGE_MIN" -gt 90 ] 2>/dev/null; then
            send_alert "Consumer agent STALE — last run ${AGE_MIN}min ago (expected every 30min)"
        fi
    fi
fi

# 6. Provider HTTP health
PROV_HEALTH=$(curl -s --max-time 5 "http://127.0.0.1:8430/health" 2>/dev/null)
PROV_OK=$(echo "$PROV_HEALTH" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("status",""))' 2>/dev/null)
if [ "$PROV_OK" != "ok" ]; then
    send_alert "Provider agent HTTP health FAILED (systemd may be active but HTTP unresponsive)"
fi

# 7. Log economic summary (no alert, just for dashboarding)
CONSUMER_INV=$(python3 -c 'import json; d=json.load(open("'"$CONSUMER_STATE"'")); print(d.get("total_invocations",0), d.get("total_settlements",0))' 2>/dev/null)
echo "$(date '+%Y-%m-%d %H:%M:%S') height=$HEIGHT calls=$TOTAL_CALLS earned=${TOTAL_EARNED}uoas consumer=$CONSUMER_INV" >> /var/log/oasyce-econ.log
