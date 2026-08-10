# Phase 3 production smart contracts

This documents `contracts/WorkProofEscrow.sol` as it stands after Phase 3
corrections. It supersedes the Phase 2 lab's simplifications (see
`docs/evidence/phase2-simulation.md`) but does not itself constitute a
completed security audit.

## What changed from the Phase 2 lab, and why

| Phase 2 lab | Phase 3 production | Why |
|---|---|---|
| Constructor took an arbitrary `IERC20Minimal token` | FTestXRP resolved live via `ContractRegistry.getAssetManagerFXRP().fAsset()` | Plan section 10: never paste a protocol address into business logic; independently confirmed live (`0x0b6A3645c240605887a5532109323A3E12273dc7`), matching what the sibling Ajose project found independently |
| `expectedTee` was a per-attempt field, re-selected at `dispatchVerification` by calling `getRandomTeeIds` fresh each attempt | `expectedTee` is selected and pinned once, in `createJob` | The client must know which TEE's public key to encrypt the private bundle to at job creation; re-selecting at dispatch would silently swap the TEE mid-job, which the plan's threat model forbids |
| `instructionId` was locally invented (`keccak256(...)`) | Real `instructionId` returned by `ITeeExtensionRegistry.sendInstructions` | The lab never actually dispatched through the real registry; production must |
| `treasury = address(uint160(uint256(keccak256("WORKPROOF_TREASURY"))))` | `treasury` is a validated, non-zero constructor parameter | The hash-derived placeholder is a real bug: nobody holds that private key, so a job's success fee would be permanently unrecoverable |
| No reentrancy guards | `nonReentrant` on every fund-moving or external-call-making function | `dispatchVerification` makes a payable external call before any state write; a malicious/reentrant registry could otherwise double-dispatch |
| `settle()` checked `expiresAt < block.timestamp` only | `_checkOutcomeBinding` requires `expiresAt == j.terms.graceEnds` exactly, plus `issuedAt` bounded between dispatch and now | A TEE (or a compromised one) must not be able to claim a later expiry than what it was actually dispatched with |
| No dispatch timeout check in `settle()` | `settleAttempt` rejects once `block.timestamp > j.current.timeoutAt`, even without an explicit `expireVerification()` call first | A timed-out instruction must never settle |
| Hand-rolled bool-return token transfers | OpenZeppelin `SafeERC20` (real pinned dependency) | Consistent, audited failure handling |
| Per-job `paused` flag | Global `paused`, blocking only `createJob`/`submitAttempt` | Matches the plan's required-functions table (`pauseNewWork()` is described as a single global switch) and liveness invariant — every other function stays permissionless during pause |

## Randomness: real historical-random-call pattern

`submitAttempt` commits `targetRound = RELAY.getVotingRoundId(block.timestamp) + 1`
(a future round, so neither party can grind a favorable seed after seeing the
artifact). `lockRandomness` then calls
`RANDOM_NUMBER_V2.getRandomNumberHistorical(targetRound)`:

- a revert (round not yet finalized) is caught and re-thrown as the clean
  `RandomNotReady()` error, leaving storage untouched;
- `isSecure == false` emits `InsecureRandomRoundSkipped` and advances
  `targetRound` by exactly one — deterministic, not caller-chosen;
- only a secure result is locked.

`getVotingRoundId` is resolved via the dedicated `"Relay"` registry entry;
`getRandomNumberHistorical` via the dedicated `"RandomNumberV2"` entry. On
live Coston2 today both resolve to the same address
(`0xa10B672D1c62e5457b17af63d4302add6A99d7dE`, confirmed live via `cast call`
against the real `FlareContractRegistry`), but the two concerns are resolved
separately so the code stays correct if Flare ever splits them.

## TEE selection and pinning

