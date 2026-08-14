# WorkProof Release Signoff

Date: 2026-08-14
Network: Coston2
Scope: Flare Summer Signal hackathon submission evidence

## Signed State

- Coston2 WorkProof escrow/instruction sender:
  `0x7B984320aA969Ad6522E7c902371dD208C1760A4`
- Extension ID: `66223`
- TEE ID: `0x962cf74e9673170f273576764c60dF2fc13A28aa`
- Public proxy: `https://retention-pasta-clip.ngrok-free.dev`
- Registry status: `2`
- Attestation mode: simulated (`magic_pass`, `TEST_PLATFORM`)

## Evidence

- FCC registration completed and availability proof was obtained:
  `0x47c909e7b96829fb6b47bcf42768c9a2c1a5845fa1b580ba3793b91c2e40436c`
- PASS/pay demo settled job `2` as `Paid`:
  `0xcddf06c7a85973f32363cafc8989c17949062413ae789ec59e320e17d206b556`
- FAIL/no-pay demo settled job `3` as `AwaitingResubmission` with
  `settled=false`:
  `0x649d527226d0949e04b112d6c4d55e9854bb39b645147d1b1b1d4ce1be1092de`
- Refund demo settled job `4` as `Refunded`:
  `0xca8d403aeec99d37ec372c26314be0843aadd0a4e78ad19e775181d2f299f53a`
- Full structured evidence:
  [`docs/evidence/demo-run.json`](demo-run.json)

## Checks

- `./scripts/check-versions.sh`
- `cd go && go test ./cmd/workproof-gateway ./cmd/workproof-phase5`
- `cd tools && go test ./...`
- `forge test --match-contract WorkProofEscrowTest` (`85/85` passed)

## Limitations

- This is a simulated-attestation FCC run. It does not claim GCP AMD SEV
  hardware attestation.
- A polished deployed web UI is not included. The submission evidence is the
  repository, live FCC endpoint, CLI demo flow, and Coston2 transaction trail.
- DoraHacks upload, final video publication, and account-level submission
  receipt are outside the repository and must be completed through the user's
  browser/account.

## Decision

The Coston2 simulated-attestation WorkProof path is release-ready for hackathon
submission with the limitations above disclosed plainly.
