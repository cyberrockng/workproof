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

## Phase 3 (production smart contracts)

- `contracts/WorkProofEscrow.sol`: rewritten around real production
  dependencies rather than the Phase 2 lab's constructor-pinned/self-invented
  values:
  - FTestXRP and the "Relay"/"RandomNumberV2" registry entries are resolved
    live via `ContractRegistry` at construction, never pasted as constants
    (plan section 10).
  - The expected TEE is selected and pinned in `createJob` (not at dispatch)
    via `ITeeMachineRegistry.getRandomTeeIds` + the live-verified
    `getTeeMachineStatus`, and never silently re-selected — `dispatchVerification`
    only re-confirms the pinned machine is still PRODUCTION.
  - Real dispatch through `ITeeExtensionRegistry.sendInstructions`, using
    the registry's own returned `instructionId` instead of a
    locally-invented one.
  - Full `createdAt`/`acceptBy`/`submitBy`/`graceEnds`/`verificationTimeout`
    deadline set, `expireVerification` for dispatch timeouts, and the
    complete section-8 event list.
  - Real `treasury` as a validated constructor parameter (the Phase 2 lab's
    `keccak256("WORKPROOF_TREASURY")`-derived address was a real bug — nobody
    holds that key, so a fee sent there would be unrecoverable).
  - `nonReentrant` (OpenZeppelin, real pinned dependency) on every fund-moving
    or external-call-making function; `VerdictOutcome.expiresAt` bound to
    the job's stored `graceEnds` exactly (a TEE can no longer claim a later
    expiry than what it was actually dispatched with); `issuedAt` bounded
    between dispatch and now; a timed-out instruction can never settle even
    without an explicit `expireVerification` call first.
  - Global `pauseNewWork`/`unpauseNewWork` blocks only `createJob`/`submitAttempt`;
    every other function (`acceptJob`, `lockRandomness`, `dispatchVerification`,
    `settleAttempt`, `expireVerification`, `cancelUnaccepted`, `refundExpired`)
    remains permissionless during pause, per the plan's liveness invariant.
- `contracts/interfaces/ITeeMachineRegistry.sol`: added `getTeeMachineStatus`,
  independently re-confirmed live against the real FlareTeeManager diamond
  (distinct custom-error selector `0xceb05b68` for an unregistered address,
  vs. the diamond's different generic selector for a truly nonexistent
  function) before relying on it — not copied from memory unverified.
- `lib/flare-foundry-periphery-package` (git submodule) +
  `@openzeppelin-contracts@5.2.0-rc.1` (via its soldeer deps): real
  `ContractRegistry`/`IAssetManager`/`IRelay`/`RandomNumberV2Interface`/
  `SafeERC20`/`ReentrancyGuard`. Same package+version already used by the
  sibling Ajose project.
- `test/mocks/Mocks.sol`: deterministic local test doubles (`MockToken`,
  `MockFlareContractRegistry`, `MockAssetManager`, `MockRelay`,
  `MockRandomNumberV2`, `MockTeeExtensionRegistry`, `MockTeeMachineRegistry`),
  each matching only the real interface signatures the escrow actually calls.
- `test/WorkProofEscrow.t.sol`: rewritten as a fully local, deterministic
  suite (78 tests) using `vm.etch` to install the mock registry at the real
  hardcoded `FlareContractRegistry` address the constructor always calls —
  covers every state transition/custom error, exact deadline boundaries,
  secure/insecure/not-ready randomness, TEE re-confirmation and pinning,
  reentrancy probes (malicious token + malicious extension registry), every
  VerdictV1 field mutation, replay, timeout-then-resubmit staleness, pause
  liveness, fee rounding, and a fuzzed solvency/single-payout invariant.
- `test/RealRegistryFork.t.sol` (new, 7 tests): proves the real Coston2
  addresses against the actual contract — constructor resolution, real
  6-decimal FTestXRP, a real FXRP holder funding a client, the live
  diamond's `getTeeMachineStatus`/`nextPublicExtensionId` being real
  selectors, and the honest (not wrong-ABI-crash) failure mode while our
  extension remains unregistered.
