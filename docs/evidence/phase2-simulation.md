# Phase 2 simulation lab

This is a pre-build validation artifact. It does not claim that the escrow is
ready for Coston2 custody or that the FCC machine is registered.

## Mandatory FCC signature-compatibility spike (plan section 9)

`contracts/lib/FccVerdict.sol` reconstructs the real FCC `ActionResult`
signing chain — `keccak256(keccak256(data) || id || keccak256(submissionTag)
|| status)` → `keccak256(abi.encode(bytes32("TEE_ACTION_RESULT"), chainId,
actionResultHash))` → EIP-191 wrap → canonical (low-S) ECDSA recovery —
copied from the pinned `tee-node` v0.0.24 (`pkg/types/actions.go` `Hash()`,
`internal/router/utils.go` `SignResult`, `pkg/utils/crypto.go` `Sign`) and
`go-flare-common` v1.2.2-...-09a10067e6a4 (`pkg/signing/hash.go`
`Payload.Hash()`) source, not guessed.

Proven two ways:

- `test/FccSignatureSpike.t.sol` reconstructs the digest and recovers the
  signer against a **real external Go-signed vector**
  (`docs/evidence/fcc-signature-spike-v1.json`, produced by
  `go/cmd/fcc-spike` using only real go-ethereum/go-flare-common/tee-node
  exported code — never a reimplementation of the crypto), including a
  cross-chain replay check and per-field mutation rejection.
- `test/WorkProofEscrow.t.sol` builds and signs verdicts dynamically with
  `vm.sign` against a locally generated TEE key, exercising every mutation
  path against the live contract rather than one fixed fixture.

Gate result: **green.** Both proof styles recover the correct signer and
reject every mutation tried (wrong signer, malformed signature, wrong chain,
wrong job/attempt/instruction, replay, high-S malleated signature).

One real bug was caught and fixed during this spike: `FccVerdict.sol`'s
`SECP256K1_HALF_N` constant was originally missing a trailing hex digit (63
digits instead of 64), making it ~1/16th of the real secp256k1 order and
rejecting nearly every valid signature, not just malleable ones. Caught by a
raw-`ecrecover` isolation test before it reached the shipped contract.

## VerdictV1 binding

`contracts/WorkProofEscrow.sol`'s `settle()` now ABI-decodes the full
`VerdictV1` (mirrors `packages/schema/schemas/verdict-v1.schema.json`
field-for-field, split into two nested static sub-structs —
`VerdictIdentity`/`VerdictOutcome` — purely to stay under solc's IR
stack-depth limit for a 20-field decode; both sub-structs are static/value
types only, so the ABI wire encoding is byte-identical to a flat tuple and
off-chain encoders don't need to know about the split) and checks every
security-critical field against job storage before any transfer: schema
version, escrow address, chain ID, job ID, attempt, instruction ID, spec
hash, private bundle hash, artifact address/block/code hash, random
round/value hash, engine version hash, and a non-zero report hash.

## Implemented checks (`test/WorkProofEscrow.t.sol`, all 10 Phase 2 required
simulations plus the accounting invariant)

1. Failing artifact returns FAIL; no balance moves.
2. Corrected resubmission; PASS transfers exactly the principal.
3. Bundle-hash mutation fails commitment verification.
4. Artifact/code-hash mutation fails.
5. Wrong signer, malformed signature, wrong chain, wrong job, wrong attempt,
   wrong instruction, and replay all fail (7 sub-cases).
6. Insecure randomness (`lockRandomness`) advances only one deterministic
   round per attempt; a second lock reverts.
7. INCONCLUSIVE moves no principal; submission after the deadline reverts.
8. Expiry (`refund`) returns exactly the locked principal and fee to the
   client only.
9. Paused mode blocks new `accept`/`submit`/`lockRandomness` but still
   permits `refund` and an already-dispatched valid `settle`.
10. A fuzzed accounting invariant (256 runs) over random principal/fee/
    outcome combinations: PASS moves exactly `principal` to the contractor
    and nothing to the client; FAIL moves nothing.

A meta-check was run to confirm the mutation tests are load-bearing (not
just always-reverting for unrelated reasons): temporarily disabling the
`artifactCodeHash` binding check made `testSimulation4_ArtifactMutationRejected`
fail as expected, then the check was restored and the full suite re-verified
green.

The Go model is under `go/internal/workproof`;
`docs/evidence/action-result-digest-v1.json` pins the plain digest vector,
`docs/evidence/fcc-signature-spike-v1.json` pins the signed vector.

## Still open before this can be called a finished Phase 2 lab

- No handler-based `forge invariant` suite (the plan's exit-condition
  command list names `WorkProofInvariantTest` specifically); the fuzz test
  above covers the same accounting property for a single job but not a
  multi-job, multi-call-sequence invariant harness.
- `SIMULATION_ONLY` labeling/banner conventions from the plan (fixtures
  under `testdata/simulation-only/`, a `SIMULATION_ONLY=true` flag, a red UI
  banner) are not yet wired up — there is no UI yet to wire them into.
- Registered FCC machine/extension registry checks are explicitly Phase 3
  scope, not Phase 2; `expectedTee` here is a constructor-time pinned
  address, not a live `TeeMachineRegistry`/`TeeExtensionRegistry` lookup.
