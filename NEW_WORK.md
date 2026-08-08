# WorkProof New Work Ledger

Scaffold base:

- Upstream: `flare-foundation/fce-extension-scaffold`
- Commit: `ffb6c4ca7c160c49be59e00fe537e24d2477b000`
- Local tag: `scaffold-base-2026-08-07`

WorkProof work starts after the scaffold-base tag. This ledger separates hackathon implementation from the upstream FCC scaffold.

## Phase 1 Additions

- `WORKPROOF_EXECUTION_PLAN.md`: audited execution plan supplied before implementation.
- `SPEC.md`: P0 job terms, state machine, outcome rules, vector types, invariants, and verdict binding.
- `THREAT_MODEL.md`: P0 threat model, controls, evidence requirements, and residual risks.
- `docs/operations/external-dependencies.md`: external dependency tracker for Flare, GCP, wallets, proxy, and funding.
- `docs/security/dependency-changes.md`: dependency change log; currently no deliberate updates after scaffold base.
- `packages/schema`: source schemas and validation harness for bundle, verdict, deployment, and evidence data.
- `packages/test-bundle-sdk`: placeholder package boundary for client-side canonicalization, commitment, encryption, and preview logic.
- `relayer`: placeholder package boundary for permissionless result relay and recovery CLI.
- `web`: placeholder package boundary for the product UI.

## Phase 2 Completion (mandatory FCC signature spike + full VerdictV1 binding)

- `contracts/lib/FccVerdict.sol`: real FCC `ActionResult` signing-chain
  reconstruction (packed digest, EIP-191 wrap, canonical/low-S recovery),
  reproduced from pinned `tee-node`/`go-flare-common` source, not guessed.
- `contracts/WorkProofEscrow.sol`: `settle()` now decodes and binds the full
  `VerdictV1` schema (split into two static nested sub-structs,
  `VerdictIdentity`/`VerdictOutcome`, for solc IR stack-depth reasons only —
  the ABI wire format is unaffected); added `lockRandomness` (single
  deterministic-round-advance lab stand-in for Phase 5 secure randomness),
  per-job `setPaused` that never blocks refund/cancel/already-dispatched
  settlement, and a named `getJob` accessor (the `jobs` mapping is now
  `internal` — its auto-generated 20-field public getter hit the same IR
  limit as the VerdictV1 decode).
- `go/cmd/fcc-spike`: generates a real Go-signed `ActionResult` vector
  (`docs/evidence/fcc-signature-spike-v1.json`) using only real
  go-ethereum/go-flare-common/tee-node exported code.
- `test/FccSignatureSpike.t.sol`, `test/WorkProofEscrow.t.sol`: the plan's
  section-9 mandatory spike proof and all 10 Phase 2 "required simulations"
  plus a fuzzed accounting invariant. See `docs/evidence/phase2-simulation.md`
  for the full breakdown and the one real bug this caught.
- `lib/forge-std` (git submodule, `foundry.lock`, `remappings.txt`): added
  for test cheatcodes/assertions; not previously a dependency.

## Scaffold-Owned Areas

The following directories are inherited from the scaffold and should be changed only when implementing WorkProof-specific behavior:

- `contracts`
- `go`
- `scripts`
- `tools`
- `docker`
- `proxy`
- `testing`
- `typescript`
- `python`

Any dependency update must be listed in `docs/security/dependency-changes.md` with reason, old version, new version, risk, and validation.
