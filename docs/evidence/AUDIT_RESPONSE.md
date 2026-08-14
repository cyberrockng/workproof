# Audit Response

Date: 2026-08-14
Reviewed public commit: `2779d77983e0f23d586b1ed56507e9a935644951`

## Fixed

| Finding | Response |
|---|---|
| H-01 relayer rejects live `threshold` submission tag | Fixed. The relayer now uses the WorkProof-specific `threshold` tag and rejects the scaffold `submit` tag. Relayer tests updated. |
| M-01 manifests cite non-public `sourceCommit` | Fixed. `deployments/coston2.json` and `docs/evidence/demo-run.json` now cite public commit `2779d77983e0f23d586b1ed56507e9a935644951`. Added `scripts/check-evidence-commits.sh`. |
| M-02 fresh checkout setup gaps | Fixed. `pre-build.sh` now invokes binding generation through `bash`, README includes the nested Soldeer bootstrap commands, and CI runs clean-checkout contract/Go/tools/workspace plus evidence/version checks. |
| M-03 verdict count fields not semantically checked on-chain | Fixed in source. `WorkProofEscrow` now rejects zero/inconsistent counts: PASS requires all executed vectors to pass; FAIL/INCONCLUSIVE require at least one non-passing/skipped result. Solidity and Go tests added. |
| H-02 multiple active TEE/ciphertext-target mismatch | Fixed in source. `createJob` now takes explicit `expectedTee`, verifies it is PRODUCTION, stores it, dispatches only to it, and the Phase 5 helper derives it from the same public key used for encryption. |
| L-01 README payment wording | Fixed. README now states principal goes to contractor and fee goes to treasury. |
| L-02 README invariant count | Fixed. README now states three invariant functions and two fuzz tests. |

## Deferred

| Finding | Status |
|---|---|
| M-04 mainnet readiness | Deferred. This is a Coston2 hackathon deployment using simulated attestation. Real GCP Confidential Space attestation, deployment verification, monitoring, key rotation, RPC trust hardening, and formal external audit remain required before production/mainnet claims. |

## Verification

- `./scripts/check-versions.sh`
- `bash ./scripts/check-evidence-commits.sh`
- `cd relayer && npm test` (`8/8`)
- `cd go && go test ./internal/verifier`
- `forge test --match-contract WorkProofEscrowTest` (`89/89`)
- `forge test --match-contract WorkProofEscrowInvariantTest` (`3/3`)
- `forge test --match-contract RealRegistryForkTest` (`7/7`)

## Note

The live Coston2 evidence remains tied to public commit
`2779d77983e0f23d586b1ed56507e9a935644951`. Source changes after that commit,
including explicit expected-TEE selection, improve the repository and should be
redeployed before claiming the exact new bytecode is live on Coston2.
