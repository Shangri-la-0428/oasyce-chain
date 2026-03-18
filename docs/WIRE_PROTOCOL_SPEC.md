# Oasyce Wire Protocol Specification

**Version**: 1.0.0-draft
**Status**: Draft
**Date**: 2026-03-19
**Task**: L01 -- Formal Wire Protocol Specification

---

## Table of Contents

1. [Overview](#1-overview)
2. [Token and Denomination](#2-token-and-denomination)
3. [Transaction Types](#3-transaction-types)
4. [Escrow Lifecycle](#4-escrow-lifecycle)
5. [Bonding Curve](#5-bonding-curve)
6. [Capability Invocation Flow](#6-capability-invocation-flow)
7. [Data Rights](#7-data-rights)
8. [Reputation System](#8-reputation-system)
9. [Dispute Resolution](#9-dispute-resolution)
10. [Governance](#10-governance)
11. [Economic Parameters](#11-economic-parameters)
12. [Security Tiers](#12-security-tiers)
13. [Diminishing Returns (Share Minting)](#13-diminishing-returns-share-minting)
14. [Block and Network Wire Formats](#14-block-and-network-wire-formats)

---

## 1. Overview

Oasyce is a decentralized AI capability marketplace and data rights settlement
network. Data owners register assets with cryptographic proof-of-provenance
(PoPc), AI developers list intelligent services, and autonomous agents
discover, price, trade, and settle transactions using bonding curves and
escrow-protected OAS tokens.

### 1.1 The Four Asset Types

All assets are validated through a unified Schema Registry.

| Type         | Description                                    | Example                              |
|--------------|------------------------------------------------|--------------------------------------|
| `data`       | Registered data assets with PoPc certificates  | Medical imaging dataset              |
| `capability` | AI services with callable HTTP endpoints       | Translation API, image generation    |
| `oracle`     | External data feeds                            | Price feeds, weather data            |
| `identity`   | Network identity records                       | Validator registration, agent profile|

### 1.2 Settlement Model

All value exchange follows the pattern:

```
Register --> Price (bonding curve) --> Escrow Lock --> Execute --> Settle (release/refund)
```

- **No order book.** Prices are determined algorithmically by a bonding curve.
- **Escrow-first.** Consumer funds are locked before any execution begins.
- **Deterministic settlement.** Release on success (provider paid minus fee), refund on failure (consumer made whole).

### 1.3 Consensus

Proof-of-Stake (PoS) with event-sourced state. A single entry point
(`apply_operation`) processes all state mutations. The chain is organized into
epochs and slots, with block production driven by stake-weighted leader
election.

---

## 2. Token and Denomination

### 2.1 OAS Token

OAS is the native protocol token. All staking, settlement, fees, and slashing
are denominated in OAS.

### 2.2 Denomination

| Unit  | Relation       | Example                |
|-------|----------------|------------------------|
| OAS   | 1 OAS          | Human-readable unit    |
| uoas  | 10^-8 OAS      | Smallest on-chain unit |

```
1 OAS = 100,000,000 uoas (10^8)
```

**Rule**: All on-chain amounts MUST be represented as unsigned 64-bit integers
in uoas. No floating-point values are permitted for monetary amounts in
consensus-critical code.

### 2.3 Supply Model

- **No hard supply cap.** Block rewards follow a halving schedule that
  converges asymptotically.
- **Asymptotic limit**: `halving_interval * base_reward * 2` total OAS.
  With mainnet parameters (4.0 OAS/block, 1,000,000-block halving interval),
  this converges toward ~8,000,000 OAS.
- **Deflationary pressure**: Slashing burns and protocol fee burns reduce
  effective circulating supply.

### 2.4 Multi-Asset Support

The protocol supports additional asset types registered via `REGISTER_ASSET`:

| Asset             | Decimals | Unit Relation          |
|-------------------|----------|------------------------|
| `OAS`             | 8        | 1 OAS = 10^8 units    |
| `USDC`            | 6        | 1 USDC = 10^6 units   |
| `DATA_CREDIT`     | 0        | 1 DC = 1 unit          |
| `CAPABILITY_TOKEN`| 0        | 1 CT = 1 unit          |

Custom asset types may be registered on-chain. Each has a per-address balance
tracked in the multi-asset ledger.

---

## 3. Transaction Types

### 3.1 Base Operation Envelope

Every state-changing message is expressed as an `Operation` -- an immutable
record that flows through a single entry point: `apply_operation()`.

```
Operation {
    op_type:         enum OperationType   // REQUIRED
    validator_id:    string               // target validator (context-dependent)
    amount:          uint64               // uoas units, MUST be >= 0
    asset_type:      string               // default "OAS"
    from_addr:       string               // sender address
    to_addr:         string               // recipient address
    reason:          string               // human-readable metadata
    commission_rate: uint32               // basis points [0, 5000]
    signature:       string               // Ed25519 signature (hex)
    chain_id:        string               // replay protection
    sender:          string               // Ed25519 public key (hex)
    timestamp:       uint64               // unix timestamp (anti-replay)
    nonce:           uint64               // per-sender sequence number
}
```

### 3.2 Global Validation Rules

Applied to ALL operations before type-specific validation:

1. **Signature check**: When signature enforcement is active, all user-initiated
   operations MUST include a valid Ed25519 signature over the canonical
   serialization of the operation, and the `sender` field must contain the
   corresponding public key. System operations (`SLASH`, `REWARD`) are exempt.
2. **Chain ID**: If `chain_id` is non-empty, it MUST match the node's
   configured chain ID.
3. **Nonce**: If the sender has previously submitted operations, the `nonce`
   MUST equal the sender's current nonce (monotonically increasing, starting at 0).
4. **Amount non-negative**: The `amount` field MUST be >= 0 (enforced at
   construction).

### 3.3 Consensus Module Operations

#### 3.3.1 REGISTER -- Register a New Validator

```
op_type:         REGISTER
validator_id:    string       // unique validator identifier
amount:          uint64       // self-stake in uoas, MUST >= MIN_STAKE
commission_rate: uint32       // basis points [0, 5000]
```

**Validation**:
- `amount` >= `MIN_STAKE` (default: 10,000 OAS = 1,000,000,000,000 uoas)
- `commission_rate` in range [0, 5000] (0% to 50%)
- `validator_id` must not already be registered (unless status is `exited` with no pending unbondings)

**State transition**:
- Create validator entry with status `active`, self-stake = `amount`, commission = `commission_rate`
- Debit `amount` from sender balance

#### 3.3.2 DELEGATE -- Delegate Stake to a Validator

```
op_type:         DELEGATE
from_addr:       string       // delegator address
validator_id:    string       // target validator
amount:          uint64       // uoas to delegate, MUST > 0
```

**Validation**:
- `amount` > 0
- Target validator exists
- Target validator status is `active` or `jailed`

**State transition**:
- Record delegation from `from_addr` to `validator_id`
- Debit `amount` from delegator balance
- Credit `amount` to validator's total stake

#### 3.3.3 UNDELEGATE -- Withdraw Delegated Stake

```
op_type:         UNDELEGATE
from_addr:       string       // delegator address
validator_id:    string       // target validator
amount:          uint64       // uoas to undelegate, MUST > 0
```

**Validation**:
- `amount` > 0
- Target validator exists
- `from_addr` has an active delegation >= `amount` to the target validator

**State transition**:
- Enter unbonding queue with 28-day (configurable) cooldown
- Delegation balance reduced immediately
- Funds released to delegator after unbonding period completes

#### 3.3.4 EXIT -- Voluntary Validator Exit

```
op_type:         EXIT
validator_id:    string       // exiting validator
```

**Validation**:
- Validator exists
- Validator status is not already `exited`

**State transition**:
- Validator status set to `exited`
- Self-stake enters unbonding queue
- All delegations enter unbonding queue

#### 3.3.5 UNJAIL -- Request Unjail After Penalty

```
op_type:         UNJAIL
validator_id:    string       // jailed validator
```

**Validation**:
- Validator exists
- Validator status is `jailed`
- Jail duration has expired

**State transition**:
- Validator status restored to `active`

#### 3.3.6 SLASH -- Penalize a Validator (system-generated)

```
op_type:         SLASH
validator_id:    string       // penalized validator
reason:          string       // "offline" | "double_sign" | "low_quality"
```

**Validation**:
- System-generated only (no user signature required)

**State transition**:
- Compute slash amount = `apply_rate_bps(total_stake, slash_rate_bps)`
- Deduct from self-stake first
- Any remainder deducted proportionally from delegators
- Slashed funds are burned (removed from total supply)
- If `reason` is "offline" or "double_sign", validator is jailed

Slash rates:

| Reason        | Rate (bps) | Percentage | Jail Duration     |
|---------------|-----------|------------|-------------------|
| `offline`     | 100       | 1%         | Standard          |
| `double_sign` | 500       | 5%         | 3x standard       |
| `low_quality` | 50        | 0.5%       | None (unless stake < MIN_STAKE) |

#### 3.3.7 REWARD -- Distribute Rewards (system-generated)

```
op_type:         REWARD
validator_id:    string       // receiving validator
```

**Validation**:
- System-generated only

**State transition**:
- Computed at epoch boundary
- Block rewards + work rewards distributed per reward split formula (see Section 11)

#### 3.3.8 TRANSFER -- Transfer Assets Between Addresses

```
op_type:         TRANSFER
from_addr:       string       // sender
to_addr:         string       // recipient
asset_type:      string       // "OAS", "USDC", etc.
amount:          uint64       // units of specified asset, MUST > 0
```

**Validation**:
- `amount` > 0
- `from_addr` and `to_addr` are non-empty and different
- `asset_type` is registered in the asset registry
- Sender has sufficient balance of the specified asset

**State transition**:
- Debit `from_addr` by `amount` of `asset_type`
- Credit `to_addr` by `amount` of `asset_type`

#### 3.3.9 REGISTER_ASSET -- Register a New Asset Type

```
op_type:         REGISTER_ASSET
asset_type:      string       // new asset identifier, MUST be non-empty
from_addr:       string       // issuer address
reason:          string       // human-readable name
commission_rate: uint32       // repurposed as decimals (0-18)
```

**Validation**:
- `asset_type` is non-empty
- `asset_type` not already registered
- `from_addr` (issuer) is non-empty

**State transition**:
- Add new asset type to the asset registry with specified decimals
- Issuer recorded as the asset creator

### 3.4 Settlement Module Operations

#### 3.4.1 ESCROW_LOCK -- Lock Funds in Escrow

```
EscrowLock {
    escrow_id:      string       // generated: "ESC_" + hex(16)
    consumer_id:    string       // REQUIRED
    provider_id:    string       // REQUIRED
    capability_id:  string       // REQUIRED
    amount:         uint64       // uoas, MUST > 0
    ttl:            uint32       // seconds until auto-refund, default 300
    invocation_id:  string       // linked invocation
    auth_token:     string       // hex(32 bytes), returned to caller
}
```

**Validation**:
- `amount` > 0
- `consumer_id` non-empty
- `provider_id` non-empty
- Consumer has sufficient OAS balance

**State transition**:
- Debit `amount` from consumer balance
- Create escrow entry with status `LOCKED`
- Return `escrow_id` and `auth_token` to caller

#### 3.4.2 ESCROW_RELEASE -- Release Escrowed Funds to Provider

```
EscrowRelease {
    escrow_id:      string       // REQUIRED
    auth_token:     string       // REQUIRED, must match lock auth_token
}
```

**Validation**:
- Escrow exists and status is `LOCKED`
- `auth_token` matches

**State transition**:
- Compute `protocol_fee = amount * PROTOCOL_FEE_BPS / 10000`
- Compute `provider_amount = amount - protocol_fee`
- Credit `provider_amount` to provider
- Credit `protocol_fee` to protocol treasury
- Set escrow status to `RELEASED`

#### 3.4.3 ESCROW_REFUND -- Refund Escrowed Funds to Consumer

```
EscrowRefund {
    escrow_id:      string       // REQUIRED
    auth_token:     string       // REQUIRED, must match lock auth_token
}
```

**Validation**:
- Escrow exists and status is `LOCKED`
- `auth_token` matches

**State transition**:
- Credit full `amount` back to consumer
- Set escrow status to `REFUNDED`

#### 3.4.4 ESCROW_EXPIRE -- Auto-Refund Stale Escrows (system-generated)

**Trigger**: `created_at + ttl < now`

**State transition**:
- Credit full `amount` back to consumer
- Set escrow status to `EXPIRED`

### 3.5 Capability Module Operations

#### 3.5.1 CAPABILITY_REGISTER -- Register an AI Endpoint

```
CapabilityRegister {
    capability_id:   string       // generated: "CAP_" + hex(SHA256(content)[:16])
    provider_id:     string       // REQUIRED, provider public key
    name:            string       // REQUIRED, human-readable name
    endpoint_url:    string       // REQUIRED, HTTP POST endpoint
    api_key_enc:     string       // AES-GCM encrypted API key (hex)
    price_per_call:  uint64       // uoas per invocation, >= 0 (0 = free)
    rate_limit:      uint32       // max calls/minute, 0 = unlimited
    input_schema:    string       // JSON Schema for request validation
    output_schema:   string       // JSON Schema for response validation
    tags:            string[]     // discovery tags
    description:     string       // free-text description
}
```

**Validation**:
- `endpoint_url` non-empty
- `endpoint_url` must not resolve to private/internal addresses (SSRF protection)
- `provider_id` non-empty
- `name` non-empty
- `price_per_call` >= 0
- Provider must hold >= `MIN_PROVIDER_STAKE` (100 OAS = 10,000,000,000 uoas) in OAS balance (Sybil barrier)

**State transition**:
- Create capability entry with status `active`
- Statistics initialized: `total_calls=0`, `total_earned=0`, `success_rate=1.0`

#### 3.5.2 CAPABILITY_INVOKE -- Invoke a Capability (via Settlement Protocol)

```
CapabilityInvoke {
    invocation_id:   string       // generated: "INV_" + hex(16)
    capability_id:   string       // REQUIRED
    consumer_id:     string       // REQUIRED
    input_payload:   bytes        // JSON-encoded input
    escrow_ttl:      uint32       // seconds, default 300
}
```

This is an orchestrated flow, not a single atomic operation. See Section 6 for
the full lifecycle.

#### 3.5.3 CAPABILITY_SUSPEND -- Suspend a Capability

Sets capability status to `suspended`. Prevents new invocations.

#### 3.5.4 CAPABILITY_DELIST -- Permanently Delist a Capability

Sets capability status to `delisted`. Irreversible.

### 3.6 Data Rights Module Operations

#### 3.6.1 DATA_REGISTER -- Register a Data Asset

```
DataRegister {
    asset_id:          string       // generated from content hash
    filename:          string       // original filename
    owner:             string       // REQUIRED, owner address
    tags:              string[]     // discovery tags
    asset_type:        string       // "data" | "capability" | "oracle" | "identity"
    risk_level:        string       // "public" | "internal" | "sensitive"
    max_access_level:  string       // "L0" | "L1" | "L2" | "L3"
    rights_type:       string       // "original" | "co_creation" | "licensed" | "collection"
    co_creators:       CoCreator[]  // optional co-creator shares
    popc_signature:    string       // proof-of-provenance certificate signature
    content_hash:      string       // SHA-256 of file content
    file_size_bytes:   uint64       // file size
    schema_version:    string       // default "1.0"
}

CoCreator {
    address:  string
    share:    uint32     // percentage points (0-100), all shares MUST sum to 100
}
```

**Validation**:
- `owner` non-empty
- `rights_type` in `{original, co_creation, licensed, collection}`
- If `co_creators` present, shares must sum to 100
- `risk_level` in `{public, internal, sensitive}`
- Content hash must be unique (no duplicate registrations)

**State transition**:
- Asset registered in Schema Registry
- PoPc certificate recorded
- `max_access_level` auto-derived from `risk_level`:
  - `public` -> `L3`
  - `internal` -> `L3`
  - `sensitive` -> `L2`

#### 3.6.2 DATA_DISPUTE -- File a Dispute Against an Asset

```
DataDispute {
    asset_id:     string       // REQUIRED
    disputant:    string       // REQUIRED, disputant address
    reason:       string       // REQUIRED, human-readable reason
}
```

**State transition**:
- Set `disputed = true` on asset
- Set `dispute_status = "open"`
- Record `dispute_reason`

#### 3.6.3 DATA_RESOLVE -- Resolve a Dispute

```
DataResolve {
    asset_id:     string       // REQUIRED
    remedy:       string       // "delist" | "transfer" | "rights_correction" | "share_adjustment"
    details:      object       // remedy-specific details
}
```

**Validation**:
- Asset must have `dispute_status == "open"`
- `remedy` must be one of the valid types

**State transition by remedy**:

| Remedy               | Effect                                           |
|----------------------|--------------------------------------------------|
| `delist`             | Set `delisted = true`, asset removed from trading |
| `transfer`           | Change `owner` to `details.new_owner`            |
| `rights_correction`  | Update `rights_type` to `details.new_rights_type`|
| `share_adjustment`   | Modify co-creator share allocations              |

### 3.7 Reputation Module Operations

#### 3.7.1 FEEDBACK_SUBMIT -- Submit Execution Feedback

```
FeedbackSubmit {
    skill_id:        string       // capability/asset being rated
    success:         bool         // did the execution succeed?
    latency_ms:      uint32       // observed latency
    caller_rating:   float32      // 0.0 to 5.0 (clamped)
    invocation_id:   string       // optional, links to settlement record
}
```

**Validation**:
- `caller_rating` clamped to [0.0, 5.0]

**State transition**:
- If `invocation_id` exists in the valid invocation set, mark feedback as `verified`
- Otherwise, mark as `unverified`
- Append to feedback store (max 200 records per skill, FIFO eviction)

---

## 4. Escrow Lifecycle

### 4.1 State Machine

```
                  +--------+
                  | (init) |
                  +---+----+
                      |
                      | lock()
                      v
                 +--------+
           +---->| LOCKED |<----+
           |     +---+----+     |
           |         |          |
     (no)  |    +----+----+    | (no)
           |    |    |    |    |
           |    v    v    v    |
       +---+--+ | +------+ +--+---+
       |REFUND| | |EXPIRE| |RELEASE|
       |  ED  | | |  D   | |  D   |
       +------+ | +------+ +------+
                 |
         (auth_token invalid)
```

```
LOCKED ---[release(auth_token)]--> RELEASED
LOCKED ---[refund(auth_token)]---> REFUNDED
LOCKED ---[TTL expires]----------> EXPIRED
```

### 4.2 Terminal States

`RELEASED`, `REFUNDED`, and `EXPIRED` are terminal. No further transitions are
possible from these states.

### 4.3 Fee Split on RELEASE

When an escrow is released (successful settlement):

```
total_amount        = escrow.amount
protocol_fee        = total_amount * 500 / 10000          // 5%
provider_amount     = total_amount - protocol_fee          // 95%
```

The protocol fee is credited to `protocol_treasury`.

### 4.4 Extended Fee Split (Data Asset Purchases)

For data asset purchases (as opposed to capability invocations), the fee
distribution is:

| Recipient       | Share | Description                        |
|-----------------|-------|------------------------------------|
| Asset owner     | 60%   | Creator/owner of the data asset    |
| Validator       | 20%   | Validator who processes settlement  |
| Protocol        | 15%   | Protocol treasury                  |
| Burn            | 5%    | Permanently removed from supply    |

```
owner_amount     = amount * 6000 / 10000
validator_amount = amount * 2000 / 10000
protocol_amount  = amount * 1500 / 10000
burn_amount      = amount * 500  / 10000
```

All arithmetic is integer-only. Rounding dust (if any) goes to the protocol
treasury.

### 4.5 TTL and Auto-Expiry

- Default TTL: 300 seconds (5 minutes)
- A background process periodically scans for stale escrows where
  `created_at + ttl < now`
- Expired escrows are auto-refunded to the consumer

---

## 5. Bonding Curve

### 5.1 Formula

```
final_price = max(
    base_price * demand * scarcity * quality * freshness * rights_type,
    min_price
)
```

### 5.2 Factor Definitions

| Factor         | Formula                                    | Range       | Purpose                    |
|----------------|--------------------------------------------|-------------|----------------------------|
| `demand`       | `1 + alpha * ln(1 + query_count)`          | [1, +inf)   | More queries drive up price|
| `scarcity`     | `1 / (1 + similar_count)`                  | (0, 1]      | Rare data is worth more    |
| `quality`      | `min(1 + weight * contribution_score, 1.5)`| [1, 1.5]    | Better data earns premium  |
| `freshness`    | `0.5^(days / halflife) + 0.5`              | (0.5, 1.5]  | Decays toward 0.5 over time|
| `rights_type`  | Lookup table (see below)                   | [0.3, 1.0]  | Rights origin affects value|

### 5.3 Rights Type Multipliers

| Rights Type    | Multiplier | Description                |
|----------------|-----------|----------------------------|
| `original`     | 1.0       | Original work by registrant|
| `co_creation`  | 0.9       | Jointly created work       |
| `licensed`     | 0.7       | Licensed for resale        |
| `collection`   | 0.3       | Personal collection/curated|

### 5.4 Default Configuration

| Parameter                   | Default | Description                     |
|-----------------------------|---------|---------------------------------|
| `demand_alpha`              | 0.1     | Demand growth coefficient       |
| `scarcity_base`             | 1.0     | Scarcity baseline               |
| `freshness_halflife_days`   | 180     | Days until freshness halves     |
| `min_price`                 | 0.001   | OAS floor price                 |
| `contribution_score_weight` | 0.5     | Quality score influence         |

### 5.5 Pricing Modes

#### Auto (default)

The bonding curve computes the price dynamically. No manual intervention.

```
final_price = bonding_curve(base_price, factors...)
```

#### Fixed

The seller sets an exact price. The bonding curve is bypassed entirely.

```
final_price = manual_price       // manual_price MUST be > 0
```

#### Floor

The bonding curve runs normally, but the price never drops below the seller's
floor.

```
final_price = max(bonding_curve_result, manual_price)
```

**Validation**: For `fixed` and `floor` modes, `manual_price` MUST be > 0.

---

## 6. Capability Invocation Flow

### 6.1 Full Lifecycle

```
+----------+     +----------+     +---------+     +---------+     +----------+
| Register |---->| Discover |---->| Escrow  |---->| Gateway |---->| Settle   |
|          |     |          |     |  Lock   |     |  Call   |     | (release |
|          |     |          |     |         |     |         |     | /refund) |
+----------+     +----------+     +---------+     +---------+     +----------+
```

### 6.2 Step 1: Register

Provider registers an AI endpoint:

```
CapabilityRegister {
    name:           "Translation API"
    endpoint_url:   "https://api.example.com/translate"
    api_key_enc:    <AES-GCM encrypted>
    price_per_call: 50000000              // 0.5 OAS
    tags:           ["nlp", "translation"]
    rate_limit:     60                    // calls/minute
    input_schema:   {"type":"object", "properties":{"text":{"type":"string"}}}
    output_schema:  {"type":"object", "properties":{"result":{"type":"string"}}}
}
```

API keys are encrypted with AES-256-GCM (PBKDF2-derived key, 100k iterations)
before storage. The encryption passphrase is node-local and never transmitted.

### 6.3 Step 2: Discover

The discovery engine uses a four-layer pipeline:

1. **Intent matching**: Parse natural language intents
2. **Semantic search**: Vector similarity on capability descriptions
3. **Tag filtering**: Exact tag match
4. **Ranking**: Trust score + economic signals + feedback loop

### 6.4 Step 3: Escrow Lock

```
consumer calls invoke(capability_id, consumer_id, input_payload)
  |
  +--> Look up capability in registry
  +--> Verify capability status == "active"
  +--> Lock escrow: amount = price_per_call, ttl = escrow_ttl
  |      On lock failure (insufficient balance): return error
  +--> Record invocation with status IN_PROGRESS
```

For free capabilities (`price_per_call == 0`), the escrow step is skipped
entirely.

### 6.5 Step 4: Gateway Call

The gateway routes the request to the provider's registered endpoint:

```
POST {endpoint_url}
Authorization: Bearer {decrypted_api_key}
Content-Type: application/json

{input_payload}
```

The gateway enforces:
- SSRF protection (reject private/internal URLs)
- Timeout enforcement
- Response validation against `output_schema`

### 6.6 Step 5: Settle

```
if gateway_result.success:
    release escrow (auth_token)
      --> provider receives: amount - protocol_fee
      --> protocol treasury receives: protocol_fee
    update invocation status = SUCCESS
    update capability stats (total_calls, avg_latency, success_rate, total_earned)
else:
    refund escrow (auth_token)
      --> consumer receives: full amount
    update invocation status = FAILED
    update capability stats (failure recorded)
```

### 6.7 Invocation Record

Every invocation produces an immutable settlement record:

```
InvocationRecord {
    invocation_id:   string     // "INV_" + hex(16)
    capability_id:   string
    consumer_id:     string
    provider_id:     string
    amount:          uint64     // uoas paid
    status:          enum       // PENDING | IN_PROGRESS | SUCCESS | FAILED | TIMEOUT | DISPUTED
    input_hash:      string     // SHA-256 of input (privacy-preserving)
    output_hash:     string     // SHA-256 of output
    escrow_id:       string
    latency_ms:      float64
    provider_earned: uint64     // uoas after fee
    protocol_fee:    uint64     // uoas
    created_at:      uint64     // unix timestamp
    settled_at:      uint64     // unix timestamp
    error:           string
}
```

### 6.8 Invocation Status State Machine

```
PENDING ---[escrow locked]--> IN_PROGRESS
IN_PROGRESS ---[call success]--> SUCCESS
IN_PROGRESS ---[call failure]--> FAILED
IN_PROGRESS ---[TTL expired]---> TIMEOUT
SUCCESS ---[consumer disputes]--> DISPUTED
FAILED ---[terminal]
TIMEOUT ---[terminal]
```

---

## 7. Data Rights

### 7.1 Registration Pipeline

The data registration pipeline runs five stages:

```
Scan --> Classify --> Metadata --> PoPc Certificate --> Register
```

1. **Scan**: File is hashed (SHA-256) and fingerprinted
2. **Classify**: Risk engine auto-classifies as `public` / `internal` / `sensitive`
3. **Metadata**: Tags, timestamps, file size, semantic vector extracted
4. **PoPc**: Proof-of-Provenance Certificate generated (Ed25519 signature over content hash + metadata)
5. **Register**: Asset stored in Schema Registry with unique `asset_id`

### 7.2 Rights Types

| Rights Type    | Multiplier | Description                              |
|----------------|-----------|------------------------------------------|
| `original`     | 1.0       | Original work by the registrant          |
| `co_creation`  | 0.9       | Jointly created with declared co-creators|
| `licensed`     | 0.7       | Licensed from another party for resale   |
| `collection`   | 0.3       | Curated/collected from public sources    |

The rights type multiplier directly affects the bonding curve price.

### 7.3 Co-Creator Shares

When `rights_type == "co_creation"`, the registrant MUST declare co-creators:

```
co_creators: [
    { "address": "addr1", "share": 60 },
    { "address": "addr2", "share": 40 }
]
```

**Constraint**: All shares MUST sum to exactly 100.

Revenue from asset sales is distributed proportionally according to declared
shares.

### 7.4 Content Fingerprinting

Every registered asset receives a fingerprint for enforcement:

```
FingerprintResult {
    fingerprint:       string     // hex-encoded fingerprint hash
    content_hash:      string     // SHA-256 of original content
    content_size:      uint64     // bytes
    watermark_found:   bool
    watermark_data:    string     // extracted watermark payload
    similarity_score:  uint32     // 0-10000 basis points (10000 = exact match)
}
```

### 7.5 Watermarking

Content may be watermarked for distribution tracing. Watermark payloads contain:
- Asset ID
- Owner address
- Timestamp
- Buyer address (at time of purchase)

Watermarks enable the enforcement system to trace unauthorized distribution
back to the leaking party.

### 7.6 Infringement Types

| Type                          | Description                              |
|-------------------------------|------------------------------------------|
| `unauthorized_distribution`   | Content found on unauthorized platform   |
| `content_tampering`           | Content modified without permission      |
| `license_violation`           | Terms of license breached                |
| `attribution_missing`         | Required attribution not present         |

---

## 8. Reputation System

### 8.1 Trust Score Calculation

The trust score for a skill/capability is computed as a time-decayed weighted
average of execution feedback:

```
score_per_record = 0.6 * success(0|1) + 0.4 * (caller_rating / 5.0)

decay = exp(-ln(2) * age_seconds / (DECAY_HALF_LIFE_DAYS * 86400))

verification_weight = VERIFIED_WEIGHT if verified else UNVERIFIED_WEIGHT

weight = decay * verification_weight

trust_score = sum(weight_i * score_i) / sum(weight_i)
```

### 8.2 Feedback Verification

| Source                           | Weight   | Condition                              |
|----------------------------------|----------|----------------------------------------|
| Verified (valid `invocation_id`) | 1.0x     | `invocation_id` exists in settlement DB|
| Unverified (no/invalid ID)       | 0.1x     | Cannot prove actual execution          |

**Rationale**: Verified feedback comes from callers who actually paid for and
received the service. Unverified feedback (no linked invocation) is counted at
1/10th weight to reduce Sybil manipulation while still allowing subjective
input.

### 8.3 Time Decay

- **Half-life**: 30 days (configurable via `DECAY_HALF_LIFE_DAYS`)
- Feedback older than ~90 days contributes < 12.5% of its original weight
- This ensures the trust score reflects recent performance

### 8.4 Record Limits

- Maximum 200 feedback records per skill (FIFO eviction)
- Minimum records for meaningful score: implementation-defined (return `None` if no records)

### 8.5 Slashing Conditions Affecting Reputation

Validators and providers face three slashing conditions:

| Condition       | Trigger                                           | Slash Rate | Jail    |
|-----------------|---------------------------------------------------|-----------|---------|
| Offline         | Missed >50% of assigned slots in an epoch         | 1% (100 bps)  | Yes     |
| Double Sign     | Two blocks signed at the same height               | 5% (500 bps)  | Yes (3x)|
| Low Quality     | Avg quality score < 0.3 over last 10 tasks         | 0.5% (50 bps) | No*     |

*Auto-jailed if remaining stake falls below `MIN_STAKE`.

### 8.6 Slash Execution

```
slash_amount = (total_stake * rate_bps) / 10000     // integer division

// Deduct from self-stake first
if self_stake >= slash_amount:
    self_stake -= slash_amount
else:
    remainder = slash_amount - self_stake
    self_stake = 0
    // Distribute remainder proportionally across delegators
    for each delegator:
        delegator_slash = (remainder * delegator_stake) / total_delegated_stake
        delegator_stake -= delegator_slash
```

---

## 9. Dispute Resolution

### 9.1 Dispute Types

There are two dispute tracks:

1. **Asset disputes**: Filed against data assets (rights violations, quality issues)
2. **Enforcement cases**: Infringement detection with bounty system

### 9.2 Asset Dispute Flow

```
+-------+     +--------+     +----------+     +----------+
| File  |---->| Open   |---->| Arbitrate|---->| Resolve  |
|       |     |        |     |          |     | (remedy) |
+-------+     +--------+     +----------+     +----------+
                                  |
                                  +--> Dismiss
```

#### 9.2.1 File Dispute

```
DataDispute {
    asset_id:   string       // target asset
    disputant:  string       // filer address
    reason:     string       // description of complaint
}
```

#### 9.2.2 Available Remedies

| Remedy               | Description                                    | Details Required         |
|----------------------|------------------------------------------------|--------------------------|
| `delist`             | Remove asset from marketplace                  | None                     |
| `transfer`           | Transfer ownership to another party            | `{"new_owner": "addr"}` |
| `rights_correction`  | Change the declared rights type                | `{"new_rights_type": "..."}`|
| `share_adjustment`   | Modify co-creator share allocations            | `{"new_shares": [...]}`  |

#### 9.2.3 Dispute Statuses

| Status      | Description                                   |
|-------------|-----------------------------------------------|
| `open`      | Dispute filed, awaiting resolution             |
| `resolved`  | Remedy applied                                 |
| `dismissed` | Dispute rejected as invalid                    |

### 9.3 Enforcement / Bounty System

#### 9.3.1 Evidence Submission

```
Evidence {
    asset_id:          string     // registered asset
    reporter:          string     // reporter's public key
    infringement_type: enum       // unauthorized_distribution | content_tampering | license_violation | attribution_missing
    platform:          string     // e.g. "github", "twitter"
    url:               string     // where the infringement was found
    content_hash:      string     // SHA-256 of found content
    fingerprint:       string     // extracted fingerprint
    similarity_score:  uint32     // 0-10000 basis points
    description:       string
    timestamp:         uint64     // unix timestamp
}
```

**Validation**:
- All required fields must be non-empty
- `similarity_score` must meet the infringement threshold (implementation-defined)
- No duplicate submissions for the same `(asset_id, url)` pair

#### 9.3.2 Enforcement Case Lifecycle

```
PENDING ---[review_case()]--> UNDER_REVIEW
UNDER_REVIEW ---[resolve_case(GUILTY)]--> VERIFIED --> RESOLVED
UNDER_REVIEW ---[resolve_case(INNOCENT)]--> REJECTED --> RESOLVED
UNDER_REVIEW ---[resolve_case(INSUFFICIENT_EVIDENCE)]--> REJECTED --> RESOLVED
```

#### 9.3.3 Verdicts

| Verdict                    | Reporter Effect               | Infringer Effect            |
|---------------------------|-------------------------------|------------------------------|
| `guilty`                   | Bounty paid (see below)       | Damages assessed             |
| `innocent`                 | Reporter slashed 2% of stake  | No action                    |
| `insufficient_evidence`    | Reporter slashed 2% of stake  | No action                    |

#### 9.3.4 Bounty Rewards

Based on assessed severity:

| Severity   | Bounty Rate (bps) | % of Damages |
|------------|-------------------|-------------|
| `low`      | 500               | 5%          |
| `medium`   | 1000              | 10%         |
| `high`     | 2000              | 20%         |
| `critical` | 3000              | 30%         |

Severity assessment:

| Similarity Score | Severity   |
|-----------------|------------|
| >= 9500 bps     | `critical` |
| >= 8000 bps     | `high`     |
| >= 5000 bps     | `medium`   |
| < 5000 bps      | `low`      |

#### 9.3.5 False Report Slashing

- Rate: 200 bps (2%) of reporter's staked amount
- Triggered when verdict is `innocent` or `insufficient_evidence`
- Prevents spam reports

---

## 10. Governance

### 10.1 Proposal Lifecycle

```
+---------+     +--------+     +--------+     +----------+     +----------+
| Propose |---->| ACTIVE |---->| Tally  |---->| PASSED   |---->| EXECUTED |
| (deposit|     | (voting|     |        |     |          |     |          |
|  locked)|     | open)  |     +---+----+     +----------+     +----------+
+---------+     +--------+         |
                                   +--> REJECTED
                                   +--> EXPIRED (no quorum)
```

### 10.2 Proposal Structure

```
Proposal {
    id:               string           // SHA-256(proposer + title + changes + created_at)
    proposer:         string           // address
    title:            string
    description:      string
    changes:          ParameterChange[]
    deposit:          uint64           // uoas, MUST >= MIN_DEPOSIT
    status:           enum             // DEPOSIT | ACTIVE | PASSED | REJECTED | EXECUTED | EXPIRED
    voting_start:     uint64           // block height
    voting_end:       uint64           // block height
    created_at:       uint64           // block height
    snapshot_height:  uint64           // block height at creation (voting power snapshot)
    snapshot_total_stake: uint64       // total stake at snapshot for quorum
}

ParameterChange {
    module:     string     // "consensus", "slashing", "rewards", etc.
    key:        string     // parameter name
    old_value:  any        // current value
    new_value:  any        // proposed value
}
```

### 10.3 Voting

```
Vote {
    proposal_id:  string
    voter:        string     // address
    option:       enum       // YES | NO | ABSTAIN
    weight:       uint64     // stake-weighted voting power (uoas)
    timestamp:    uint64     // block height
}
```

**Rules**:
- One vote per address per proposal (last vote wins)
- Voting power is snapshotted at `snapshot_height` to prevent stake manipulation
- Validators and delegators may both vote

### 10.4 Tally Rules

| Parameter           | Value  | Description                             |
|---------------------|--------|-----------------------------------------|
| Quorum              | 40%    | 4000 bps of total staked OAS must vote  |
| Pass threshold      | 66.67% | 6667 bps of voting power must vote YES  |
| Voting period       | 60,480 blocks | ~7 days at 10s block time         |
| Minimum deposit     | 1,000 OAS | 100,000,000,000 uoas                |

```
participation = (yes_votes + no_votes + abstain_votes) / snapshot_total_stake

if participation < QUORUM_BPS / 10000:
    status = EXPIRED
elif yes_votes / (yes_votes + no_votes) >= PASS_THRESHOLD_BPS / 10000:
    status = PASSED
else:
    status = REJECTED
```

Note: `ABSTAIN` votes count toward quorum but not toward pass/fail ratio.

### 10.5 Execution

Passed proposals are auto-executed. Parameter changes are applied to the
governance parameter registry. The following keys are NOT governable:

- `chain_id`
- `crypto_algorithm`

### 10.6 Deposit Mechanics

- Proposer must deposit >= `MIN_DEPOSIT` at proposal creation
- Deposit is locked for the duration of voting
- If proposal passes: deposit returned to proposer
- If proposal is rejected or expires: deposit is slashed (sent to protocol treasury)

---

## 11. Economic Parameters

### 11.1 Master Parameter Table

| Parameter                     | Value                | Unit    | Governable |
|-------------------------------|----------------------|---------|-----------|
| `OAS_DECIMALS`                | 10^8                 | -       | No        |
| `MIN_STAKE`                   | 10,000 OAS           | uoas    | Yes       |
| `MAX_COMMISSION_BPS`          | 5000                 | bps     | Yes       |
| `UNBONDING_PERIOD`            | 28 days              | days    | Yes       |
| `BLOCKS_PER_EPOCH` (testnet)  | 10                   | blocks  | Yes       |
| `BASE_BLOCK_REWARD`           | 4.0 OAS              | uoas    | Yes       |
| `HALVING_INTERVAL` (testnet)  | 10,000               | blocks  | Yes       |
| `HALVING_INTERVAL` (mainnet)  | 1,000,000            | blocks  | Yes       |
| `PROTOCOL_FEE_BPS`            | 500                  | bps     | Yes       |
| `MIN_PROVIDER_STAKE`          | 100 OAS              | uoas    | Yes       |
| `OFFLINE_SLASH_BPS`           | 100                  | bps     | Yes       |
| `DOUBLE_SIGN_SLASH_BPS`       | 500                  | bps     | Yes       |
| `LOW_QUALITY_SLASH_BPS`       | 50                   | bps     | Yes       |
| `OFFLINE_THRESHOLD`           | 50%                  | %       | Yes       |
| `LOW_QUALITY_THRESHOLD`       | 0.3                  | score   | Yes       |
| `LOW_QUALITY_WINDOW`          | 10                   | tasks   | Yes       |
| `ESCROW_DEFAULT_TTL`          | 300                  | seconds | Yes       |
| `DEMAND_ALPHA`                | 0.1                  | -       | Yes       |
| `SCARCITY_BASE`               | 1.0                  | -       | Yes       |
| `FRESHNESS_HALFLIFE`          | 180                  | days    | Yes       |
| `MIN_PRICE`                   | 0.001                | OAS     | Yes       |
| `CONTRIBUTION_SCORE_WEIGHT`   | 0.5                  | -       | Yes       |
| `FEEDBACK_DECAY_HALFLIFE`     | 30                   | days    | Yes       |
| `VERIFIED_WEIGHT`             | 1.0                  | -       | Yes       |
| `UNVERIFIED_WEIGHT`           | 0.1                  | -       | Yes       |
| `MAX_FEEDBACK_PER_SKILL`      | 200                  | records | Yes       |
| `FALSE_REPORT_SLASH_BPS`      | 200                  | bps     | Yes       |
| `BOUNTY_REWARD_LOW_BPS`       | 500                  | bps     | Yes       |
| `BOUNTY_REWARD_MEDIUM_BPS`    | 1000                 | bps     | Yes       |
| `BOUNTY_REWARD_HIGH_BPS`      | 2000                 | bps     | Yes       |
| `BOUNTY_REWARD_CRITICAL_BPS`  | 3000                 | bps     | Yes       |
| `GOVERNANCE_QUORUM_BPS`       | 4000                 | bps     | Yes       |
| `GOVERNANCE_PASS_BPS`         | 6667                 | bps     | Yes       |
| `GOVERNANCE_VOTING_PERIOD`    | 60,480               | blocks  | Yes       |
| `GOVERNANCE_MIN_DEPOSIT`      | 1,000 OAS            | uoas    | Yes       |
| `OWNER_FEE_BPS`               | 6000                 | bps     | Yes       |
| `VALIDATOR_FEE_BPS`           | 2000                 | bps     | Yes       |
| `PROTOCOL_TREASURY_BPS`       | 1500                 | bps     | Yes       |
| `BURN_BPS`                    | 500                  | bps     | Yes       |

### 11.2 Block Reward Schedule

```
halvings = block_height / HALVING_INTERVAL     // integer division
reward   = BASE_BLOCK_REWARD >> halvings       // right shift = divide by 2^halvings
```

| Halving | Block Range (testnet)   | Reward per Block |
|---------|-------------------------|------------------|
| 0       | 0 -- 9,999              | 4.0 OAS          |
| 1       | 10,000 -- 19,999        | 2.0 OAS          |
| 2       | 20,000 -- 29,999        | 1.0 OAS          |
| 3       | 30,000 -- 39,999        | 0.5 OAS          |
| n       | ...                     | 4.0 / 2^n OAS    |

### 11.3 Reward Distribution at Epoch Boundary

At each epoch boundary:

```
block_reward_total = blocks_proposed * current_block_reward
work_reward_total  = sum(final_value) from settled tasks in epoch

// Validator income
validator_block_income = block_reward_total * commission_rate / 10000
validator_work_income  = work_reward_total * 9000 / 10000          // 90%

// Delegator pool
delegator_block_pool   = block_reward_total - validator_block_income
delegator_work_pool    = work_reward_total * 1000 / 10000          // 10%

// Per-delegator distribution (proportional to delegation)
for each delegator:
    share = delegator_stake / total_delegated_stake
    delegator_reward = (delegator_block_pool + delegator_work_pool) * share
```

### 11.4 Integer Arithmetic

All economic calculations use integer arithmetic with basis points:

```
func apply_rate_bps(amount uint64, rate_bps uint32) uint64 {
    return (amount * uint64(rate_bps)) / 10000
}
```

No floating-point values are permitted in consensus-critical monetary calculations.

---

## 12. Security Tiers

### 12.1 Tier Definitions

Security tiers control access to data assets based on sensitivity and the
accessor's stake/verification level.

| Tier | Name          | Stake Required | Max Data Size | Max Liability  | Description                          |
|------|---------------|---------------|---------------|----------------|--------------------------------------|
| L0   | Public        | 0 OAS         | Unlimited     | None           | Publicly accessible, no restrictions |
| L1   | Basic         | 100 OAS       | 100 MB        | 1,000 OAS      | Basic identity verification          |
| L2   | Verified      | 1,000 OAS     | 1 GB          | 10,000 OAS     | Verified identity, stake collateral  |
| L3   | Trusted       | 10,000 OAS    | 10 GB         | 100,000 OAS    | Full verification, TEE attestation   |

### 12.2 Risk Level to Access Tier Mapping

| Risk Level   | Maximum Access Tier | Description                      |
|-------------|--------------------|------------------------------------|
| `public`     | L3                 | No restrictions on accessor tier   |
| `internal`   | L3                 | No restrictions on accessor tier   |
| `sensitive`  | L2                 | Requires at least L2 accessor      |

### 12.3 Tier Enforcement

When a consumer requests access to a data asset:

```
if consumer_tier < asset.max_access_level:
    reject("insufficient security tier")
```

Where tier comparison uses the ordering: L0 < L1 < L2 < L3.

### 12.4 TEE Attestation (L3)

L3 access requires Trusted Execution Environment attestation:
- Compute runs inside a secure enclave
- Zero-knowledge Proof of Execution (zk-PoE) verifies correct execution
  without revealing input/output data
- TEE attestation certificates are validated on-chain

---

## 13. Diminishing Returns (Share Minting)

### 13.1 Formula

When buyers purchase access to a data asset, they receive shares with
diminishing returns to reward early participants:

| Purchase Order | Share Rate | Shares Received               |
|---------------|-----------|-------------------------------|
| 1st buyer     | 100%      | `payment * 10000 / 10000`     |
| 2nd buyer     | 80%       | `payment * 8000 / 10000`      |
| 3rd buyer     | 60%       | `payment * 6000 / 10000`      |
| 4th+ buyer    | 40%       | `payment * 4000 / 10000`      |

### 13.2 Implementation

```
func share_rate_bps(buyer_index uint32) uint32 {
    switch {
    case buyer_index == 0:
        return 10000     // 100%
    case buyer_index == 1:
        return 8000      // 80%
    case buyer_index == 2:
        return 6000      // 60%
    default:
        return 4000      // 40%
    }
}

func mint_shares(payment uint64, buyer_index uint32) uint64 {
    return (payment * uint64(share_rate_bps(buyer_index))) / 10000
}
```

### 13.3 Economic Rationale

- Early supporters of valuable data are rewarded with proportionally more
  ownership
- Later buyers still participate in the asset's economics (40% floor)
- The decreasing rate creates a natural incentive for early price discovery
- Share ownership is recorded on-chain and confers proportional revenue rights

---

## 14. Block and Network Wire Formats

### 14.1 Block Header

```
BlockHeader {
    chain_id:     string     // chain identifier
    block_number: uint64     // monotonically increasing
    prev_hash:    string     // SHA-256 hex of parent block
    merkle_root:  string     // Merkle root of operations
    timestamp:    uint64     // unix timestamp
    proposer:     string     // proposer's validator ID
    signature:    string     // Ed25519 signature (hex)
}
```

### 14.2 Block Hash

Deterministic block hash computation:

```
block_hash = SHA256(chain_id || block_number || prev_hash || merkle_root || timestamp)
```

Where `||` denotes string concatenation of the string representations.

### 14.3 Block

```
Block {
    // All BlockHeader fields, plus:
    operations:   Operation[]    // ordered list of operations in this block
}
```

### 14.4 Epoch and Slot Scheduling

All timing is derived from block height (deterministic, no wall-clock
dependency for consensus):

```
epoch = block_height / blocks_per_epoch     // integer division
slot  = block_height % blocks_per_epoch
```

Key derived values:
- `epoch_start_block(epoch) = epoch * blocks_per_epoch`
- `epoch_end_block(epoch) = (epoch + 1) * blocks_per_epoch - 1`
- `is_epoch_boundary(height) = (height + 1) % blocks_per_epoch == 0`

### 14.5 Sync Protocol Messages

#### GetHeight Request

```json
{ "type": "get_height" }
```

#### HeightResponse

```json
{
    "height": 12345,
    "best_hash": "abc123...",
    "chain_id": "oasyce-mainnet-1",
    "timestamp": 1711000000
}
```

#### GetBlocks Request

```json
{
    "from_height": 100,
    "to_height": 200,
    "count": 100
}
```

#### GetBlocks Response

```json
{
    "blocks": [ <Block>... ],
    "status": "ok"
}
```

### 14.6 Fork Choice Rule

1. **Longest chain**: The chain with the most blocks is canonical
2. **Stake-weighted tiebreaker**: On equal length, the chain whose tip was
   proposed by the validator with more stake wins
3. **Max reorg depth**: Reorgs beyond a configurable depth limit are rejected
4. **Rollback**: Event-sourced design allows safe rollback by replaying events
   up to a prior block

### 14.7 Cryptographic Primitives

| Purpose               | Algorithm              | Key Size  |
|-----------------------|------------------------|-----------|
| Signatures            | Ed25519                | 256-bit   |
| Content hashing       | SHA-256                | 256-bit   |
| Block hashing         | SHA-256                | 256-bit   |
| API key encryption    | AES-256-GCM (PBKDF2)  | 256-bit   |
| Key derivation        | PBKDF2-SHA256          | 100k iter |

### 14.8 Serialization

All wire messages are JSON-encoded for the HTTP JSON transport layer. Nodes
communicate via HTTP on the sync port (default 9528). The canonical
serialization for signature verification is JSON with sorted keys.

---

## Appendix A: Basis Points Reference

Throughout this specification, rates and percentages are expressed in basis
points (bps) for integer precision:

| bps   | Percentage | Fraction   |
|-------|-----------|------------|
| 1     | 0.01%     | 1/10000    |
| 50    | 0.5%      | 50/10000   |
| 100   | 1%        | 100/10000  |
| 500   | 5%        | 500/10000  |
| 1000  | 10%       | 1000/10000 |
| 4000  | 40%       | 4000/10000 |
| 5000  | 50%       | 5000/10000 |
| 6667  | 66.67%    | 6667/10000 |
| 10000 | 100%      | 10000/10000|

## Appendix B: ID Format Reference

| Entity         | Format                                     | Example                            |
|----------------|--------------------------------------------|------------------------------------|
| Escrow         | `ESC_` + hex(UUID)[:16]                    | `ESC_A1B2C3D4E5F60718`            |
| Invocation     | `INV_` + hex(UUID)[:16]                    | `INV_F0E1D2C3B4A59687`            |
| Capability     | `CAP_` + hex(SHA256(content))[:16]         | `CAP_1234ABCD5678EF90`            |
| Proposal       | SHA-256(proposer + title + changes + time) | `a1b2c3d4e5f6...` (64 hex chars)  |
| Dispute        | SHA-256(asset + reporter + url + time)[:16]| `f0e1d2c3b4a59687`               |
| Enforcement    | SHA-256(`case:` + dispute_id)[:16]         | `1a2b3c4d5e6f7890`               |

## Appendix C: Complete State Machine Summary

### Validator Lifecycle

```
           REGISTER
              |
              v
          +--------+
          | ACTIVE |<----------+
          +---+----+           |
              |                |
         SLASH (jail)     UNJAIL
              |                |
              v                |
          +--------+           |
          | JAILED |-----------+
          +---+----+
              |
             EXIT
              |
              v
          +--------+
          | EXITED |  (terminal, after unbonding)
          +--------+
```

### Escrow Lifecycle

```
          LOCK
           |
           v
       +--------+
       | LOCKED |
       +---+----+
           |
     +-----+-----+-----+
     |     |           |
  release refund    expire
     |     |           |
     v     v           v
 RELEASED REFUNDED  EXPIRED
```

### Governance Proposal Lifecycle

```
   PROPOSE (deposit)
        |
        v
    +--------+
    | ACTIVE |
    +---+----+
        |
      TALLY
        |
   +----+----+-----+
   |         |     |
   v         v     v
 PASSED  REJECTED EXPIRED
   |
   v
 EXECUTED
```

### Enforcement Case Lifecycle

```
  SUBMIT EVIDENCE
        |
        v
    +---------+
    | PENDING |
    +----+----+
         |
      REVIEW
         |
         v
  +--------------+
  | UNDER_REVIEW |
  +------+-------+
         |
     RESOLVE
         |
    +----+----+
    |         |
    v         v
 VERIFIED  REJECTED
    |         |
    v         v
 RESOLVED  RESOLVED
```
