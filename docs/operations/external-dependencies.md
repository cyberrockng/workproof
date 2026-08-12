# External Dependencies

Status: open — 5 of 7 confirmed, 2 still pending past the original deadline.
**Escalation deadline (2026-08-09) has arrived; the remaining Phase 5 blocker is
the Coston2 FTDC availability-check proof not appearing on the normal proxy.**
Deadline: 2026-08-08 12:00 WAT (passed)

This file tracks dependencies that cannot be completed from code alone. Do not mark WorkProof green until the required external evidence is attached or linked.

| Dependency | Owner | Required evidence | Status | Deadline |
|---|---|---|---|---|
| DoraHacks project registration | Product | Project URL and team confirmation | **Confirmed 2026-08-08** (user-reported; project URL/team confirmation not yet attached here) | 2026-08-08 12:00 WAT |
| Deployer/client/contractor/treasury/relayer wallets | Operations | Public addresses only; no private keys | **Confirmed 2026-08-12** — addresses generated 2026-08-09 (keys held locally in gitignored `.env.coston2`, never committed) and funded; see table below. | 2026-08-08 12:00 WAT |
| C2FLR and FTestXRP test funds | Operations | Faucet or transfer transaction links | **Confirmed 2026-08-12** — deployer, client, contractor, treasury, and relayer each have 100 C2FLR and 10 FTestXRP by live Coston2 RPC. | 2026-08-08 12:00 WAT |
| Coston2 indexer credentials/path | Operations | Sanitized connectivity result, no credentials | **Confirmed 2026-08-12 via fallback path** — Flare-provided hosted DB values were applied locally but `35.241.249.150:3306` refused/timed out from this network. WorkProof is running against a self-hosted `flare-system-c-chain-indexer` MySQL at `host.docker.internal:3306`, using Coston2 RPC from the Flare DevHub-listed QuickNode endpoint. `/health` returns 200 and `ext-proxy /info` is live. | 2026-08-08 12:00 WAT |
| GCP Confidential Space AMD SEV operator | Operations | Written owner confirmation and deployment handoff path | Pending — **re-scoped: not required for Phase 5.** Phase 5 runs `SIMULATED_TEE=true`/container `MODE=1` (`WORKPROOF_EXECUTION_PLAN.md:1035-1036`), no real hardware involved. Only actually blocks Phase 7 ("deploy real FCC hardware"). | Needed before Phase 7 |
| Stable HTTPS proxy route to port 6664 | Operations | Domain, TLS route, and uptime plan | **Confirmed 2026-08-13** — reserved ngrok domain `https://retention-pasta-clip.ngrok-free.dev` reaches the extension proxy `/info` and returns the WorkProof extension ID. | Needed before TEE registration |
| Flare support contact path | Product | Telegram/support thread link or summary | Pending — now needed for the Coston2 FTDC availability-check 404 if a fresh request also fails. | 2026-08-08 12:00 WAT |

## Generated wallets (public addresses only — 2026-08-09)

Real Coston2-format keypairs generated locally via `cast wallet new --json`,
each validated against `^0x[0-9a-fA-F]{40}$`/`^0x[0-9a-fA-F]{64}$` before use
(not hand-transcribed). Private keys live only in the gitignored
`.env.coston2` at the repo root — never in this file, never committed.

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

The deployer now has C2FLR for gas
(`pre-build.sh` now deploys the real `WorkProofEscrow` + registers the
extension — the pipeline previously deployed the scaffold's sample
`HelloWorldInstructionSender` instead, a real bug found and fixed during
audit remediation, see `NEW_WORK.md` "Post-audit remediation"); the client
additionally needs FTestXRP to fund a WorkProof job.

`.env.coston2` also sets `WORKPROOF_TREASURY` (the Treasury address above)
and `WORKPROOF_RPC_URL`/`WORKPROOF_RANDOM_NUMBER_V2_ADDR` for the Go
extension; `pre-build.sh` fills in `WORKPROOF_ESCROW_ADDRESS` automatically
once deployment succeeds. Re-ran `pre-build.sh` for real against live
Coston2 after this fix: it now resolves the real `FlareTeeManager` diamond
and the configured treasury/fee correctly and fails at the same known
funding blocker (0 wei balance) — confirming it never touches HelloWorld.

## Current Phase 5 status

Funding is solved. `scripts/pre-build.sh` succeeded on Coston2 on 2026-08-12:
`WorkProofEscrow` is deployed at `0x62D7AFE78bC7D8E1D8266C8C248C7Cdb35ad4EFc`
and registered as extension
`0x0000000000000000000000000000000000000000000000000000000000010281`
(decimal 66177). Public deployment evidence is recorded in
`deployments/coston2.json`.

Service startup now works through Docker Desktop from WSL by calling the Windows
Docker CLI. The first hosted-indexer attempt failed after applying the supplied
credentials because `35.241.249.150:3306` refused/timed out from this network.
The current working path is a local self-hosted C-chain indexer backed by
MySQL on `host.docker.internal:3306`. It initially lagged on the official public
RPC, then recovered after switching to the Flare DevHub-listed public QuickNode
Coston2 endpoint and smaller commit batches.

As of 2026-08-13, the stable ngrok endpoint
`https://retention-pasta-clip.ngrok-free.dev/info` returns the WorkProof
extension ID, Coston2 chain ID 114, and simulated attestation `magic_pass`.
The old `trycloudflare.com` smoke-test tunnel was stopped and must **not** be
used for on-chain TEE registration.

`scripts/post-build.sh` now partially succeeds:

- Step 1 (`allow-tee-version`) succeeded for code hash
  `0x194844cf417dde867073e5ab7199fa4d21fd82b5dbe2bdea8b3d7fc18d10fdc2`.
- Step 2 (`set-governance`) succeeded for governance hash
  `0xec29826578cec3cf256b8254484fa7faa3609dd00991a6c3628885c5d93a1b7d`.
- Step 3 (`register-tee`) reached saved state `rRa`: pre-register, fresh TEE
  attestation request, and FTDC availability-check request were all sent.

The remaining external blocker for end-to-end Phase 5 is:

1. **FTDC availability-check result** — the documented Coston2 normal proxy
   `https://tee-proxy-coston2-1.flare.rocks` stayed reachable and policy-consistent
   with on-chain reward epoch 5932, but `/action/result/<instruction>` returned
   `404 not found` for availability-check instruction
   `0xab0c8151cccaa60d1a0a5166567b85073ec80c69a12e3014774ddb4a2113bafb`
   after several minutes of polling. `config/register-tee.state` preserves the
   resumable state; if the proof appears, resume only the promotion step with
   `REGISTER_TEE_COMMAND=p` and `-resume`.

## Rules

- Never commit private keys, credentials, VPN profiles, bearer tokens, plaintext hidden vectors, or unredacted logs.
- Test indexer reachability without printing the URL if it embeds credentials.
- Record only public addresses and transaction links for wallet/funding evidence.
- Escalate if indexer path or GCP hardware access is not confirmed by 2026-08-09.
