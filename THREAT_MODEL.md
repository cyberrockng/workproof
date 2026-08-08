# WorkProof Threat Model

Status: Phase 1 freeze draft

The owner can stop new exposure but cannot decide outcomes. Emergency actions for existing jobs must be limited to allowing refunds or processing already-valid settlement.

| Threat | Control | Required evidence |
|---|---|---|
| Client refuses payment | Full pre-funding and automatic PASS settlement | PASS pays without client transaction |
| Contractor submits wrong work | Artifact, address, block, and code-hash binding | Wrong artifact FAIL; corrected artifact PASS |
| Client changes tests | Spec, plaintext bundle, ciphertext, and locator commitments fixed at creation | Mutated bundle rejected |
| Client hides unfair rules | Public template and bounds; private examples only; contractor acceptance | Unsupported bundle type rejected |
| Seed grinding | Exact future round committed at submission; deterministic insecure-round advance | Caller-supplied round impossible |
| Relayer forges verdict | Native TEE signature and exact stored TEE recovery | Mutated data and wrong key revert |
| Cross-job or cross-chain replay | Signed verdict includes chain, escrow, job, attempt, and instruction | Replay suite |
| Old valid result after retry | Current instruction generation and one-time consumption | Old proof rejected |
| Stale TEE remains active | Pre/post deployment query and pause runbook | Exactly one live P0 machine |
| TEE restart loses decryption key | Drain before rotate; no silent rewrap | Rotation drill |
| Proxy returns 404 | Trace instruction, proxy queue, TEE logs, then retry only after timeout | Runbook and monitoring |
| Indexer DB unreachable | Port/VPN preflight and alert before registration | Connectivity check recorded |
| Ciphertext gateway swaps content | Ciphertext hash plus plaintext commitment | Changed blob rejected |
| SSRF through bundle locator | Fixed HTTPS host allowlist; no redirects to private networks | SSRF tests |
| Malicious contract exhausts verifier | RPC-only execution with gas, time, size, and response caps | Timeout becomes INCONCLUSIVE |
| Admin steals funds | No admin settlement or accounted-principal withdraw path | ABI and access-control review |
| Frontend lies | Chain events and explorer are source of truth | Evidence page links raw records |
| Secret leaks | No plaintext in logs, chain, reports, analytics, or screenshots | Secret scan and canary test |
| Token incompatibility | P0 accepts only resolved FTestXRP | Wrong token rejected |
| Reentrancy/nonstandard ERC20 | SafeERC20, ReentrancyGuard, and state-first accounting | Malicious-token tests |

## Residual Risks

- FTestXRP is a testnet asset and has no monetary value.
- Real mainnet use needs an independent contract audit, operational hardening, legal review, and production custody/payment partners.
- P0 does not support subjective work, multi-transaction stateful scenarios, arbitrary native test code, or high-availability TEE secret rewrapping.
- FCC ActionResult compatibility is a mandatory spike. Backend-signed settlement is not an acceptable substitute.
