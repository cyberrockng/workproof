# WorkProof Release Signoff

Date: 2026-08-14
Network: Coston2
Scope: Flare Summer Signal hackathon submission evidence

## Signed State

- Coston2 WorkProof escrow/instruction sender:
  `0x2eA5bBb676AD142cFa24A20A9Fd950e81640E2dD`
- Extension ID: `66282`
- TEE ID: `0x04635178b552b00d363557e1296aC71a7241eE87`
- Public proxy: `https://retention-pasta-clip.ngrok-free.dev`
- Registry status: `2`
- Attestation mode: simulated (`magic_pass`, `TEST_PLATFORM`)
- Source commit: `4b4358ca941bf6f64926d98425b34921f09b15e7`
- Source verification: Sourcify `exact_match`
  (`faa71f5a-0f62-4fae-b317-befae579ffe3`)

## Evidence

- FCC registration completed and availability proof was obtained:
  `0x691a17cad8068642315e4a9b38ee8c917dcb857f7e580af246a56a8edeb0cbd5`
- PASS/pay demo settled job `1` as `Paid`:
  `0xe03f1262d6c2d11b7d924fde3ad0547012e6088f98ec377a66675fe29305ece7`
- FAIL/no-pay demo settled job `2` as `AwaitingResubmission` with
  `settled=false`:
  `0x773c68ec74311c7081f12b656ab7fb4764206fb8fdc51aa7415c4a29e641c726`
- Refund demo settled job `3` as `Refunded`:
  `0xa461ff44d918c0d23a28f4271ec7803948a8c2b129d42ea8777a585f8b2963df`
- Full structured evidence:
  [`docs/evidence/demo-run.json`](demo-run.json)

## Checks

- `./scripts/check-versions.sh`
- `cd go && go test ./...`
- `cd tools && go test ./...`
- `forge test --match-contract WorkProofEscrowTest` (`89/89` passed)
- `forge test --match-contract WorkProofEscrowInvariantTest` (`3/3` passed)
- `forge test --match-contract RealRegistryForkTest` (`7/7` passed)
- `cd relayer && npm test` (`8/8` passed)
- `bash ./scripts/check-evidence-commits.sh`
- `forge verify-contract ...` returned Sourcify `exact_match`
- `curl https://retention-pasta-clip.ngrok-free.dev/info` returned HTTP 200
  at the latest check

## Limitations

- This is a simulated-attestation FCC run. It does not claim GCP AMD SEV
  hardware attestation.
- Role-separated evidence is not complete. The fresh Coston2 run used the
  available deployer key as the runner because separate client, contractor, and
  relayer private keys are not available locally.
- A public deployed frontend URL is not recorded. The submission evidence is
  the repository, live FCC endpoint, CLI demo flow, local static UI, and
  Coston2 transaction trail.
- DoraHacks upload, final video publication, and account-level submission
  receipt are outside the repository and must be completed through the user's
  browser/account.

## Decision

The Coston2 simulated-attestation WorkProof path is release-ready for hackathon
submission with the role-separation and hardware-attestation limitations above
disclosed plainly.
