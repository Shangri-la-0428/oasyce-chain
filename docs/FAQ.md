# Oasyce Capability Surface — FAQ

> Looking for the general Oasyce FAQ (for users, investors, and developers)? See [oasyce-sdk/FAQ.md](https://github.com/Shangri-la-0428/oasyce-sdk/blob/main/FAQ.md).

## General

### What is the capability surface?

It is the chain-level service invocation surface. Providers register callable capabilities on-chain with a fixed price. Consumers invoke them, funds move into escrow, delivery enters a challenge window, and settlement or dispute becomes public fact on-chain.

### How is this different from Stripe / x402 / Tempo?

Those solve **how to pay**. Oasyce solves **why the payment is fair**:

| | Payment Rails | Oasyce |
|--|--------------|--------|
| Core problem | How to transfer money | Why the transfer is justified |
| Service call | API call + pay | On-chain contract (escrow + challenge window + arbitration) |
| Dispute | Customer support | On-chain consumer dispute within challenge window |
| Trust | Platform reputation | On-chain reputation score (time-decay + verifiable feedback) |
| Transparency | Provider dashboard | All usage data on-chain, publicly queryable |

---

## Pricing

### How does pricing work?

**Fixed price per call.** The provider sets `price_per_call` when registering the capability. Every invocation costs exactly that amount, locked in escrow until settlement.

```bash
oasyced tx oasyce_capability register "Codex API" \
  "https://provider.example.com/api/v1/process" 50000uoas \
  --from provider
```

### My upstream API charges per token. How do I price my capability?

**Don't do metered billing. Use tiered capabilities instead:**

Register multiple capabilities at different price points, each covering a different usage tier:

| Capability | Price | Upstream limit |
|-----------|-------|---------------|
| `codex-small` | 10 OAS | ≤1,000 tokens |
| `codex-medium` | 30 OAS | ≤5,000 tokens |
| `codex-large` | 80 OAS | ≤20,000 tokens |

Consumers pick the tier they need. Your provider agent enforces the token limit server-side. The price difference between your cost and the fixed price is your margin.

### Why not metered billing (pay-per-token)?

1. **Simplicity**: Fixed pricing is already fully supported on-chain — no partial escrow release, no unit tracking complexity.
2. **Predictability**: Consumers know the exact cost upfront. No surprise bills.
3. **Trust**: Metered billing requires trusting the provider's usage report. Fixed pricing eliminates that trust assumption.
4. **Speed**: No post-call settlement negotiation. Escrow locks exact amount, releases exact amount.

### Can I track actual token usage even with fixed pricing?

**Yes.** The `usage_report` field records resource consumption on-chain for transparency, without affecting billing:

```bash
oasyced tx oasyce_capability complete-invocation INV_001 <output_hash> \
  --usage-report '{"prompt_tokens":150,"completion_tokens":80,"total_tokens":230}'
```

This data is:
- Stored on-chain in the Invocation record
- Queryable via REST: `GET /oasyce/capability/v1/invocation/{id}`
- Useful for: cost analysis, consumer transparency, provider accounting
- **Does not affect payment** — the fixed `price_per_call` is always the settlement amount

The provider agent (`scripts/provider_agent.py`) auto-extracts token usage from OpenAI-compatible upstream API responses.

### What's the fee split?

On every successful settlement:

| Recipient | Share |
|-----------|-------|
| Provider | 90% |
| Protocol (validators) | 5% |
| Burn (deflationary) | 2% |
| Treasury | 3% |

---

## Provider Operations

### How do I become a provider?

1. **Register** your capability on-chain:
   ```bash
   oasyced tx oasyce_capability register "My API" \
     "https://my-server:8430/api/v1/process" 50000uoas \
     --description "GPT-4 wrapper with custom prompts" \
     --tags "nlp,gpt4,translation" \
     --from provider
   ```

2. **Run** the provider agent:
   ```bash
   export UPSTREAM_API_URL="https://api.openai.com/v1/chat/completions"
   export UPSTREAM_API_KEY="sk-..."
   export OASYCE_CAPABILITY_ID="CAP_..."
   python3 scripts/provider_agent.py
   ```

   On long-running VPS deployments, do not hand-edit capability IDs forever. Keep the current active ID in `/etc/oasyce/provider-capability.env` and rotate with:

   ```bash
   bash scripts/rotate_provider_capability.sh
   ```

3. The agent handles everything: verify invocation on-chain → call upstream → hash output → complete on-chain → auto-claim after challenge window.

### What is the challenge window?

After a provider submits the output hash (`CompleteInvocation`), there's a **100-block window** (~8 minutes at 5s/block) where the consumer can dispute the result.

- If **no dispute**: provider calls `ClaimInvocation` → escrow released (90/5/2/3 split)
- If **disputed**: consumer calls `DisputeInvocation` → escrow refunded to consumer
- The provider agent auto-claims after the window passes

### What if my upstream API fails?

The provider agent calls `FailInvocation`, which refunds the consumer's escrow in full. No penalty to the provider.

Single transient upstream failures do **not** deactivate the capability. The provider only auto-deactivates after repeated consecutive buyer-path failures (default threshold: 3), so short upstream blips do not take the service offline.

```bash
oasyced tx oasyce_capability fail-invocation INV_001 --from provider
```

### Why did I receive a burst of old alert emails?

There are two different failure modes:

1. **Live alert churn** — the monitor is still emitting fresh alerts.
2. **Delayed delivery** — the SMTP provider delivers older messages late.

The current healthcheck is hardened to reduce live churn:

- alert dedupe state is persisted under `/var/lib/oasyce-healthcheck`
- one healthcheck instance runs at a time via a lock file
- each alert key mails once per active incident and resets after recovery
- consumer stale monitoring is only enabled when the consumer is actually deployed
- provider HTTP monitoring and economy stale monitoring remain opt-in by default

If email bursts happen again, check `/var/log/oasyce-alert.log` and `/var/lib/oasyce-healthcheck/` first. If there are no new `ALERT:` lines and no new `.active` state files, the mailbox is receiving delayed delivery rather than fresh alerts.

### My local Mac cannot SSH to `47.93.32.88:29222`, but GitHub Actions or Cloud Assistant can. Is the VPS broken?

Not necessarily.

We have already verified a failure mode where:

- the VPS `sshd` process is healthy
- port `29222` is listening
- `UFW` and the ECS security group both allow `29222/tcp`
- GitHub Actions runners can log in with the same key
- Cloud Assistant is online and can run commands

In that situation, the remaining fault is usually **the path between your local machine and the ECS edge**, not the VPS itself.

Operational rule:

- treat **Alibaba Cloud CLI + Cloud Assistant** as the primary control plane
- treat **GitHub Actions manual workflows** as the secondary control plane
- treat **direct SSH** as a convenient entry, not the only entry

If local SSH still hangs during banner exchange, continue operating through `scripts/ecs_cloud_run.sh` or the GitHub Actions workflows instead of assuming the VPS is down.

### Can I update my price?

Yes:
```bash
oasyced tx oasyce_capability update CAP_001 \
  --price 80000uoas \
  --from provider
```

Existing in-flight invocations keep their original price. Only new invocations use the updated price.

### Can I take my capability offline?

```bash
oasyced tx oasyce_capability deactivate CAP_001 --from provider
```

No new invocations will be accepted. Existing in-flight invocations continue to settlement.

---

## Consumer Operations

### How do I invoke a capability?

```bash
oasyced tx oasyce_capability invoke CAP_001 \
  --input '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}' \
  --from consumer
```

This creates an escrow for the capability's `price_per_call` and returns an `invocation_id`.

### How do I get the result?

The invocation creates the on-chain record. To get the actual API result, POST to the provider's endpoint:

```bash
curl -X POST http://provider-host:8430/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{"invocation_id":"INV_001","input":{...}}'
```

The provider agent verifies your invocation on-chain before processing.

### How do I dispute a bad result?

Within the 100-block challenge window after the provider completes:

```bash
oasyced tx oasyce_capability dispute-invocation INV_001 \
  --reason "Output was incorrect / incomplete" \
  --from consumer
```

Your escrow is refunded immediately. The provider's reputation is affected.

### How do I check invocation status?

```bash
# CLI
oasyced query oasyce_capability invocation INV_001

# REST
curl http://<node>:1317/oasyce/capability/v1/invocation/INV_001
```

Returns: status, output_hash, usage_report, escrow_id, completed_height.

---

## Usage Tracking

### What does `usage_report` contain?

A free-form JSON string set by the provider. Typical format for LLM APIs:

```json
{
  "prompt_tokens": 150,
  "completion_tokens": 80,
  "total_tokens": 230,
  "model": "gpt-4"
}
```

The chain stores it as-is — no schema enforcement. This is purely for transparency and accounting.

### Is usage_report mandatory?

No. It's optional. If omitted, the invocation works exactly the same — fixed price settlement is unaffected.

### Can I query usage across all my invocations?

Currently you query per-invocation. Aggregation is done off-chain by querying each invocation's `usage_report` field. A batch query endpoint is planned.

### Can providers lie about usage?

Yes — `usage_report` is self-reported by the provider. But since **billing is fixed-price** (not usage-based), lying about usage doesn't affect what anyone pays. It only affects the provider's own cost accounting transparency.

If a consumer suspects the provider didn't actually process the request, they should **dispute** within the challenge window — that's the trust mechanism.

---

## Escrow & Settlement

### How does escrow work?

1. **Consumer invokes** → `price_per_call` locked from consumer to settlement module
2. **Provider completes** → output hash recorded, challenge window starts
3. **After 100 blocks** → provider claims → escrow released with fee split
4. **OR consumer disputes** → escrow refunded to consumer

### What if the provider never completes?

The escrow has a timeout (configurable, default from settlement params). After timeout, the escrow auto-refunds to the consumer via `ExpireStaleEscrows` in EndBlock.

### What if I lose connectivity during the challenge window?

- **Provider**: The provider agent retries claiming automatically. You can also claim manually later.
- **Consumer**: If you don't dispute within 100 blocks, the provider can claim. Plan accordingly.

---

## Reputation

### Does the capability surface affect reputation?

Yes. After settlement, consumers can submit feedback:

```bash
oasyced tx reputation submit-feedback INV_001 400 "fast and accurate"
```

Score (0-500) feeds into the provider's on-chain reputation, which affects:
- Future visibility in capability search
- Access level caps in datarights
- Jury selection eligibility

### What happens to reputation on dispute?

A disputed invocation signals unreliability. The dispute itself is recorded on-chain. Repeated disputes damage the provider's reputation score through negative feedback patterns.
