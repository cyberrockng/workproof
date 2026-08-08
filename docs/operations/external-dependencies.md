# External Dependencies

Status: open — 1 of 7 confirmed, 6 still pending past the original deadline
Deadline: 2026-08-08 12:00 WAT (passed; escalation deadline 2026-08-09 per Rules below)

This file tracks dependencies that cannot be completed from code alone. Do not mark WorkProof green until the required external evidence is attached or linked.

| Dependency | Owner | Required evidence | Status | Deadline |
|---|---|---|---|---|
| DoraHacks project registration | Product | Project URL and team confirmation | **Confirmed 2026-08-08** (user-reported; project URL/team confirmation not yet attached here) | 2026-08-08 12:00 WAT |
| Coston2 indexer credentials/path | Operations | Sanitized connectivity result, no credentials | Pending | 2026-08-08 12:00 WAT |
| GCP Confidential Space AMD SEV operator | Operations | Written owner confirmation and deployment handoff path | Pending | 2026-08-08 12:00 WAT |
| Stable HTTPS proxy route to port 6664 | Operations | Domain, TLS route, and uptime plan | Pending | 2026-08-08 12:00 WAT |
| Deployer/client/contractor/treasury/relayer wallets | Operations | Public addresses only; no private keys | Pending | 2026-08-08 12:00 WAT |
| C2FLR and FTestXRP test funds | Operations | Faucet or transfer transaction links | Pending | 2026-08-08 12:00 WAT |
| Flare support contact path | Product | Telegram/support thread link or summary | Pending | 2026-08-08 12:00 WAT |

These six remaining items block Phase 5 (Coston2 simulated-attestation
integration) and everything after it — they are operations/product tasks,
not engineering, and cannot be resolved by writing more code. Per the Rules
below, escalate if the indexer path or GCP hardware access specifically is
not confirmed by 2026-08-09.

## Rules

- Never commit private keys, credentials, VPN profiles, bearer tokens, plaintext hidden vectors, or unredacted logs.
- Test indexer reachability without printing the URL if it embeds credentials.
- Record only public addresses and transaction links for wallet/funding evidence.
- Escalate if indexer path or GCP hardware access is not confirmed by 2026-08-09.