`createJob` calls `TEE_MACHINE_REGISTRY.getRandomTeeIds(extensionId, 1)`,
then confirms the returned machine is live `PRODUCTION` status via
`getTeeMachineStatus` before pinning it into `JobTerms.expectedTee`
(immutable for the life of the job). `dispatchVerification` re-confirms the
*same* pinned machine is still `PRODUCTION` — it never calls
`getRandomTeeIds` again, so a later "random" registry result can never
silently swap the TEE out from under an in-flight job.

`getTeeMachineStatus` is not part of the scaffold's original minimal
`ITeeMachineRegistry` interface; it was added to
`contracts/interfaces/ITeeMachineRegistry.sol` after independently
re-confirming it live against the real FlareTeeManager diamond
(`0x1a9C4A0f9D76c0b1D91d22E24E573a9b377618aE`): calling it with a bogus
address reverts with the distinct custom-error selector `0xceb05b68`,
different from the diamond's generic "function not found" revert for a
truly nonexistent selector — proof the interface matches real deployed
bytecode, not a guess. See `test/RealRegistryFork.t.sol`.

## Test suites

Two suites, deliberately kept separate:

- `test/WorkProofEscrow.t.sol` — 85 tests (80 as of the initial Phase 3 pass;
  5 more added during post-audit remediation, see
  `NEW_WORK.md` "Post-audit remediation"), fully local and deterministic.
  Since the constructor unconditionally resolves the real hardcoded
  `FlareContractRegistry` address, local tests install a mock registry there
  via `vm.etch` before deploying (`test/mocks/Mocks.sol`), rather than
  weakening the production contract to make it easier to unit test. Covers:
  every state transition and custom error; `acceptBy`/`submitBy`/
  `graceEnds`/dispatch-timeout boundaries, including the *exact* boundary
  tick (not just one-past-deadline) for `acceptBy`/`submitBy`/`graceEnds` --
  an independent audit correctly found the original test suite only proved
  the strictly-after side of each deadline, never that the exact boundary
  timestamp itself still succeeds; fixed by adding
  `testAcceptJobSucceedsExactlyAtAcceptByBoundary`,
  `testSubmitAttemptSucceedsExactlyAtSubmitByBoundary`, and
  `testSettleSucceedsExactlyAtGraceEndsBoundary`; not-ready / insecure / secure
  randomness (including a chained insecure-insecure-secure sequence);
  zero and non-`PRODUCTION` TEE registry responses; TEE pinning surviving a
  registry "random pick" change after creation; reentrancy probes (a
  malicious token and a malicious extension registry, each proving the
  *outer* call still succeeds exactly once while the *inner* reentrant
  attempt is blocked); malformed ABI data and malformed/malleable
  signatures; every `VerdictV1` field mutation (spec/bundle/artifact/
  code hash/random round/random hash/engine version/report hash/expiry/
  issuedAt/job id/attempt/instruction id/chain id/escrow address/schema
  version); replay; a stale instruction surviving `expireVerification` and
  resubmission; PASS/FAIL/INCONCLUSIVE; 6-decimal fee rounding (fuzzed);
  pause liveness (blocks only new jobs/attempts, never progress on an
  existing one); owner-authority boundaries; and a fuzzed solvency/
  single-payout/correct-recipient invariant.
- `test/RealRegistryFork.t.sol` — 7 tests against a live Coston2 fork
  (`vm.createSelectFork("coston2")`). Proves construction against the real
  `FlareTeeManager` diamond, real FTestXRP/Relay/RandomNumberV2 resolution,
  a real FXRP holder funding a client, `getTeeMachineStatus`/
  `nextPublicExtensionId` being real live selectors, and — honestly, not
  hidden — that `setExtensionId()`/`createJob()` fail the way a genuinely
  unregistered deployment should while Phase 0's external registration
  remains blocked, rather than crashing on a wrong-ABI decode.

## Real bugs caught while building this (not hidden, not just claimed)

1. `SECP256K1_HALF_N` in `contracts/lib/FccVerdict.sol` was missing a
   trailing hex digit (Phase 2), rejecting ~15/16 of all valid signatures.
