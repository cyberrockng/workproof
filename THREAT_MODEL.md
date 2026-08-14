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
| Stale or unintended TEE remains active | Pre/post deployment query and pause runbook; P0 operation requires exactly one active WorkProof TEE for the extension because the client encrypts the hidden bundle to the selected TEE key | Exactly one live P0 machine; stale machine paused before jobs are created |
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
| Compromised/lying RPC provider | None dedicated -- every artifact/storage/call/randomness read in `internal/verifier` goes through one configured RPC. Partially bounded, not eliminated: a lying RPC can only affect the outcome of the job it's asked about (there is no cross-job blast radius), and every value it returns is independently re-derived/recomputed by the verifier rather than trusted from the relayed instruction (artifactCodeHash, randomValueHash, on-chain job state) -- but a compromised RPC could still lie consistently enough to make the attested TEE sign a wrong PASS/FAIL for that one job | Not yet demonstrated; see Residual Risks |
| TEE clock drift | `_checkOutcomeBinding` bounds `issuedAt` to `[dispatchedAt, block.timestamp]` on-chain -- a drifted/wrong TEE clock causes safe settlement REJECTION (liveness), never acceptance of a falsely-timestamped verdict | `testMutationRejected_issuedAtBeforeDispatch`/`testMutationRejected_issuedAtInFuture` |

## Residual Risks

- FTestXRP is a testnet asset and has no monetary value.
- Real mainnet use needs an independent contract audit, operational hardening, legal review, and production custody/payment partners.
- P0 does not support subjective work, multi-transaction stateful scenarios, arbitrary native test code, or high-availability TEE secret rewrapping.
- FCC ActionResult compatibility is a mandatory spike. Backend-signed settlement is not an acceptable substitute.
- The Go verifier trusts a single configured RPC endpoint (`WORKPROOF_RPC_URL`) as its only view of chain state for a given verification. This is a real, currently-undemonstrated trust dependency, not a solved problem -- mitigating it for real (multiple independent providers with quorum/cross-check, or a light-client-verified read path) is out of P0 scope. `VerdictOutcome.IssuedAt` also uses the TEE host's real wall-clock time (`time.Now()`), which is intentionally non-deterministic across runs (it is a genuine timestamp, not a derived value) -- a mainnet operator must run NTP-synchronized clocks on TEE hosts, an operational assumption Confidential Space's attestation stack typically already satisfies.
- P0 does not include an explicit user-supplied TEE allowlist in `createJob`.
  The contract pins and verifies the production TEE returned by the registry,
  but if more than one production machine is active for the extension the
  selected TEE may differ from the key a client used for ciphertext encryption.
  That is a liveness/failure risk rather than a fund-theft path: the wrong TEE
  cannot decrypt a valid bundle and cannot pass settlement binding, but the job
  may need retry/refund. Mainnet hardening should add explicit expected-TEE
  selection or an allowlist/codeHash/governanceHash policy before job creation.
