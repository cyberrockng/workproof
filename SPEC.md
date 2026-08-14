# WorkProof Specification

Status: Phase 1 freeze draft
Network target: Coston2
Primary asset: official FTestXRP resolved from Flare registry

WorkProof is a pre-funded payment workflow for objective smart-contract deliverables. A client locks principal and a success fee, a named contractor submits a deployed Coston2 contract, Flare Confidential Compute verifies private examples of public requirements, and the escrow releases principal only after an authentic PASS result signed by the expected TEE machine.

## P0 Scope

P0 supports exactly one objective milestone per on-chain job:

- named client and contractor addresses;
- full principal pre-funding in official Coston2 FTestXRP;
- separate success fee locked by the client and paid only on PASS;
- public requirement template plus private encrypted vector bundle;
- deployed Coston2 contract artifact;
- immutable artifact address, submission block, and runtime code hash;
- future secure Flare random round selection;
- FCC Go verifier on one intended production TEE;
- contract-verified native TEE ActionResult settlement;
- PASS, FAIL, INCONCLUSIVE, timeout, cancellation-before-acceptance, and expiry/refund paths.

P0 does not judge subjective quality, execute arbitrary native binaries, adjudicate IP ownership, or claim mainnet custody of valuable funds.

## Job Terms

Each job freezes these values before contractor acceptance:

- client;
- contractor;
- payment token;
- contractor principal;
- locked success fee;
- acceptance deadline;
- submission deadline;
- verification grace period;
- public requirement template ID and version;
- public spec hash;
- private bundle hash;
- ciphertext hash and content-addressed locator;
- target chain ID;
- artifact type;
- expected TEE ID;
- FCC extension ID;
- verifier version hash;
- retry policy.

Existing job economics are immutable. Fee changes apply only to future jobs.

## State Machine

```text
CREATED/FUNDED
  | client cancels before acceptance
  v
CANCELLED -> client refund

CREATED/FUNDED
  | contractor accepts
  v
ACCEPTED / AWAITING_SUBMISSION
  | contractor submits artifact before submitBy
  v
RANDOMNESS_PENDING
  | secure future round locked
  v
READY_TO_VERIFY
  | instruction dispatched
  v
VERIFYING
  | PASS
  v
PAID

VERIFYING
  | FAIL
  v
AWAITING_RESUBMISSION

VERIFYING
  | INCONCLUSIVE or timeout
  v
RETRYABLE

AWAITING_RESUBMISSION or RETRYABLE
  | contractor resubmits before submitBy
  v
RANDOMNESS_PENDING

Any non-paid accepted job after submitBy plus verification grace
  -> REFUNDED
```

Terminal states are `CANCELLED`, `PAID`, and `REFUNDED`.

## Outcome Rules

| Event | Principal | Success fee | Next state |
|---|---|---|---|
| Client cancels before contractor acceptance | Client | Client | CANCELLED |
| Authentic PASS from expected TEE | Contractor | Treasury | PAID |
| FAIL before deadline | Locked | Locked | AWAITING_RESUBMISSION |
| INCONCLUSIVE or infrastructure error | Locked | Locked | RETRYABLE |
| Timely submission still verifying at deadline | Locked during grace | Locked | VERIFYING |
| No valid PASS by grace end | Client | Client | REFUNDED |
| Relayer disappears | Locked | Locked | Anyone can relay |
| TEE/proxy disappears | Locked | Locked | Timeout then retry/refund |

## P0 Test Vector Types

Private vectors are data only. They are private examples of public rules, not secret rules.

- `ETH_CALL_EQUALS`: calldata, caller/value, and expected return bytes.
- `ETH_CALL_REVERTS`: calldata and expected revert selector or bounded pattern.
- `ERC165_SUPPORTS_INTERFACE`: interface ID and expected support result.
- `CODE_SIZE_RANGE`: inclusive runtime bytecode size bounds.
- `STORAGE_AT_EQUALS`: optional fixed-slot assertion at the submission block.

The template ID, template version, input domains, selection count, threshold, gas cap, timeout, and maximum response bytes are public.

## Invariants

- Accounted FTestXRP balance equals locked principal plus locked fees.
- A job pays at most once.
- Principal leaves only to the contractor after PASS or to the client after cancellation/refund.
- No owner/admin function can mark PASS, replace a verdict, redirect principal, or change accepted terms.
- An attempt result is accepted at most once.
- FAIL and INCONCLUSIVE never transfer principal.
- Fee changes do not affect existing jobs.
- Pausing stops new jobs and attempts, not refunds or valid already-dispatched settlement.
- Stale, wrong, unregistered, malformed, or replayed TEE results cannot release funds.
- Total outgoing principal never exceeds total funded principal.

## VerdictV1 Binding

The client supplies `expectedTee` to `createJob` after reading the TEE public
key and before encrypting the hidden bundle. The escrow verifies that explicit
TEE is currently PRODUCTION, stores it with the job, dispatches only to it, and
settles only signatures recovered from it.

The FCC extension ABI-encodes `VerdictV1` into signed `ActionResult.data`. Every security-critical field is repeated inside the signed verdict and checked against escrow storage:

- schema version;
- escrow address;
- chain ID;
- job ID;
- attempt;
- instruction ID;
- public spec hash;
- private bundle hash;
- artifact address;
- artifact block;
- artifact runtime code hash;
- random voting round;
- random value hash;
- engine version hash;
- outcome;
- passed count;
- executed count;
- report hash;
- issued at;
- expires at.

The escrow rejects semantically inconsistent counts: at least one vector must be
represented, `passedCount` cannot exceed `executedCount`, PASS requires
`passedCount == executedCount`, and FAIL/INCONCLUSIVE require
`passedCount < executedCount`.

The public report exposes vector IDs, statuses, timing, and redacted diagnostics. It must not expose hidden inputs or expected outputs.
