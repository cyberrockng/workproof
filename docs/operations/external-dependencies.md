# External Dependencies

Status: mostly closed for the hackathon submission. Phase 5 is complete on
Coston2 with simulated attestation. Remaining items are submission operations
and post-hackathon production hardening, not code blockers.

This file tracks dependencies that cannot be completed from source code alone.
Do not commit private keys, database credentials, VPN profiles, bearer tokens,
plaintext hidden vectors, or unredacted logs.

| Dependency | Owner | Required evidence | Status |
|---|---|---|---|
| DoraHacks project registration | Product | Project URL and team confirmation | **External/account task** — user reported registration existed, but final submission receipt is not stored in this repo. |
| Deployer/client/contractor/treasury/relayer wallets | Operations | Public addresses only; no private keys | **Confirmed 2026-08-12** — addresses generated locally and kept in gitignored `.env.coston2`; public addresses are below. |
| C2FLR and FTestXRP test funds | Operations | Faucet or transfer evidence | **Confirmed 2026-08-12** — deployer, client, contractor, treasury, and relayer each had 100 C2FLR and 10 FTestXRP by live Coston2 RPC. |
| Coston2 indexer credentials/path | Operations | Sanitized connectivity result, no credentials | **Resolved 2026-08-14** — ext-proxy is using the Flare-provided shared Coston2 indexer DB. Credentials are intentionally not recorded here. |
| Stable HTTPS proxy route to port 6664 | Operations | Domain, TLS route, and uptime plan | **Confirmed 2026-08-14** — reserved ngrok domain `https://retention-pasta-clip.ngrok-free.dev` reaches `/info` through the WorkProof gateway. |
| FCC machine registration and availability | Engineering | Extension ID, TEE ID, status, instruction IDs | **Confirmed 2026-08-14** — extension `66282`, TEE `0x04635178b552b00d363557e1296aC71a7241eE87`, registry status `2`, availability proof obtained. |
| GCP Confidential Space AMD SEV operator | Operations | Real hardware attestation and image handoff | **Deferred** — organizers confirmed simulated TEE is acceptable for the hackathon demo. Real hardware attestation remains Phase 7/post-hackathon work. |
| Web UI and demo video | Product | Public URL/video link | **Partially complete** — the repo contains a local static web UI and live FCC evidence, but no public deployed frontend URL is recorded here. Video/upload must be completed through the user's accounts. |
| Role-separated evidence keys | Operations | Separate client, contractor, and relayer transaction signatures | **Blocked locally** — the environment available to this run contains only deployer and proxy private keys. Public role addresses exist, but their private keys are not present locally. |

## Generated Wallets

Real Coston2-format keypairs were generated locally via `cast wallet new --json`
and validated before use. Private keys live only in gitignored local environment
files and must never be committed.

| Role | Address |
|---|---|
| Deployer (= `INITIAL_OWNER`) | `0x9D51803c7aEC6F8A67E0A578158e5f84F774EEAB` |
| Client | `0xcdd7C06e8CD8423f9cea9c3dd5857fCcE69ABc27` |
| Contractor | `0x29FBa7E26254822B5B85a6cB319Fea81921C3786` |
| Treasury | `0xDA02a4df41581756F209e73B4aD1Ab70fC1B575e` |
| Relayer | `0xfd146695691Cbe8b85F4c5696789b0D11325e6E9` |

Funding status checked against live Coston2 RPC on 2026-08-12:

| Role | C2FLR | FTestXRP raw |
|---|---:|---:|
| Deployer | 100 | 10000000 |
| Client | 100 | 10000000 |
| Contractor | 100 | 10000000 |
| Treasury | 100 | 10000000 |
| Relayer | 100 | 10000000 |

## Current Phase 5 Status

Phase 5 is complete on Coston2 with one evidence limitation:

- `scripts/pre-build.sh` deployed the current WorkProof escrow from commit
  `4b4358ca941bf6f64926d98425b34921f09b15e7` as extension `66282`.
- WorkProofEscrow / InstructionSender:
  `0x2eA5bBb676AD142cFa24A20A9Fd950e81640E2dD`.
- Deployment transaction:
  `0xb0f71283af1ebdc8c6e0bffbe205075522f1b7e689bb6041104927c91f0851f5`.
- Source verification completed through Sourcify with status `exact_match`
  and verification job `faa71f5a-0f62-4fae-b317-befae579ffe3`.
- `scripts/post-build.sh` completed after the gateway was fixed to forward
  provider `POST /instruction` traffic to ext-proxy and ext-proxy was rebuilt
  on tee-proxy `v0.0.22`.
- The runtime `.env` was fixed to set
  `WORKPROOF_CIPHERTEXT_HOSTS=retention-pasta-clip.ngrok-free.dev`; without
  that, the verifier cannot fetch ciphertext and returns an inconclusive job.
- The live `/info` endpoint returns chain ID `114`, extension `66282`,
  simulated attestation `magic_pass`, platform `TEST_PLATFORM`, code hash
  `0x194844cf417dde867073e5ab7199fa4d21fd82b5dbe2bdea8b3d7fc18d10fdc2`,
  and signing policy `5939` at the last check.
- `getTeeMachineStatus(0x04635178b552b00d363557e1296aC71a7241eE87)` returned
  `2`.
- Stale TEE `0x242eEB8ddEA1A07Cc1A2257Be0e18EEd784dAaf3` was paused in
  transaction
  `0x60778c16151d1c9e1b71cf2cc84635b39dc306a4a99b48372569e16a3c50661b`.
- PASS/pay, FAIL/no-pay, and refund evidence is recorded in
  [`../../deployments/coston2.json`](../../deployments/coston2.json) and
  [`../evidence/demo-run.json`](../evidence/demo-run.json).
- The fresh run is still single-operator evidence. Role-separated evidence
  remains blocked until separate client, contractor, and relayer private keys
  are supplied.

## Current Open Items

1. Keep the ngrok tunnel and local gateway/proxy stack running through judging
   if a live endpoint is desired.
2. Supply/use separate client, contractor, and relayer keys if role-separated
   evidence is required before final submission.
3. Record and publish the demo video from the evidence flow.
4. Complete the DoraHacks account-level submission and store the receipt or URL.
5. After the hackathon, repeat Phase 7 on real GCP Confidential Space hardware
   before making any production hardware-attestation claim.
