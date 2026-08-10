# External Dependencies

Status: open — 2 of 7 confirmed, 5 still pending past the original deadline.
**Escalation deadline (2026-08-09) has arrived and the two hardest items
(indexer DB credentials, funded wallets) are still unresolved** — see Rules.
Deadline: 2026-08-08 12:00 WAT (passed)

This file tracks dependencies that cannot be completed from code alone. Do not mark WorkProof green until the required external evidence is attached or linked.

| Dependency | Owner | Required evidence | Status | Deadline |
|---|---|---|---|---|
| DoraHacks project registration | Product | Project URL and team confirmation | **Confirmed 2026-08-08** (user-reported; project URL/team confirmation not yet attached here) | 2026-08-08 12:00 WAT |
| Deployer/client/contractor/treasury/relayer wallets | Operations | Public addresses only; no private keys | **Addresses generated 2026-08-09** (keys held locally in gitignored `.env.coston2`, never committed) — see table below. **Not yet funded.** | 2026-08-08 12:00 WAT |
| C2FLR and FTestXRP test funds | Operations | Faucet or transfer transaction links | Pending — addresses above are ready to receive; needs a human visit to https://faucet.flare.network/coston2 (captcha-gated, cannot be automated) for at least the deployer address | 2026-08-08 12:00 WAT |
| Coston2 indexer credentials/path | Operations | Sanitized connectivity result, no credentials | Pending — confirmed genuinely blocking: `config/proxy/extension_proxy.coston2*.toml` `[db]` section needs a real `host`/`database`/`username`/`password` for `35.241.249.150:3306`, obtainable only via VPN access + Flare-team-issued credentials (`docs/deployment-steps.md:12`, `README.md:384`). No usable default exists in the shipped example config — checked directly, it is a placeholder. The sibling Ajose project hit this identical blocker independently and it remains unresolved there too as of 2026-08-06. | 2026-08-08 12:00 WAT |
| GCP Confidential Space AMD SEV operator | Operations | Written owner confirmation and deployment handoff path | Pending — **re-scoped: not required for Phase 5.** Phase 5 runs `SIMULATED_TEE=true`/container `MODE=1` (`WORKPROOF_EXECUTION_PLAN.md:1035-1036`), no real hardware involved. Only actually blocks Phase 7 ("deploy real FCC hardware"). | Needed before Phase 7 |
| Stable HTTPS proxy route to port 6664 | Operations | Domain, TLS route, and uptime plan | Pending — **re-scoped: not required for Phase 5.** Phase 5's own precondition explicitly permits "stable development tunnel only for this phase" (`WORKPROOF_EXECUTION_PLAN.md:1037`); `cloudflared` is confirmed installed locally (`~/.local/bin/cloudflared`) and can serve as that dev tunnel. The permanent route is a Phase 0/final-demo requirement ("Do not use a temporary tunnel in the final demo", `WORKPROOF_EXECUTION_PLAN.md:831`), not a Phase 5 one. | Needed before final demo |
| Flare support contact path | Product | Telegram/support thread link or summary | Pending | 2026-08-08 12:00 WAT |

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

None of these are funded. At minimum the deployer needs C2FLR for gas
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

## Two hard blockers left for Phase 5 specifically

Everything else needed for Phase 5 is now prepared and waiting:
`config/coston2/deployed-addresses.json` already exists (real Coston2
periphery addresses), Docker + Docker Compose v5.4.0 confirmed installed,
`cloudflared` confirmed installed, `.env.coston2` and
`config/proxy/extension_proxy.coston2{,.docker}.toml` are staged (gitignored,
`[db]` left as an honest placeholder — not a guess). The two items below are
the only remaining reasons Phase 5 cannot actually run:

1. **Funding** — needs a human to visit the Coston2 faucet (captcha-gated).
2. **Indexer DB credentials** — needs Flare team/Telegram to supply real
   `[db]` values for `35.241.249.150:3306`, or VPN access plus credentials.

## Rules

- Never commit private keys, credentials, VPN profiles, bearer tokens, plaintext hidden vectors, or unredacted logs.
- Test indexer reachability without printing the URL if it embeds credentials.
- Record only public addresses and transaction links for wallet/funding evidence.
- Escalate if indexer path or GCP hardware access is not confirmed by 2026-08-09.
