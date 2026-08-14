# WorkProof

A pre-funded escrow for objective smart-contract deliverables, settled automatically by [Flare Confidential Compute](https://dev.flare.network/fassets/guides/fcc) (FCC).

A client locks principal plus a success fee. A contractor deploys a Coston2 contract as their submission. A Go extension running inside an FCC TEE independently re-derives the job's randomness, selects a subset of the client's hidden test vectors, executes them against the deployed artifact over `eth_call`/`eth_getCode`/`eth_getStorageAt`, and returns a signed verdict. The escrow verifies that signature came from the TEE it pinned at job creation, checks every field of the verdict against on-chain state, and on PASS pays principal to the contractor and the success fee to the treasury — or lets the client refund after the grace deadline. No admin ever touches the funds.

Built for the Flare Summer Signal hackathon on the [official FCC extension scaffold](https://github.com/flare-foundation/fce-extension-scaffold) (Go implementation only — WorkProof does not use the scaffold's Python/TypeScript paths).

## Status

- **Phase 3 (production contracts)** and **Phase 4 (Go verifier)** are complete: `contracts/WorkProofEscrow.sol` resolves every Flare dependency live (FTestXRP, secure randomness, TEE registry) instead of pasting addresses; `go/internal/verifier` runs the full VERIFY flow — bundle decrypt/validate, independent on-chain re-verification, deterministic vector selection and execution, signed `VerdictV1` production.
- **Independent audit status**: source-level hardening has been applied for the August 2026 independent audit findings that can be fixed in code: bounded verification windows, late non-pass refund-pending state, HTTP body/time limits, stricter config parsing, safer relayer key handling, and fresh-clone bootstrap. Remaining submission gates are operational evidence gates: redeploy the current source to Coston2, verify explorer/source provenance, and record a role-separated demo journey.
- **Phase 5 (Coston2 simulated-attestation integration test)** is complete on the live Coston2 FCC deployment recorded at source commit `2779d77983e0f23d586b1ed56507e9a935644951`: extension `66223`, escrow/instruction sender `0x7B984320aA969Ad6522E7c902371dD208C1760A4`, proxy `https://retention-pasta-clip.ngrok-free.dev`, TEE `0x962cf74e9673170f273576764c60dF2fc13A28aa`, registry status `2`, and successful PASS/pay, FAIL/no-pay, and refund paths. Evidence is in [`deployments/coston2.json`](deployments/coston2.json) and [`docs/evidence/demo-run.json`](docs/evidence/demo-run.json). Newer source commits include audit hardening and must be redeployed before claiming that exact bytecode is live.
- **Simulation honesty**: this hackathon deployment uses `SIMULATED_TEE=true`/`MODE=1` with attestation `magic_pass` and platform `TEST_PLATFORM`. Flare organizers confirmed simulated TEEs are acceptable for the hackathon demo; this is not claimed as GCP AMD SEV hardware attestation.
- **Not yet built**: a polished deployed web UI. Three Foundry invariant functions and two fuzz tests exist. Stated plainly rather than implied otherwise.

Last verified checks are listed in [`docs/evidence/AUDIT_RESPONSE.md`](docs/evidence/AUDIT_RESPONSE.md). Re-run them after any contract or verifier change.

## Architecture

```
Client ──createJob(contractor, expectedTee, principal, fee, deadlines, specHash,
       │            privateBundleHash, ciphertextHash, engineVersionHash)
       ▼
WorkProofEscrow.sol ──funds locked, TEE pinned via TeeMachineRegistry──┐
       │                                                                │
Contractor ──submitAttempt(artifactAddress)──▶ escrow records          │
       │      real .codehash at submission block                      │
       ▼                                                                │
escrow ──lockRandomness()──▶ Flare secure random round committed       │
       │                                                                │
escrow ──dispatchVerification()──▶ TeeExtensionRegistry.sendInstructions
       │                                    (WorkProofInstruction, ABI-encoded)
       ▼
Go FCC extension (go/internal/verifier) inside the TEE:
  1. re-reads job/attempt on-chain, rejects any mismatch
  2. recomputes artifact codehash and the random value's hash
  3. fetches + decrypts the private bundle, checks every commitment
  4. derives the test seed, selects vectors deterministically (Fisher-Yates)
  5. executes each vector as a bounded eth_call / eth_getCode / eth_getStorageAt
  6. signs a VerdictV1 (PASS / FAIL / INCONCLUSIVE) as the ActionResult
       │
       ▼
escrow.settleAttempt(verdict, signature) ──verifies signer == pinned TEE,
       │                                    checks every VerdictV1 field
       ▼
  PASS → principal to contractor + fee to treasury · FAIL → contractor may resubmit
  · past graceEnds → client may refund instead, regardless of outcome
```

The escrow is itself the FCC instruction sender — it calls `TeeExtensionRegistry.sendInstructions` directly from `dispatchVerification`, so no separate wrapper contract like the scaffold's sample `HelloWorldInstructionSender` is needed.

## Repository layout

```
├── contracts/
│   ├── WorkProofEscrow.sol          # the escrow + FCC instruction sender
│   ├── lib/FccVerdict.sol           # real FCC ActionResult signing-chain reconstruction
│   └── interfaces/                  # ITeeExtensionRegistry, ITeeMachineRegistry
├── go/
│   ├── internal/verifier/           # the VERIFY handler: bundle, ciphertext, vectors, verifier
│   ├── internal/extension/          # OPType/OPCommand routing (processWorkProof/processVerify)
│   ├── internal/config/             # WORKPROOF_* env vars, resource limits
│   └── pkg/types/workproof.go       # hand-written ABI codec, byte-proven against real Solidity output
├── tools/cmd/
│   ├── deploy-workproof-escrow/     # deploys the real WorkProofEscrow (not HelloWorld)
│   ├── set-workproof-extension-id/  # adopts the registry-assigned extension id
│   └── register-extension/          # generic FlareTeeManager registration (scaffold-provided)
├── test/
│   ├── WorkProofEscrow.t.sol        # local, deterministic (vm.etch'd mock registry)
│   ├── RealRegistryFork.t.sol       # against a real live Coston2 fork
│   └── FccSignatureSpike.t.sol      # the mandatory FCC signature-compatibility proof
├── SPEC.md                          # job terms, state machine, outcome table, vector types
├── THREAT_MODEL.md                  # threats, controls, residual risks
├── NEW_WORK.md                      # what WorkProof added on top of the scaffold, phase by phase
├── docs/evidence/                   # per-phase evidence, including post-audit corrections
├── docs/operations/                 # externally-blocked dependencies, tracked honestly
└── WORKPROOF_EXECUTION_PLAN.md      # the phase-by-phase build plan this was executed against
```

`go/` is the only active language implementation — `python/`, `typescript/`, and the scaffold's own `contracts/InstructionSender.sol` sample are inherited but unused.

## Building and testing

```bash
# One-time dependency bootstrap for a fresh clone.
bash ./scripts/bootstrap.sh

# Solidity: local + spike suites (no network needed)
forge test --match-contract "WorkProofEscrowTest|FccSignatureSpikeTest"

# Solidity: against a real live Coston2 fork
forge test --match-contract RealRegistryForkTest

# Solidity: everything, plus coverage
forge test
forge coverage --ir-minimum   # full via-ir isn't supported by the coverage instrumenter

# Go: build, format, vet, test with the race detector
cd go && go build ./... && gofmt -l . && go vet ./... && go test ./... -race -cover

# Wire-contract conformance (starts the extension, no chain/Docker needed)
./scripts/test-conformance.sh go

# Evidence provenance: each recorded sourceCommit must be in git history.
bash ./scripts/check-evidence-commits.sh
```

Deploying to Coston2 (`./scripts/pre-build.sh`) needs `DEPLOYMENT_PRIVATE_KEY` (funded) and `WORKPROOF_TREASURY` set. The current live hackathon deployment is recorded in [`deployments/coston2.json`](deployments/coston2.json); operational caveats and remaining non-code items are tracked in [`docs/operations/external-dependencies.md`](docs/operations/external-dependencies.md).

## Documentation map

| Doc | Covers |
|---|---|
| [SPEC.md](SPEC.md) | Job terms, state machine, outcome rules, P0 vector types, VerdictV1 binding |
| [THREAT_MODEL.md](THREAT_MODEL.md) | Threats, controls, required evidence, residual risks |
| [NEW_WORK.md](NEW_WORK.md) | Everything WorkProof added on top of the scaffold, phase by phase, including every real bug found and fixed |
| [docs/evidence/phase3-production-contracts.md](docs/evidence/phase3-production-contracts.md) | Contract test suites, coverage, and post-audit corrections |
| [docs/evidence/phase4-go-verifier.md](docs/evidence/phase4-go-verifier.md) | Verifier test coverage, cross-language ABI proofs, and post-audit corrections |
| [docs/evidence/demo-run.json](docs/evidence/demo-run.json) | Coston2 PASS, FAIL/no-pay, and refund transaction evidence |
| [docs/operations/external-dependencies.md](docs/operations/external-dependencies.md) | What's blocked on funding/credentials vs. what's just not built yet |
| [REPRODUCIBILITY.md](REPRODUCIBILITY.md) | What the Go build actually guarantees |
| [docs/extension-contract.md](docs/extension-contract.md) | The normative FCC wire/container contract (scaffold-authored, still binding) |
| [WORKPROOF_EXECUTION_PLAN.md](WORKPROOF_EXECUTION_PLAN.md) | The phase-by-phase plan this was executed against |
