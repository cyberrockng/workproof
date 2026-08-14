# Audit Response

Date: 2026-08-14
Latest source reviewed locally: current working tree after the independent audit follow-up.

## Fixed In Source

| Finding | Response |
|---|---|
| H-02 upgradeable/replaced artifact can pass then change behavior | Partially fixed in contract source. `settleAttempt` now rechecks the live artifact `codehash` against the submitted codehash before accepting any verdict, and regression coverage proves changed code cannot settle. This blocks direct bytecode replacement before payment, but generic proxy implementation changes remain a residual risk until WorkProof either rejects proxy patterns in the verifier or defines an immutable-artifact policy. |
| M-02 unbounded verification timeout | Fixed. `createJob` enforces `MIN_VERIFICATION_TIMEOUT` and `MAX_VERIFICATION_TIMEOUT`; `dispatchVerification` clamps the live timeout to `graceEnds`, so `timeoutAt` cannot outlive the client refund boundary. |
| M-03 late FAIL advertises resubmission when impossible | Fixed. Late FAIL/INCONCLUSIVE results after `submitBy` now move the job to `RefundPending` instead of resubmission states. |
| M-05 HTTP action endpoint lacks body/server timeout limits | Fixed. `/action` now enforces JSON content type, a `MaxBytesReader` body cap, strict one-object JSON decoding, and server read/write/header/idle timeouts. The gateway server also has explicit timeouts. |
| M-06 relayer secrets in process args and deployer fallback | Fixed. The relayer no longer accepts `--private-key`, no longer falls back to `PROXY_PRIVATE_KEY` or `DEPLOYMENT_PRIVATE_KEY`, and does not put the private key into `cast` argv. Use `WORKPROOF_RELAYER_PRIVATE_KEY` or `RELAYER_PRIVATE_KEY`. |
| M-08/L-02 verifier chain and state cross-check gaps | Fixed. The verifier rejects full-width chain IDs instead of `uint64` truncating them, and cross-checks on-chain job state, settled flag, expiry, and instruction ID before signing. |
| M-09 fresh-clone build instructions omit generated bindings | Fixed. `scripts/bootstrap.sh` installs Soldeer dependencies and regenerates contract bindings for a fresh clone. README now points to this script. |
| L-01 relayer expected-TEE preflight is a no-op | Fixed by removal. The relayer rejects `--expected-tee`; the escrow remains the settlement authority for pinned-TEE signature verification. |
| L-03 invalid configured addresses silently become zero | Fixed. WorkProof extension config now rejects malformed or zero addresses instead of `HexToAddress` silently producing zero. |
| L-05 owner single non-transferable | Fixed. Owner transfer is now two-step: `transferOwnership` then `acceptOwnership`. |
| L-06 stale/overbroad production wording and dashboard counts | Updated. README now separates source hardening from deployment/evidence gates; dashboard test count updated. |

## Still Open / Requires Operational Evidence

| Finding | Status |
|---|---|
| H-01 one compromised RPC can cause false PASS/FAIL | Not fully fixed. The verifier now has stricter on-chain state checks, but vector execution still trusts one configured RPC. Production-grade mitigation requires multiple independent Coston2 RPC providers with quorum comparison for `eth_call`, `eth_getCode`, `eth_getStorageAt`, and randomness reads. |
| H-02 generic proxy upgradeability | Partially fixed only. Direct artifact code replacement before settlement is blocked, but proxy implementation changes with unchanged proxy bytecode require verifier-side proxy detection/denial or an explicit immutable-artifact policy. |
| H-03 live deployment predates security-relevant changes and is unverified | Closed for hackathon evidence. The current source commit `4b4358ca941bf6f64926d98425b34921f09b15e7` was redeployed to Coston2 as WorkProofEscrow `0x2eA5bBb676AD142cFa24A20A9Fd950e81640E2dD`, extension `66282`, and Sourcify returned `exact_match` for verification job `faa71f5a-0f62-4fae-b317-befae579ffe3`. |
| H-04 recorded demo is single-operator | Still open. The fresh Coston2 run proves PASS/pay, FAIL/no-pay, and refund, but it still used the available deployer key as the runner because separate client, contractor, and relayer private keys are not present locally. |
| M-01 production registry status accepted without codehash/attestation policy | Still open for production. Simulated TEE is acceptable for hackathon judging, but mainnet needs a real attestation/codehash policy. |
| M-04 public/private fairness not enforceable | Still residual. WorkProof proves commitments and hidden-vector execution, but it cannot fully prove public spec fairness without a stronger specification/review process. |
| M-07 live endpoint availability | Closed at latest check. `https://retention-pasta-clip.ngrok-free.dev/info` returned HTTP 200 on 2026-08-14, reporting extension `66282`, simulated attestation `magic_pass`, and signing policy `5939`. It still must be kept running through judging if live verification is expected. |
| Mainnet readiness | Deferred. Real GCP Confidential Space attestation, formal audit, deployment verification, monitoring, key rotation, RPC quorum, and operational runbooks remain required before valuable assets. |

## Verification Run

- `./scripts/check-versions.sh`
- `bash ./scripts/check-evidence-commits.sh`
- `cd relayer && npm test` (`8/8`)
- `cd go && go test ./...`
- `cd tools && go test ./...`
- `forge test --match-contract WorkProofEscrowTest` (`94/94`)
- `forge test --match-contract WorkProofEscrowInvariantTest` (`3/3`)
- `forge test --match-contract FccSignatureSpikeTest` (`10/10`)
- `forge test --match-contract RealRegistryForkTest` (`7/7`)
- Fresh Coston2 deployment and FCC evidence run from source commit
  `4b4358ca941bf6f64926d98425b34921f09b15e7`.
- `forge verify-contract ...` returned Sourcify `exact_match` for
  `0x2eA5bBb676AD142cFa24A20A9Fd950e81640E2dD`.

## Submission Gate

The hackathon submission evidence now has a fresh Coston2 deployment, exact-match source verification, live endpoint check, and PASS/pay, FAIL/no-pay, and refund evidence. The only audit-evidence item still open is role-separated transaction evidence, which requires separate client, contractor, and relayer private keys. Mainnet remains out of scope until real hardware attestation, RPC quorum, proxy policy, and a formal production audit are complete.