2. A test-harness chain-ID mismatch (Phase 2) made every "happy path"
   settle fail *and* made every mutation-rejection test pass for the wrong
   reason simultaneously.
3. `vm.expectRevert()` catching an inline `escrow.OP_TYPE()`/
   `_matchingVerdict()` staticcall used as a settlement argument, instead of
   the intended call — affected ~20 Phase 3 tests at once until every
   `OP_TYPE`/`OP_COMMAND` reference was hoisted into a value cached once in
   `setUp()`.
4. `_createJob`'s default `submitBy`/`graceEnds` window was narrower than
   `VERIFICATION_TIMEOUT`, so a dispatch → timeout → resubmit test cycle
   could warp past `submitBy` and hit the wrong revert reason.
5. A pause-liveness test paused the contract *before* creating a second job
   the test still needed — correctly hit `IsPaused()`, which was never the
   property under test.
6. `testReplayRejected` expected `InvalidVerdict` but a settled job's state
   already leaves `Verifying` after any outcome, so replay is actually
   caught by the earlier state guard (`InvalidState`). Following this
   through, the `j.settled` checks in `settleAttempt` and `cancelUnaccepted`
   turned out to be provably unreachable given the state machine (`settled`
   only ever becomes `true` in the same statement that also moves state
   away from the state each check was guarding) — removed as dead code
   rather than left as inert "defense in depth", which also improved the
   bytecode margin.
7. `issuedAt = 1` in a mutation test wasn't actually "before dispatch" on a
   local Foundry chain, whose `block.timestamp` starts at exactly `1` —
   confirmed via `console.log`, fixed by warping forward first.

## Post-audit corrections (2026-08-09)

An independent audit reviewed the state of the repository after Phase 4 and
found four release-blocking defects plus several high-severity gaps. Every
finding checked against source was accurate — full list and fixes in
`NEW_WORK.md` "Post-audit remediation". The ones specific to what this
document claims about `WorkProofEscrow.sol` itself:

- **C3** (a real Solidity bug, not just a documentation gap): `settleAttempt`
  only checked the dispatch timeout, never `graceEnds` — a late Pass could
  pay the contractor after the client's refund deadline had passed,
  contradicting the economic invariant "the client is entitled to a refund
  after `graceEnds`." Fixed with an explicit guard; see the updated test
  count/coverage above and `NEW_WORK.md`.
- This document's "exact ... boundaries" claim (Test suites, above) was
  **not accurate as originally written** — the suite only ever tested one
  tick *past* each deadline, never the exact boundary tick itself (which
  should still succeed). Corrected above, with the tests that now actually
  prove it.
- Two of the four release-blocking defects (C1: missing `instructionId`,
  C4: RPC-error/revert conflation) are Go-side bugs in
  `go/internal/verifier`, not Solidity bugs — this document's claims about
  `WorkProofEscrow.sol`'s own test coverage were not wrong on those counts,
  but the *system* (contract + verifier together) could not have settled a
  single real Go-produced result regardless of how well-tested the contract
  alone was. A contract can be correctly tested in isolation and the
  overall system can still be non-functional; both halves need checking.
- Not corrected, stated plainly instead: this suite has 2 fuzz tests and 0
  Foundry invariant functions. That was true before the audit and remains
  true — the coverage numbers below measure line/branch/statement coverage
  from the existing unit + fuzz tests, not invariant/stateful-fuzzing
  coverage, which this repository does not have.
- Phase 0's own exit condition ("indexer path and GCP owner confirmed",
  `WORKPROOF_EXECUTION_PLAN.md:843`) was still unmet when Phase 3 and 4
  work proceeded, and remains unmet as of this correction — see
  `docs/operations/external-dependencies.md` for the current, honestly
  tracked status and why Phase 3/4 (pure contract/Go work, no external
  chain-access dependency) were not actually blocked by it even though the
  plan's literal phase ordering was not followed to the letter.

## Coverage

