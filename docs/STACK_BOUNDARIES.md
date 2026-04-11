# Oasyce Stack Boundaries

This document freezes the chain-side architecture boundary for the pre-mainnet period.

Use it when a feature request, script, or doc makes it unclear which layer owns what.

## Stack Roles

- `Sigil / Oasyce-Sigil`: protocol constitution, object model, tiering, and long-term direction.
- `Oasyce Chain`: lifecycle ledger, authorization truth, commitments, settlement, and public finality.
- `oasyce-sdk`: AI-facing front door, local binding, native signer, delegate runtime, and chain bridge.
- `Thronglets`: shared environment, presence, signal, trace, peer discovery, and collaboration continuity.
- `Psyche`: subjective continuity, relation residue, reply bias, and behavior-shaping control surfaces.

## Chain Owns

- facts that must be public, durable, auditable, and final
- lifecycle events that need immutable ordering
- authorization boundaries that need public truth
- escrow, settlement, anchoring, and chain-level economic state
- validator, genesis, upgrade, and network-readiness concerns

## Chain Does Not Own

- device `join` / `share` flows
- handoff artifacts or local setup UX
- session or presence tracking
- shared discovery or agent coordination
- direct messaging
- emotional state, subjectivity, or reply control
- dashboard / local product front-door flows
- agent workflow orchestration

## Layer Anti-Goals

- `Chain` is not the high-frequency runtime and not the default product front door.
- `SDK` is not the final truth source for authorization or settlement.
- `Thronglets` is not the final authorization judge or settlement ledger.
- `Psyche` is not the authorization layer, wallet layer, or public memory layer.

## Ownership Gate

Before adding a new chain feature, answer these questions in order:

1. Is this a fact that must be public, durable, auditable, and finally adjudicable?
2. Is this primarily about how an AI starts, signs, recovers, or connects on a device?
3. Is this primarily about multi-agent discovery, coordination, signal, or shared environment state?
4. Is this primarily about how interaction history changes later behavior distribution?

Default routing:

- If `1` is yes: `Chain`
- Else if `2` is yes: `oasyce-sdk`
- Else if `3` is yes: `Thronglets`
- Else if `4` is yes: `Psyche`

If a request spans multiple layers, split it by concern:

- `Chain` records public fact
- `SDK` executes
- `Thronglets` coordinates
- `Psyche` changes subjective bias

## Pre-Mainnet Chain Scope

For the chain repo, pre-mainnet work should stay inside three buckets:

- `truth-critical`: `x/sigil`, `x/anchor`, `x/onboarding`, `x/settlement`, `x/delegate`
- `launch-surface compatibility`: `x/capability`, `x/reputation`, `x/work`
- `network readiness`: validators, genesis, upgrades, monitoring, recovery, and soak testing

The canonical AI write path is outside this repo's long-term ownership:

`Wallet.auto()` -> `OasyceClient` -> `NativeSigner` -> delegate policy

Chain-side scripts may exist as compatibility wrappers or test harnesses, but they should route through the SDK-native path instead of maintaining their own signer/runtime stack.

## Sigil Pulse: V1 vs V2 Trajectory

V1 (current, post-v0.8.0):

- `MsgPulse` is **owner-signed only**. The sigil creator is the sole authority for pulses on their own sigil.
- Dormancy is **one-way**. Pulses on dormant/dissolved sigils are rejected; revival requires a fresh `MsgGenesis`.
- The chain does not interpret dimension names — it only reads `MaxPulseHeight` across all dimensions for liveness decay.
- This is a deliberate simplification to ship a safe, abuse-resistant primitive before the economic model for field-driven pulses is settled.

V2 (direction, not yet in scope):

- Pulses should be **field-driven**. External activity that already carries economic weight — another sigil entering a `Bond` with this one, an `x/anchor` trace citing this sigil, a shared-environment event naming this sigil — should implicitly contribute a heartbeat.
- This requires: (1) a spam-cost model so implicit pulses can't be used as free ping attacks, (2) attribution rules for who pays the pulse gas when the signer is not the sigil owner, and (3) alignment with the Sigil constitution's "field is the subject, individuals are synapses" stance.
- Until these are resolved, do **not** relax the `msg.Signer == sigil.Creator` check in `x/sigil/keeper/msg_server.go::Pulse`. The owner-lock is load-bearing for V1 safety.

Anti-goal for the V1 → V2 transition: do **not** add a `MsgRevive` that walks dormant → active. Dormant is committed pruning; if an agent needs to come back, it creates a new Sigil and can record lineage via `MsgFork`. This keeps the state machine monotonic and the pruning metaphor honest.