- `foundry.toml`: added `evm_version = "cancun"` and the `coston2` RPC
  endpoint.

## Phase 4 (Go FCC verifier)

- `contracts/WorkProofEscrow.sol`, `contracts/interfaces/`: added
  `bytes32 ciphertextHash` to `JobTerms`/`WorkProofInstruction`/`createJob`
  -- a real gap found mid-phase (SPEC.md's Job Terms require it, Phase 3
  never added it). Touched every `createJob` call site. Also: `foundry.toml`
  now sets `optimizer = true`/`optimizer_runs = 200` (it never had been —
  `via-ir` alone doesn't enable the optimizer), which dropped
  `WorkProofEscrow` from a critical 44-byte to a healthy 14,121-byte
  EIP-170 margin. Both changes re-verified against the full local (90-test)
  and real Coston2 fork (7-test) suites, still green.
- `go/internal/config/config.go`: `OPTypeWorkProof`/`OPCommandVerify`
  constants, `WorkProofEngineVersion`, resource limits (plan section 11),
  and `WORKPROOF_*` env vars (RPC URL, escrow address, RandomNumberV2
  address, ciphertext gateway allowlist).
- `go/pkg/types/workproof.go`: hand-written ABI codec for
  `WorkProofInstruction`/`VerdictV1`, proven byte-exact against real
  Solidity `abi.encode(...)` output in both directions
  (`workproof_test.go`, `workproof_instruction_test.go`). Caught a real
  go-ethereum ABI/Go-naming interop bug along the way (struct-to-tuple
  `Pack` matches field names via `"id"`->`"Id"` capitalization, not
  idiomatic Go `ID` acronym casing) — documented in-code so it isn't
  "fixed" back by a future refactor.
- `go/pkg/contracts/workproofescrow/` (abigen-generated, not hand-written)
  + `scripts/generate-workproof-bindings.sh`: real read bindings for
  `WorkProofEscrow.sol`, the same recipe the scaffold's own
  `scripts/generate-bindings.sh` uses for `HelloWorldInstructionSender`.
- `go/internal/verifier/`: the verification engine — bundle schema
  validation + real RFC 8785 (`gowebpki/jcs`) canonicalization; the real
  `/decrypt` client (base64 wire format per `docs/extension-contract.md`
  §3); an HTTPS-only, host-allowlisted, SSRF/DNS-rebinding-resistant
  ciphertext fetcher; independent on-chain re-verification of the random
  number and job state (never trusting the relayed instruction blindly);
  deterministic Fisher-Yates vector selection and execution of all 5 P0
  vector types; `VerdictV1` + redacted report-hash production; status=1/0
  semantics per plan Phase 4 tasks 11-12.
- `go/internal/extension/extension.go`: `processWorkProof`/`processVerify`
  wired into the existing OPType/OPCommand routing, matching the scaffold's
  `processGreeting`/`processSayHello` pattern exactly.
- `go/go.mod`/`go/go.sum`: `go mod tidy` corrected `gowebpki/jcs` from a
  stray `// indirect` marker to a proper direct `require` (it's imported
  directly by `go/internal/verifier/bundle.go`); see
  `docs/security/dependency-changes.md` for the full entry.
- See `docs/evidence/phase4-go-verifier.md` for full test coverage, the two
  real bugs/gaps found and fixed, and honestly-tracked remaining gaps
  (WORKPROOF conformance fixtures, full `Verify()` end-to-end testing —
  deliberately deferred to Phase 5, which exists specifically for that).

## Post-audit remediation (2026-08-09)

An independent audit found four release-blocking defects and several
high-severity gaps in the Phase 3/4 work above. Every finding checked
against source was accurate. Fixed so far:

- **C1** (`go/internal/verifier/verifier.go`): `Verify()` never threaded the
  real `ActionResult.ID` into `VerdictIdentity.InstructionId` — every
  Go-built verdict had a zero instructionId, so `WorkProofEscrow.sol`'s
  signature recovery (which reconstructs the digest using
  `v.id.instructionId` extracted from the decoded data) always recovered
  the wrong signer. No Go-generated result could ever settle. Fixed by
  threading `action.Data.ID` through from `processVerify`
  (`go/internal/extension/extension.go`) and cross-checking it against
  `job.Current.InstructionId`.
- **C3** (`contracts/WorkProofEscrow.sol`): `settleAttempt` only checked the
  dispatch timeout, never `graceEnds` — since `timeoutAt` is
  caller-chosen and unbounded relative to `graceEnds`, a late Pass could
  pay the contractor after the client's refund deadline had already
  passed, racing `refundExpired`. Fixed with an explicit
  `block.timestamp > j.terms.graceEnds` guard; added exact-boundary tests
  for `acceptBy`/`submitBy`/`graceEnds` and a test proving the client wins
  the race via `refundExpired` instead.
- **C4** (`go/internal/verifier/vectors.go`): `executeEthCallReverts`
  treated any RPC/transport error identically to a genuine EVM revert — an
  RPC outage or adversarial provider could force a false Pass. Fixed by
  distinguishing genuine JSON-RPC errors (`rpc.Error`) from transport
  failures. Also fixed: `msg.value` was declared/validated in the bundle
  schema but silently never applied to payable-function test calls.
- **High**: `bundle.PublicSpecHash` was format-validated but never compared
  to the real on-chain `instr.SpecHash` — a client could submit hidden
  tests unrelated to the public agreement. Fixed in `fetchAndDecryptBundle`.
- **High**: `EngineVersionHash` was copied from client/job input straight
  into every verdict rather than reflecting the verifier's own actual
  running code — the binding was circular. Fixed by computing
  `keccak256(config.WorkProofEngineVersion)` once and using/checking that.
- **C2**: the deployment pipeline (`scripts/pre-build.sh`,
  `tools/cmd/deploy-contract`) deployed the scaffold's sample
  `HelloWorldInstructionSender`, never `WorkProofEscrow` — confirmed
  firsthand by actually running it. `register-extension` turned out to
  already be fully generic (it registers any address against the
  `FlareTeeManager` diamond via `fccutils.SetupExtension`, no HelloWorld
  dependency), so the real gap was narrower than it first looked: only the
  deploy step, plus a missing `WorkProofEscrow.setExtensionId()` call
  (which adopts whatever id the registration step just created — it does
  not register anything itself). Added `tools/pkg/contracts/workproofescrow`
  (full Deploy+Transact abigen bindings, via new
  `scripts/generate-workproof-tools-bindings.sh`), `tools/pkg/utils/workproof.go`
  (`DeployWorkProofEscrow`, `SetWorkProofEscrowExtensionId`), and two new
  commands (`tools/cmd/deploy-workproof-escrow`,
  `tools/cmd/set-workproof-extension-id`). `pre-build.sh` now deploys
  WorkProofEscrow, registers it, calls its `setExtensionId()`, and writes
  `WORKPROOF_ESCROW_ADDRESS` into `config/extension.env` alongside the
  existing `EXTENSION_ID`/`INSTRUCTION_SENDER` keys. Verified for real
  against live Coston2 (`--preflight-only`): resolves the real
  `FlareTeeManager` diamond and the configured treasury/fee, and fails
  honestly at the same known funding blocker as before — never touches
  HelloWorld. The scaffold's own `deploy-contract`/HelloWorld tooling is
  left intact (untouched, still compiles, still usable for the unrelated
  `run-test` SayHello/SayGoodbye E2E harness), just no longer wired into
  `pre-build.sh`.

Remaining tracked findings (DNS-rebinding TOCTOU gap, unused resource-limit
wiring, fail-closed verifier config + `/state` health, image/compiler
pinning, verdict-timestamp determinism + RPC-trust threat-model
documentation, evidence-doc corrections) are still open.

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