`forge coverage --ir-minimum` (full `via-ir` isn't supported by the coverage
instrumenter; `--ir-minimum` is forge's own recommended fallback):

Re-run after post-audit remediation (2026-08-09, see `NEW_WORK.md`
"Post-audit remediation"):

| File | Lines | Statements | Branches | Funcs |
|---|---|---|---|---|
| `contracts/WorkProofEscrow.sol` | 99.47% (187/188) | 99.24% (260/262) | 90.38% (47/52) | 100.00% (23/23) |
| `contracts/lib/FccVerdict.sol` | 86.36% (19/22) | 90.00% (27/30) | 100.00% (4/4) | 100.00% (5/5) |

`FccVerdict.sol` reached 100% branches after adding
`testInvalidVByteReturnsZero` (an out-of-range `v` signature byte, the one
genuine gap found there). Its remaining 3 uncovered lines (64-66) are the
three statements *inside* `recoverCanonical`'s `assembly ("memory-safe")`
block that extracts `r`/`s`/`v` — another tool-attribution artifact: forge's
lcov line-tracking does not instrument inline assembly bodies at all, even
though every signature-recovery test in both suites (all of
`test/FccSignatureSpike.t.sol`, plus every `settleAttempt` test that signs
with `vm.sign`) passes a real 65-byte signature through this exact block on
every call.

The remaining 5 uncovered `WorkProofEscrow.sol` branches (lines 229, 283×2,
296×2 — the `onlyOwner` modifier and the scaffold's own `setExtensionId`/
`_getExtensionId` `require(...)` checks) are a **proven tool-attribution
artifact, not missing tests**: `testOnlyOwnerCanPauseOrSetFee`,
`testSetExtensionIdCannotBeSetTwice`, and the fork suite's
`testCreateJobFailsHonestlyWhenExtensionUnregistered` /
`testSetExtensionIdFailsHonestlyWhenUnregistered` directly assert on these
exact revert reasons and pass. Isolated re-run of coverage on the local
suite alone (which alone already exercises all five paths — no fork needed)
still reports 0 hits on all five, confirming `--ir-minimum`'s branch
instrumentation doesn't correctly attribute `require(condition, "string")`
statements and modifier-inlined checks, not that the code paths are
untested. `setExtensionId`/`_getExtensionId` are intentionally left
unmodified (`setExtensionId` carries the scaffold's own "DO NOT MODIFY"
comment) rather than rewritten to custom errors just to satisfy the tool.

## Known limitations / not yet done

- Bytecode size: resolved during Phase 4. Adding `ciphertextHash` to
  `JobTerms`/`WorkProofInstruction` (see `docs/evidence/phase4-go-verifier.md`)
  dropped the margin to a critical 44 bytes, which led to discovering that
  `foundry.toml` never actually enabled the Solidity optimizer (`optimizer =
  false` was forge's own default; `via-ir = true` alone does not imply it).
  Enabling `optimizer = true` / `optimizer_runs = 200` dropped
  `WorkProofEscrow` from 24,532 to 10,455 bytes -- a 14,121-byte margin,
  re-verified with the full 90-test local+spike suite and the real Coston2
  fork suite both still green (optimization changes bytecode size/gas, never
  semantics, but this was confirmed rather than assumed). Still worth
  monitoring with `forge build --sizes` before any future large addition,
  just no longer on a knife's edge. After post-audit remediation
  (`graceEnds` guard, `ciphertextHash`-adjacent changes): 10,478 bytes,
  14,098-byte margin -- still healthy. `foundry.toml` now also pins
  `solc = "0.8.35"` explicitly (was auto-detected, a separate reproducibility
  gap an audit found independently -- see `NEW_WORK.md`).
- No full external security audit (plan Phase 12 "Security and release
  audit" is separate, later work).
- Real end-to-end dispatch (an actual signed `ActionResult` arriving from a
  real registered production TEE machine) is unproven and cannot be proven
  until Phase 0's external blockers clear (GCP Confidential Space +
  registration) — `test/RealRegistryFork.t.sol` proves everything *up to*
  that boundary honestly, not past it.
