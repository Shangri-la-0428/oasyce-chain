# Token Economics

> Oasyce Chain — Protocol Economics & Incentive Design

## Token

| Property | Value |
|----------|-------|
| Name     | OAS   |
| Smallest unit | uoas (1 OAS = 1,000,000 uoas) |
| Supply model | Deflationary (2% burn on settlement) |

## Bonding Curve

Oasyce uses a **Bancor continuous bonding curve** for data asset pricing. Unlike fixed pricing, the curve automatically adjusts price based on supply and demand.

### Parameters

| Constant | Value | Description |
|----------|-------|-------------|
| Reserve Ratio (CW) | 0.5 | Connector weight — controls curve steepness |
| Initial Price | 1 uoas | Bootstrap price when reserve = 0 |
| Reserve Solvency Cap | 95% | Max reserve withdrawable on sell |

### Buy Formula

```
Bootstrap (reserve = 0):
  tokens = payment / 1 uoas

Active Curve (CW = 0.5):
  tokens = supply × (√(1 + payment/reserve) − 1)
```

### Sell Formula (Inverse Curve)

```
payout = reserve × (1 − (1 − tokens/supply)²)

Constraints:
  payout ≤ reserve × 0.95  (solvency cap)
  fee = max(payout × 5%, 1 uoas)  (minimum fee guard)
  net_payout = payout − fee
```

### Price Behavior

```
Price
  │        ╱
  │       ╱
  │      ╱       ← price rises with each purchase
  │     ╱
  │    ╱
  │   ╱
  │  ╱
  │ ╱
  │╱_______________ Supply
```

- **Early buyers** get more tokens per uoas (lower price)
- **Price increases** with each purchase (more demand → higher price)
- **Selling** reduces supply, lowering price for future buyers
- **No rug pull**: 95% solvency cap ensures reserve always has liquidity

## Fee Structure

### Escrow Release (Capability Settlement)

When an escrow is released upon successful invocation:

```
Total Payment
  ├── 93% → Provider (AI capability owner)
  ├──  5% → Protocol (fee_collector module account)
  └──  2% → Burned 🔥 (permanently destroyed)
```

### Share Sell Fee

When a shareholder sells tokens back to the curve:

```
Gross Payout (from inverse Bancor)
  ├── 95% → Seller
  └──  5% → Protocol (fee_collector)
```

Minimum fee: 1 uoas (prevents zero-fee on tiny amounts).

## Access Gating

Equity ownership in a data asset grants tiered access:

| Level | Name    | Min Equity | Description |
|-------|---------|------------|-------------|
| L0    | Query   | ≥ 0.1%    | Read metadata, basic queries |
| L1    | Sample  | ≥ 1%     | Access data samples |
| L2    | Compute | ≥ 5%     | Run computations on data |
| L3    | Deliver | ≥ 10%    | Full data delivery |

### Reputation Cap

High equity alone isn't enough — reputation gates access:

| Reputation Score | Max Access Level |
|-----------------|-----------------|
| R < 20          | L0 only |
| 20 ≤ R < 50    | L1 max |
| R ≥ 50          | All levels |

This prevents new accounts from buying 10% and immediately accessing full data.

## Dispute Economics

### Filing

- **Deposit**: configurable (default 1000 OAS for testnet)
- Deposit locked in module account during dispute

### Resolution

| Outcome | Plaintiff Deposit | Provider Reputation | Consumer Reputation |
|---------|-------------------|--------------------|--------------------|
| Upheld (plaintiff wins) | Returned | −10 | unchanged |
| Rejected (defendant wins) | Forfeited | unchanged | −5 |

### Jury Incentives

| Juror Vote | Outcome |
|------------|---------|
| Voted with majority | +1 reputation |
| Voted against majority | −2 reputation |

- **Jury size**: 5 jurors
- **Threshold**: 2/3 majority (≥ 4 of 5 must agree)
- **Selection**: deterministic scoring `sha256(disputeID + nodeID) × ln(1 + reputation)`

## Reputation Scoring

```
score = Σ(weight_i × rating_i) / Σ(weight_i)

weight = decay × credibility
  decay = exp(−0.693 × age_days / 30)  [half-life: 30 days]
  credibility = 2.0 (verified) | 0.5 (unverified)

Rating range: 0 − 500
```

- **Verified feedback** (from actual invocation consumers) counts 4x more than unverified
- **Recent feedback** matters more — scores naturally decay toward 0 without activity
- **Cooldown**: 1 hour between same submitter→target pair

## Deflationary Pressure

Every capability settlement burns 2% of the payment. As network usage grows:

```
Annual burn = Total settlement volume × 2%
```

There is no minting mechanism for OAS beyond initial genesis allocation, making the token supply strictly decreasing over time.

## Summary

| Mechanism | Purpose |
|-----------|---------|
| Bancor curve | Fair, liquid pricing without order books |
| 2% burn | Deflationary pressure, align long-term value |
| 5% protocol fee | Sustainable protocol development |
| Access gating | Incentivize equity ownership |
| Reputation cap | Prevent access-by-wealth without track record |
| Jury deposit | Discourage frivolous disputes |
| Reputation decay | Keep scores relevant, reward active participants |
