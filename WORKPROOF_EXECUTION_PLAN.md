# WorkProof — Audited A–Z Execution Plan

**Plan version:** 1.0  
**Plan date:** 7 August 2026 (WAT)  
**Hackathon:** Flare Summer Signal  
**Primary bounty:** Bounty 2 — Confidential Compute Apps  
**Final submission deadline:** 14 August 2026 (organizer page does not expose a reliable timezone; internal submission target is **14 August, 12:00 WAT**)  
**Internal code freeze:** 13 August 2026, 18:00 WAT  
**Status:** Executable, source-checked, and risk-audited. No implementation has been claimed in this document.

---

## 1. Executive decision

Build **WorkProof**, a pre-funded payment platform for objectively testable digital work.

> A client locks payment, a contractor submits the work, Flare Confidential Compute privately runs the agreed tests, and the smart contract releases payment only when the work passes.

The first real, honest vertical is **deployed smart-contract work on Coston2**. WorkProof will verify immutable runtime bytecode and deterministic behavioral tests. It will not claim to judge designs, writing, business quality, or arbitrary software during the hackathon.

The one-glance product flow is:

1. **Lock** — the client pre-funds the job with FTestXRP.
2. **Verify privately** — the contractor submits a deployed contract; FCC runs committed private test vectors selected using secure Flare randomness.
3. **Pay automatically** — a TEE-signed PASS releases the funds; FAIL leaves the principal untouched and permits resubmission.

### Why this project deserves to exist

Businesses delay or dispute payment because they cannot cheaply establish whether outsourced digital work satisfies the agreed requirements. Contractors fear completing work and not being paid. WorkProof moves both risks into a deterministic agreement:

- the client cannot refuse a valid payment after PASS;
- the contractor cannot receive payment for a failed delivery;
- neither party can change the test commitment after accepting the job;
- no participant, guarantor, insurer, or remaining group absorbs another person's loss;
- the principal is fully funded before work begins.

### Defensible differentiation

Escrow products already exist, automated test services already exist, and TEE
demos already exist. WorkProof's defensible unit is the combination judges can
verify in one transaction path:

- pre-funded XRP-denominated work escrow;
- private examples of publicly agreed requirements;
- secure future-round selection that neither party can choose after submission;
- a result signed by the actual FCC machine and verified inside the payment contract;
- artifact, specification, randomness, verifier version, and instruction bound into one proof;
- fail safely, resubmit, and pay without giving an administrator outcome authority.

The novelty claim is this composed trust-minimized workflow, not the false claim
that escrow, software testing, or confidential computing was invented here.

### What is economically real and what is not

- The escrow rules, token transfers, gas expenditure, FCC execution, and balance changes will be real on Coston2.
- FTestXRP is a testnet asset and has **no monetary value**. The demo proves the economic mechanism, not production custody of real money.
- Mainnet use requires an external contract audit, operational hardening, legal review, and production partners. The hackathon submission must state this plainly.

### Win thesis

WorkProof is competitive only if the final evidence proves all of the following at the same time:

- a useful two-sided economic problem;
- meaningful FCC privacy, not an FCC label;
- secure Flare randomness used in the decision process;
- official FTestXRP as the escrow asset;
- a real hardware TEE with production attestation;
- automatic on-chain settlement from a contract-verifiable TEE result;
- a clean, understandable failure-then-fix-then-payment demonstration;
- at least one external person completing or observing the actual product flow.

No plan can guarantee a judge's decision. This plan maximizes credibility by making every important claim independently verifiable.

---

## 2. Non-negotiable release gates

WorkProof is submission-ready only when every P0 gate below is green. A screenshot, mocked badge, local-only result, or team assertion does not satisfy a gate.

| Gate | Required evidence | Pass condition |
|---|---|---|
| G1 — Real escrow | Coston2 contract, verified source, FTestXRP transfers | Client funds 100%; PASS pays contractor; expiry refunds client |
| G2 — Official asset | Registry-resolution log and deployment manifest | `AssetManagerFXRP` is resolved through Flare's registry and its `fAsset()` is used; no mock token in the real deployment |
| G3 — Secure randomness | Coston2 events and transaction | Exact future voting round is committed before the seed is known; historical random reports `isSecure=true` |
| G4 — Native TEE proof | Solidity/Go cross-language test vector | Contract reconstructs the official FCC `ActionResult` signing digest and recovers the expected TEE machine address |
| G5 — Real hardware FCC | Public `/info`, systems explorer, image digest | Platform is GCP AMD SEV, `MODE=0`, `SIMULATED_TEE=false`, non-simulated code hash, correct extension ID, machine in production |
| G6 — Artifact binding | On-chain attempt plus verifier report | Result binds job, attempt, instruction, chain, target address, block, runtime code hash, spec hash, random round, and engine version |
| G7 — Automatic settlement | Explorer transactions and balance deltas | Permissionless relayer submits an authentic PASS; contract pays without a client/admin approval transaction |
| G8 — Negative path | Test and explorer evidence | A deliberately failing artifact cannot release principal; replay, altered verdict, wrong TEE, and wrong artifact all revert |
| G9 — Recoverability | Tested timeout and expiry path | Proxy/TEE failure moves no principal; a timed-out verification can be retried; an expired job refunds only the client |
| G10 — Usable product | Live app and external pilot record | A non-team user understands Lock → Verify → Pay and completes the scripted flow without developer intervention |
| G11 — Submission evidence | Public repository, video, deployment manifest | Judges can reproduce the claims and find every address, transaction, hash, and limitation |

**Submission stop rule:** if G4 or G5 is not green by 11 August, 18:00 WAT, do not present a simulated or centrally signed version as WorkProof. Repair the real path or stop this submission.

---

## 3. Hackathon alignment and evidence map

The organizer lists two bounties. WorkProof should submit to **Bounty 2 — Confidential Compute Apps**. FXRP strengthens the product but is a payment rail, not enough reason to dilute the entry across two bounties.

| Judging criterion | WorkProof answer | Evidence judges receive |
|---|---|---|
| Product usefulness | Removes payment and acceptance risk from outsourced smart-contract work | Two-party pilot, fail/pass flow, real escrow balances |
| Flare integration quality | FCC protects hidden test vectors; secure randomness prevents chosen-test manipulation; FAssets supplies XRP-denominated payment | Extension ID, TEE ID, code hash, random round, FTestXRP address, explorer links |
| Technical execution | On-chain state machine verifies the FCC machine's native result signature before settlement | Source, test vectors, verified contracts, real E2E transactions |
| Evidence of new work | WorkProof contracts, verifier DSL, Go FCC handler, relayer, UI, evidence tooling | Git history after the pinned scaffold base and a `NEW_WORK.md` ledger |
| Clarity and future potential | Lock → Verify privately → Pay; first vertical expands through verifier adapters | One-screen demo, honest roadmap, pilot interviews |

### Required final submission fields

- Project name: WorkProof.
- Selected bounty: Confidential Compute Apps.
- Short product description and target users.
- Working application and short video.
- Public GitHub repository and technical architecture.
- Explanation of why FCC, randomness, and FXRP are necessary.
- New-work statement with pinned scaffold base.
- Coston2 contract addresses, extension ID, TEE ID, code hash, image digest, proxy URL, and transaction links.
- User-testing or pilot evidence.
- Honest limitations and roadmap.

---

## 4. Product scope

### P0 — real hackathon release

P0 is one complete product loop, not several shallow integrations:

- specified client and contractor wallet addresses;
- one objective milestone per on-chain job;
- full principal pre-funding with official Coston2 FTestXRP;
- a refundable success fee locked in addition to the contractor's principal;
- contractor acceptance;
- committed public requirements plus a private, encrypted acceptance bundle;
- deployed Coston2 smart-contract artifact;
- immutable runtime bytecode hash and submission block;
- secure future-round random seed;
- FCC Go verifier on a real GCP Confidential Space machine;
- native TEE-signed result verified by the escrow contract;
- PASS, FAIL, INCONCLUSIVE, timeout, cancellation-before-acceptance, and expiry/refund paths;
- permissionless result delivery;
- web application, event indexer, evidence page, and operator runbook;
- at least two demo artifacts: one intentionally wrong and one corrected.

### P1 — implement only after every P0 gate is green

- off-chain project grouping for multiple milestones;
- non-transferable Work Receipts derived from completed jobs;
- redundant permissionless relayer instance;
- downloadable verification report;
- public template library for standard contract interfaces;
- email/Telegram status notifications that do not control funds.

### P2 — post-hackathon company scope

- stateful EVM sandbox with strict resource isolation;
- API, webhook, container, WASM, and AI-agent delivery adapters;
- mainnet FXRP and approved stablecoin rails;
- milestone dependency graphs;
- organization accounts, team roles, accounting exports, and invoicing;
- high-availability multi-TEE secret rewrapping and result quorum;
- named arbitration providers for subjective or ambiguous work;
- marketplace and freelance-platform SDKs;
- compliance, audit, monitoring, insurance options, and regulated custody/payment partners where required.

### Explicit non-goals for P0

- judging subjective quality;
- an AI agent deciding who receives money;
- converting stablecoins to XRP or operating a bridge/exchange;
- lending, credit, pooled loss absorption, or insurance;
- source-code ownership or intellectual-property adjudication;
- reversible blockchain payments;
- production mainnet custody;
- executing arbitrary native binaries inside the TEE;
- claiming every type of digital work is already supported.

---

## 5. User and job contract

### Target customers

The initial buyer is a business or protocol hiring a contractor to deliver a smart contract with objective behavior. The contractor is an independent developer or software agency. Later buyers include bounty platforms, grant programs, freelance marketplaces, and AI-agent marketplaces.

### Job terms visible before acceptance

Every job must show:

- client and contractor;
- exact principal the contractor receives after PASS;
- separate protocol success fee paid by the client;
- who pays each FCC/network attempt fee;
- acceptance deadline, work deadline, and verification grace period;
- public requirement template and parameter bounds;
- number and types of public/private test vectors;
- private bundle commitment;
- target chain and artifact type;
- retry policy;
- objective outcome rules;
- selected TEE ID and verifier version;
- whether the job is objective-only or uses a named arbitrator. P0 supports objective-only.

### Fair hidden tests

“Private tests” must mean **private examples of public rules**, not secret rules.

- The requirement template, input domains, pass threshold, and test types are public.
- The hidden bundle contains test inputs and expected outputs generated from those public rules.
- The client cannot add a new rule after contractor acceptance.
- Both parties accept the same `specHash` and bundle commitment.
- P0 uses approved WorkProof templates; arbitrary client-supplied native test code is rejected.
- A contractor can run public smoke vectors before accepting.

This restriction prevents a client from hiding an impossible condition and using FCC to make the unfairness invisible.

---

## 6. Economic model and loss allocation

### Funds

For a job with contractor principal `P` and protocol fee rate `r`:

- client deposits `P + (P × r)` in FTestXRP;
- `P` is always reserved for the contractor or returned to the client;
- the fee is earned only after an authentic PASS;
- cancellation before contractor acceptance returns principal and fee;
- final failure or expiry returns principal and fee to the client;
- every FCC/native network execution fee is paid explicitly by the transaction sender and is not taken from another participant's principal;
- no other participant absorbs a loss.

Recommended beta price after the hackathon: 1% success fee, with a minimum verification fee based on actual compute cost. The hackathon contract should make the fee configurable only for **future** jobs and cap it in code. Existing job economics must be immutable.

### Outcome table

| Event | Principal | Success fee | Attempt/network cost | Next state |
|---|---|---|---|---|
| Contractor has not accepted; client cancels | Client | Client | Client's prior gas | Cancelled |
| PASS from expected TEE | Contractor | Treasury | Already paid by attempt sender | Paid |
| FAIL before deadline | Locked | Locked | Attempt sender | Awaiting resubmission |
| INCONCLUSIVE or infrastructure error | Locked | Locked | Attempt sender; no principal movement | Retryable |
| Timely submission still verifying at deadline | Locked during fixed grace period | Locked | Attempt sender | Verifying |
| No valid PASS by grace-period end | Client | Client | Already spent fees remain spent | Refunded |
| Relayer disappears | Locked | Locked | None until another caller relays | Anyone can relay |
| TEE/proxy disappears | Locked | Locked | Failed caller's gas only | Timeout then retry/refund |

### Invariants

The smart contracts must enforce these properties under unit, fuzz, and invariant testing:

1. The contract's accounted FTestXRP balance equals the sum of locked principals and locked fees.
2. A job pays at most once.
3. Principal can leave only to the named contractor after PASS or to the original client after a valid cancellation/refund.
4. No owner/admin function can mark PASS, replace a verdict, redirect principal, or change accepted terms.
5. An attempt result is accepted at most once.
6. FAIL and INCONCLUSIVE never transfer principal.
7. Fee changes do not affect existing jobs.
8. Pausing stops new jobs and attempts, not valid refunds or already-authorized settlement.
9. A stale, wrong, or unregistered result cannot release funds.
10. Total outgoing principal never exceeds total funded principal.

---

## 7. System architecture

```text
Client browser
  ├─ creates public requirements and private vector bundle
  ├─ encrypts private bundle to the live FCC TEE public key
  └─ funds job with FTestXRP
            │
            ▼
WorkProofEscrow + InstructionSender (Coston2)
  ├─ holds principal and success fee
  ├─ commits job/spec/artifact/random round
  ├─ selects/stores expected production teeId
  ├─ sends VERIFY instruction through TeeExtensionRegistry
  └─ verifies the TEE-signed ActionResult before settlement
            │
            ▼
Flare FCC routing → ext-proxy → GCP Confidential Space / AMD SEV
  ├─ decrypts committed bundle inside TEE
  ├─ re-reads artifact and job facts from Coston2
  ├─ deterministically selects test vectors from secure random seed
  ├─ runs bounded verification
  └─ returns byte-exact ActionResult signed by the TEE machine key
            │
            ▼
Permissionless relayer
  ├─ polls /action/result/<instructionId>
  ├─ submits result + native TEE signature to WorkProofEscrow
  └─ cannot alter result or choose recipient
            │
            ▼
PASS → FTestXRP contractor payment
FAIL → no payment, resubmission remains possible
```

### Components

| Component | Technology | Responsibility | Trust level |
|---|---|---|---|
| `WorkProofEscrow.sol` | Solidity/Foundry | Job state, funding, randomness lock, FCC dispatch, result verification, settlement/refund | Controls funds; must be minimal and audited |
| FCC registry interfaces | Official scaffold interfaces | Select production TEE and send instruction | Flare infrastructure |
| WorkProof extension | Go | Decode, validate, decrypt, fetch state, run deterministic test engine, encode verdict | Attested code |
| Test bundle SDK | TypeScript | Canonicalize, commit, encrypt, upload, and preview templates | Client-side; cannot settle funds |
| Ciphertext gateway | Content-addressed HTTPS object storage | Stores only encrypted bundles and reports | Availability dependency; no plaintext |
| Relayer | TypeScript/Node | Poll signed result and submit it on-chain | Untrusted and replaceable |
| Web app | Next.js, viem/wagmi | Wallet actions, job creation, timeline, balances, evidence | Untrusted UI; chain is source of truth |
| Event indexer | viem polling | Read contract events for UI | Read-only; UI must tolerate lag |
| Evidence exporter | Node script | Produce machine-readable deployment/demo manifest | Read-only |

### Repository layout

Use the official FCC scaffold as the pinned base rather than recreating FCC infrastructure.

```text
workproof/
├─ contracts/
│  ├─ WorkProofEscrow.sol
│  ├─ ResultVerifier.sol
│  ├─ interfaces/
│  ├─ script/
│  └─ test/
├─ go/
│  ├─ internal/config/
│  ├─ internal/extension/
│  ├─ internal/verifier/
│  ├─ pkg/types/
│  └─ tools/cmd/run-test/
├─ packages/
│  ├─ schema/
│  └─ test-bundle-sdk/
├─ relayer/
├─ web/
├─ config/
├─ deployments/
├─ docs/
│  ├─ architecture/
│  ├─ evidence/
│  ├─ operations/
│  └─ security/
├─ testdata/
│  ├─ cross-language/
│  └─ simulation-only/
├─ scripts/
├─ docker-compose.yaml
├─ NEW_WORK.md
└─ README.md
```

---

## 8. On-chain design

### Core records

`Job` must contain at least:

- `jobId`;
- `client` and `contractor`;
- `paymentToken`;
- `principal` and locked success fee;
- `createdAt`, `acceptBy`, `submitBy`, and `verificationGraceEnds`;
- `publicSpecHash`, `privateBundleHash`, and content-addressed bundle locator;
- public template ID and template version;
- `targetChainId` and artifact type;
- `expectedTeeId`, FCC `extensionId`, and verifier version hash;
- current job status and attempt number.

`Attempt` must contain at least:

- job and attempt number;
- artifact address;
- submission block and `extcodehash`;
- target random voting round;
- locked random value hash and security flag;
- instruction ID and expected TEE ID;
- dispatch time and timeout;
- result report hash and outcome;
- settlement flag.

### Required external functions

Names may change during implementation, but behavior may not.

| Function | Caller | Required behavior |
|---|---|---|
| `createJob(...)` | Client | Validate terms, select/confirm the only active P0 TEE, transfer principal + fee, emit full commitment |
| `acceptJob(jobId)` | Named contractor | Accept immutable terms before `acceptBy` |
| `cancelUnaccepted(jobId)` | Client | Return all locked tokens only if contractor never accepted and acceptance window is valid/expired |
| `submitAttempt(jobId, artifact)` | Contractor | Require accepted job and deadline; record runtime code hash and deterministic future random round |
| `lockRandomness(jobId, attempt)` | Anyone | Fetch only the committed historical round; accept only secure randomness; deterministically advance after insecure round |
| `dispatchVerification(jobId, attempt)` | Anyone, payable | Select/confirm expected production TEE, store instruction ID, send exact job payload, start timeout |
| `settleAttempt(actionResult, teeSignature)` | Anyone | Verify native FCC signature and all bindings; transition state; transfer only on PASS |
| `expireVerification(jobId, attempt)` | Anyone | Make a timed-out instruction retryable without moving principal; invalidate old instruction generation |
| `refundExpired(jobId)` | Anyone | After deadline + grace, return principal and fee only to client |
| `pauseNewWork()` | Emergency role | Stop new jobs/attempts only; must not block refunds or a valid already-dispatched result |

### Events

Emit indexed events sufficient to reconstruct the whole product without a private database:

- `JobCreated`;
- `JobAccepted`;
- `JobCancelled`;
- `AttemptSubmitted`;
- `RandomRoundCommitted`;
- `RandomnessLocked`;
- `InsecureRandomRoundSkipped`;
- `VerificationDispatched`;
- `VerificationTimedOut`;
- `AttemptSettled`;
- `PaymentReleased`;
- `JobRefunded`;
- `ProtocolFeeChangedForFutureJobs`;
- `NewWorkPaused` and `NewWorkUnpaused`.

### Contract controls

- OpenZeppelin `SafeERC20`, `ReentrancyGuard`, and explicit custom errors.
- Checks-effects-interactions before token transfers.
- No upgradeable proxy during the hackathon. Deploy immutable, reviewable contracts.
- Owner may pause new exposure and set a capped fee for future jobs.
- Owner may not mutate jobs, substitute TEE results, change a job's token, or withdraw accounted principal.
- Any accidentally sent unsupported token recovery must prove it is not an accounted payment token balance.
- Constructor rejects zero addresses and contracts without code.
- Compile for Cancun as required by Flare guidance.

---

## 9. Contract-verifiable FCC result

This is WorkProof's most important technical gate.

The FCC node returns an `ActionResponse` containing:

- `result`: the byte-exact `ActionResult`;
- `signature`: the TEE machine signature;
- `proxySignature`: a proxy field that WorkProof does not trust for settlement.

Only `result.data`, `result.id`, `result.submissionTag`, and `result.status` are
covered by the current official `ActionResult.Hash()` formula. Fields such as
`opType`, `opCommand`, `version`, and `log` are useful consistency checks but are
**not settlement authority**. Every security-critical binding therefore lives
inside the signed `VerdictV1` data as well as contract storage.

The current official node computes:

```text
ActionResultHash = keccak256(
  keccak256(result.data)
  || result.id
  || keccak256(bytes(result.submissionTag))
  || uint8(result.status)
)

SigningPayload = FCC domain payload(
  TEEActionResult,
  chainId,
  ActionResultHash
)

TEE signature = secp256k1 signature over the Ethereum signed-message form
                of SigningPayload
```

The implementation must copy the exact domain construction from the pinned `go-flare-common` version and prove equivalence with a cross-language test vector. Do not guess a prefix or handwave this step.

### Settlement verification order

`settleAttempt` must perform every check before changing job state:

1. Job and attempt exist and are currently verifying.
2. Result has not been consumed and current instruction generation has not timed out.
3. `result.status == 1`; FCC handler failures or pending results never settle.
4. `opType == bytes32("WORKPROOF")` and `opCommand == bytes32("VERIFY")` as
   routing-consistency checks only; never rely on them instead of signed verdict data.
5. Reconstructed digest matches the signature.
6. Recovered signer equals the attempt's stored `expectedTeeId`.
7. Decoded verdict version is supported.
8. Verdict chain ID, escrow address, job ID, attempt, instruction ID, and expiry match storage.
9. Verdict spec hash, private bundle hash, artifact address, artifact block, runtime code hash, random round/hash, and engine version match storage.
10. Verdict outcome is exactly PASS, FAIL, or INCONCLUSIVE.
11. Report hash is nonzero and verdict is not replayed.
12. Mark settled before any external token transfer.

### Mandatory spike: FCC signature compatibility

Complete this before building the full escrow:

1. Use the scaffold-pinned FCC node packages to create a reference `ActionResult` in Go.
2. Sign it with a fixed test key through the same `SignResult` path used by the node.
3. Save all fields, intermediate hashes, signature, and expected address as a JSON test vector.
4. Write a Foundry test that reconstructs the digest and recovers the same address.
5. Mutate every result field one at a time and prove recovery/checks fail.
6. Repeat with Coston2 chain ID 114 and a second chain ID to prove cross-chain replay fails.

**Gate:** four hours maximum. If this cannot be made byte-exact against the pinned official packages, stop the automatic-settlement design and resolve it with Flare maintainers. A backend-signed substitute is not acceptable.

### Why the relayer is not trusted

The relayer only transports `(ActionResult, teeSignature)` from the public proxy to the contract. It cannot:

- change PASS to FAIL or FAIL to PASS;
- change the contractor, amount, job, artifact, or random seed;
- sign as the stored TEE ID;
- settle a consumed or timed-out instruction;
- redirect payment.

Anyone can run the same relay operation. If the hosted relayer fails, the UI offers “Relay result” and a CLI accepts the instruction ID.

---

## 10. Randomness design

Flare secure randomness is used to prevent either party from choosing a favorable hidden-vector subset after seeing the artifact.

### Algorithm

1. Contractor submits the artifact before the seed is known.
2. The contract records `targetRound = Relay.getVotingRoundId(block.timestamp) + 1`.
3. No caller can provide a different round.
4. After finalization, `lockRandomness` calls `getRandomNumberHistorical(targetRound)`.
5. If the call reverts because the round is not ready, storage does not change.
6. If `isSecure == false`, emit `InsecureRandomRoundSkipped` and set `targetRound = targetRound + 1`.
7. Only a secure historical random value is locked.
8. Derive:

```text
testSeed = keccak256(
  randomNumber,
  address(WorkProofEscrow),
  jobId,
  attempt,
  specHash,
  artifactCodeHash
)
```

9. Store the round and `keccak256(randomNumber)` on-chain; include both and the seed in the FCC instruction.
10. The extension re-reads the job/attempt at the dispatch block and rejects a mismatch.

### Anti-grinding rules

- Only one live attempt per job.
- A caller cannot skip a secure round.
- Resubmission requires a new artifact commitment and a new attempt fee.
- INCONCLUSIVE retry advances according to a deterministic rule and invalidates the old instruction generation.
- The verifier uses deterministic Fisher-Yates selection over a fixed committed vector list.
- All mandatory vectors must pass; randomness changes selection/order, not the public acceptance rule.

### Address resolution

Use Flare's `ContractRegistry`/Flare Contract Registry at deployment or runtime. Do not permanently paste a current protocol address into business logic. Record every resolved address in `deployments/coston2.json` with the resolution transaction/block.

---

## 11. Private acceptance bundle and verifier

### Bundle format

Use versioned canonical JSON (RFC 8785/JCS) or an equally deterministic encoding. The same canonicalizer must be tested in TypeScript and Go.

Minimum fields:

```text
formatVersion
templateId
templateVersion
targetChainId
publicSpecHash
vectorCount
selectionCount
gasLimitPerCall
timeoutMsPerCall
maxResponseBytes
vectors[]
```

Each P0 vector is data, not executable native code. Supported P0 vector types:

- `ETH_CALL_EQUALS` — calldata, caller/value, expected return bytes;
- `ETH_CALL_REVERTS` — calldata and expected revert selector/pattern;
- `ERC165_SUPPORTS_INTERFACE` — expected interface support;
- `CODE_SIZE_RANGE` — bounded runtime bytecode size;
- `STORAGE_AT_EQUALS` — optional fixed-slot assertion at the submission block.

P0 does not claim stateful multi-transaction execution. Add that only after a resource-isolated deterministic EVM sandbox exists.

### Commitment and encryption

1. Canonicalize plaintext bundle.
2. Compute `privateBundleHash = keccak256(canonicalPlaintext)`.
3. Query the production proxy `/info`; validate platform, code hash, extension ID, owner, TEE ID, and public key.
4. Query active machines for the extension; P0 requires exactly one live production machine.
5. Encrypt the canonical bundle using the FCC ECIES flow compatible with the node's decrypt endpoint.
6. Upload ciphertext to a content-addressed HTTPS location.
7. Compute and store `ciphertextHash` and locator.
8. `createJob` commits plaintext hash, ciphertext hash, locator, and `expectedTeeId`.
9. The FCC extension downloads ciphertext, checks `ciphertextHash`, decrypts through the node's local `/decrypt` interface, canonicalizes again, and checks `privateBundleHash`.
10. Plaintext never enters a transaction, proxy log, browser analytics event, report, or public object.

### TEE rotation consequence

The encrypted bundle is bound to the live TEE key. Confidential Space does not persist that key across relaunches. Therefore:

- stop new job creation before a TEE restart;
- drain or resolve all jobs encrypted to the old TEE;
- pause the stale on-chain machine after confirming the new live ID;
- re-encrypt only jobs whose parties explicitly authorize rewrapping;
- never silently point an existing secret at a new machine.

High-availability secret rewrapping is post-hackathon work, not a hidden assumption.

### Deterministic result schema

The extension's `result.data` should ABI-encode a versioned `VerdictV1`:

```text
schemaVersion
escrowAddress
chainId
jobId
attempt
instructionId
specHash
privateBundleHash
artifactAddress
artifactBlock
artifactCodeHash
randomRound
randomValueHash
engineVersionHash
outcome              // PASS | FAIL | INCONCLUSIVE
passedCount
executedCount
reportHash
issuedAt
expiresAt
```

The public report exposes vector IDs, statuses, timings, and redacted diagnostics. It must not expose hidden inputs or expected outputs.

### Resource limits

- maximum bundle and ciphertext bytes;
- maximum vector count and selected count;
- per-RPC-call timeout;
- total attempt timeout;
- fixed call gas cap;
- response-size cap;
- strict URL scheme and host allowlist for the ciphertext gateway and Coston2 RPC;
- no redirects to private network addresses;
- no arbitrary shell execution;
- no dynamic code download;
- bounded logs with secret redaction;
- deterministic ordering and no wall-clock-dependent verdict logic.

---

## 12. State machine

```text
CREATED/FUNDED
  ├─ client cancel before acceptance ───────────────► CANCELLED → client refund
  └─ contractor accepts ───────────────────────────► ACCEPTED
                                                       │
                                                       ▼
                                             AWAITING_SUBMISSION
                                                       │ artifact committed
                                                       ▼
                                             RANDOMNESS_PENDING
                                                       │ secure round locked
                                                       ▼
                                                READY_TO_VERIFY
                                                       │ instruction sent
                                                       ▼
                                                   VERIFYING
                    ┌──────────────────────────────────┼─────────────────────────┐
                    │ PASS                             │ FAIL                    │ INCONCLUSIVE/TIMEOUT
                    ▼                                  ▼                         ▼
              PAID (terminal)               AWAITING_RESUBMISSION          RETRYABLE
                                                       │                         │
                                                       └──────────┬──────────────┘
                                                                  │ before deadline
                                                                  ▼
                                                       AWAITING_SUBMISSION

Any non-paid accepted job after submission deadline + verification grace
  ────────────────────────────────────────────────────► REFUNDED (terminal)
```

### Deadline rules

- The contractor must submit before `submitBy`.
- A timely submission gets a fixed verification grace period, initially 30 minutes for testnet.
- A PASS from the current instruction received during grace pays even if `submitBy` has passed.
- After grace, anyone can expire the verification and refund the client.
- An old result arriving after timeout or instruction replacement is rejected.
- Blockchain timestamp is the only deadline clock used for funds.

---

## 13. Threat model

| Threat | Control | Required test/evidence |
|---|---|---|
| Client refuses payment | Full pre-funding and automatic PASS settlement | PASS pays without client transaction |
| Contractor submits wrong work | Artifact/address/code-hash binding and objective vectors | Wrong artifact FAIL; correct artifact PASS |
| Client changes tests | Plaintext/bundle/spec commitments fixed at job creation | Mutated bundle rejected |
| Client hides unfair rules | Public template/bounds; private examples only; contractor acceptance | Unsupported bundle type rejected |
| Seed grinding | Future exact round committed at submission; deterministic insecure-round advancement | Caller-supplied round impossible |
| Relayer forges verdict | Native TEE signature and exact `teeId` recovery | Mutated data and wrong key revert |
| Cross-job/chain replay | Domain includes chain, escrow, job, attempt, instruction | Replay suite |
| Old valid result after retry | Current instruction generation and one-time consumption | Old proof rejected |
| Stale TEE remains active | Pre/post deployment query and pause runbook | Exactly one live P0 machine |
| TEE restart loses decryption key | Drain-before-rotate; no silent rewrap | Rotation drill |
| Proxy returns 404 | Trace instruction → proxy queue → TEE logs → result; retry only after timeout | Runbook and monitoring |
| Indexer DB unreachable | Port/VPN preflight and alert before registration | Connectivity check recorded |
| Ciphertext gateway swaps content | `ciphertextHash` and plaintext commitment | Changed blob rejected |
| SSRF through bundle locator | Fixed HTTPS host allowlist, no redirects/private IPs | SSRF tests |
| Malicious contract exhausts verifier | RPC-only P0 execution, gas/time/size caps | Timeout becomes INCONCLUSIVE, no payout |
| Admin steals funds | No admin settlement/withdraw path; invariant tests | ABI and access-control review |
| Frontend lies | Events and explorer are source of truth | Evidence page links raw chain records |
| Secret leaks | No plaintext logging/analytics/on-chain fields; secret scans | Log inspection and canary secret test |
| Token incompatibility | P0 allows only resolved FTestXRP | Wrong token rejected |
| Reentrancy/nonstandard ERC20 | SafeERC20 + guard + state-first accounting | malicious-token unit tests where applicable |

### Admin policy

The owner can stop creating new risk but cannot decide outcomes. Any emergency action affecting existing jobs must be limited to allowing refunds, never redirecting funds or declaring PASS.

---

## 14. Environment and source lock

### Required environment

- Windows Docker Desktop in Linux-container mode;
- Git Bash or WSL for official shell scripts; use one shell consistently;
- Git;
- Docker/Compose;
- Go 1.25.1 or later compatible version required by the scaffold;
- Foundry (`forge`, `cast`);
- `jq`;
- Node.js 22 LTS and `pnpm`;
- funded Coston2 deployer and separate client/contractor test wallets;
- Coston2 indexer read-only access/VPN supplied by Flare;
- a stable public HTTPS proxy URL;
- a GCP Confidential Space operator/project capable of AMD SEV production attestation;
- Coston2 C2FLR and FTestXRP test funds.

### Preflight commands

Run in Git Bash from the repository root:

```bash
git --version
docker --version
docker compose version
go version
forge --version
cast --version
jq --version
node --version
pnpm --version
bash --version
```

Save sanitized output to `docs/evidence/toolchain.txt`. Do not include usernames, private paths, credentials, or keys.

### Pinned upstream

The audited scaffold head on 7 August 2026 is:

```text
flare-foundation/fce-extension-scaffold
ffb6c4ca7c160c49be59e00fe537e24d2477b000
```

Bootstrap the currently empty repository from that exact commit:

```bash
git remote get-url scaffold >/dev/null 2>&1 || \
  git remote add scaffold https://github.com/flare-foundation/fce-extension-scaffold.git
git fetch scaffold main
git switch -c codex/workproof ffb6c4ca7c160c49be59e00fe537e24d2477b000
git tag scaffold-base-2026-08-07 ffb6c4ca7c160c49be59e00fe537e24d2477b000
```

Then:

- keep the scaffold's pinned `go.mod`, `go.sum`, and proxy dependency refs;
- record any deliberate dependency update in `docs/security/dependency-changes.md`;
- never build production from a floating `main` branch;
- set `SOURCE_DATE_EPOCH` from the frozen Git commit;
- deploy the exact image digest tested, not a mutable tag.

### Secrets policy

- `.env`, deployer keys, indexer credentials, cloud credentials, and private test plaintext are never committed.
- Provide sanitized `.env.example` files only.
- Use separate deployer, client, contractor, treasury, and relayer wallets.
- Use testnet-only keys during the hackathon.
- Rotate any key ever pasted into chat, a ticket, a log, or a screenshot.
- Enable repository secret scanning before the first public push.
- Redact proxy logs and never return configuration secrets from `/info` or application APIs.

---

## 15. Execution phases

Each phase has an owner role, commands/actions, artifacts, tests, and a hard exit condition. If one person executes the project, follow the roles sequentially in P0 order rather than multitasking.

**Role legend:** Product (P), Contracts (C), FCC/Go (F), App/Relayer (A), Operations (O), QA/Security (Q).

### Phase 0 — unblock external dependencies

**Owners:** P, O  
**Deadline:** 8 August, 12:00 WAT

Actions:

1. Confirm the official DoraHacks project is registered under the correct team.
2. Ask Flare support/Telegram for current Coston2 indexer credentials and permitted network path.
3. Confirm who will deploy the image to a real GCP Confidential Space AMD SEV VM.
4. Reserve a stable HTTPS domain/proxy route to port 6664. Do not use a temporary tunnel in the final demo.
5. Fund five testnet wallets with enough C2FLR; obtain FTestXRP for the client.
6. Record owners and response deadlines in `docs/operations/external-dependencies.md`.
7. Test network reachability to the indexer without printing credentials.

Artifacts:

- external dependency checklist;
- wallet role/address list with no private keys;
- stable-domain plan;
- screenshot or written confirmation of real hardware deployment access.

Exit condition:

- indexer path and GCP owner are confirmed. If not confirmed by 9 August, mark WorkProof red and escalate; do not discover this after the code is finished.

### Phase 1 — repository and specification freeze

**Owners:** P, C, F, Q  
**Deadline:** 8 August, 15:00 WAT

Actions:

1. Bootstrap from the pinned scaffold commit.
2. Add the repository layout in section 7.
3. Write `SPEC.md` containing job terms, state transitions, outcome table, and P0 test types.
4. Write `THREAT_MODEL.md` from section 13.
5. Create `NEW_WORK.md` separating scaffold code from WorkProof code.
6. Define shared `VerdictV1` and bundle schemas before contract/Go implementation.
7. Generate TypeScript, Go, and Solidity fixtures from one checked-in schema source.
8. Configure formatter, linter, tests, and secret scanning in CI.

Commands:

```bash
forge fmt --check
cd go && go fmt ./... && go test ./...
cd ../web && pnpm install
cd ../relayer && pnpm install
```

Commit the generated lockfiles. CI and every later clean install use
`pnpm install --frozen-lockfile`.

Exit condition:

- the state machine, schema, and economic invariants have no unresolved P0 decisions.

### Phase 2 — pre-build validation lab (simulation only)

**Owners:** C, F, Q  
**Deadline:** 8 August, 21:00 WAT

This phase is an internal test environment. It is not a product feature, final deployment, traction claim, or hackathon proof.

Build:

- local mock ERC20 with FTestXRP decimals;
- deterministic random provider that can return secure/insecure/not-ready states;
- fixed local TEE test key;
- the real canonical bundle parser and Go verifier core;
- a minimal escrow state machine;
- the official FCC `ActionResult` signing-digest cross-language vector.

Required simulations:

1. Client funds; failing artifact returns FAIL; no balance moves.
2. Contractor resubmits corrected artifact; PASS transfers exactly the principal.
3. Bundle mutation fails commitment verification.
4. Artifact/address/code-hash mutation fails.
5. Wrong signer, high-S/malformed signature, wrong chain, wrong job, wrong attempt, wrong instruction, and replay all fail.
6. Insecure randomness advances only one deterministic round.
7. INCONCLUSIVE and timeout move no principal.
8. Expiry returns only the client principal and locked fee.
9. Paused mode still permits refunds and valid existing settlement.
10. Accounting invariant holds over fuzzed sequences.

Commands:

```bash
forge test -vvv
forge test --match-contract WorkProofInvariantTest -vvv
cd go && go test ./... -race
cd ../packages/schema && pnpm test
```

Simulation labeling:

- store fixtures only under `testdata/simulation-only/`;
- set `SIMULATION_ONLY=true` in local scripts;
- display a red `SIMULATION ONLY` banner in any temporary UI;
- exclude simulated addresses, screenshots, and code hashes from final evidence;
- production build must fail startup if a simulation-only adapter is selected.

Exit condition:

- all ten simulations pass and the FCC signature compatibility spike is green. If the signer path fails, stop before expanding the UI.

### Phase 3 — production smart contracts

**Owners:** C, Q  
**Deadline:** 9 August, 15:00 WAT

Implement:

- `WorkProofEscrow` as the registered FCC instruction sender;
- exact job/attempt state machine;
- FTestXRP-only P0 allowlist resolved during deployment;
- future-round commitment and historical secure-random lock;
- instruction dispatch through `TeeExtensionRegistry`;
- byte-exact native TEE result verification;
- PASS/FAIL/INCONCLUSIVE/timeout/refund paths;
- immutable current-job economics and bounded future-job fee;
- emergency pause that cannot block withdrawals/refunds.

Required contract tests:

- branch and line coverage target of at least 95% on fund-moving contracts;
- fuzz every externally callable state transition;
- invariant suite from section 6;
- timestamp boundaries at exactly `acceptBy`, `submitBy`, and grace end;
- decimals and rounding with 6-decimal FTestXRP;
- malicious/reentrant token harness even though P0 token is fixed;
- duplicate settlement and stale generation;
- signature malleability and malformed ABI data;
- registry returning zero/multiple/unexpected TEE IDs;
- random historical call not ready/insecure/secure.

Commands:

```bash
forge fmt --check
forge build --sizes
forge test -vvv
forge coverage
```

Exit condition:

- all invariants pass; no owner method can transfer job principal or declare an outcome; bytecode size fits limits.

### Phase 4 — Go FCC verifier

**Owners:** F, Q  
**Deadline:** 9 August, 21:00 WAT

Use Go because the official scaffold identifies it as the smallest single-process, cross-machine reproducible path.

Implementation tasks:

1. Set exact constants across Solidity, Go config, and tests:

```text
OP_TYPE = WORKPROOF
OP_COMMAND = VERIFY
```

2. Define ABI decoder for the instruction payload.
3. Validate all fixed fields and bounds before any network request.
4. Fetch ciphertext only from configured allowlisted HTTPS storage.
5. Verify ciphertext hash, decrypt via local node interface, canonicalize, and verify plaintext commitment.
6. Resolve/read WorkProof attempt state and runtime bytecode from Coston2 at the committed block.
7. Recompute artifact code hash and test seed.
8. Select vectors deterministically.
9. Execute bounded `eth_call`/read checks.
10. Produce deterministic `VerdictV1` and redacted report hash.
11. Return `status=1` only for a successfully completed verification, including business outcomes PASS/FAIL/INCONCLUSIVE inside `data`.
12. Use `status=0` for malformed/failed handlers; such a result cannot settle funds.
13. Increment semantic version whenever behavior or wire format changes.

Commands:

```bash
cd go
go fmt ./...
go vet ./...
go test ./... -race -cover
cd ..
bash ./scripts/test-conformance.sh
```

Exit condition:

- byte-exact conformance passes; same input produces same result bytes; secret canary is absent from logs and reports.

### Phase 5 — Coston2 simulated-attestation integration test

**Owners:** F, O, Q  
**Deadline:** 10 August, 12:00 WAT

This validates Flare routing before consuming real hardware time. It remains internal evidence and is never represented as production FCC.

Activate the current official simulated-Coston2 path:

```bash
./scripts/use-chain.sh local coston2 go
./scripts/pre-build.sh
./scripts/start-services.sh
./scripts/post-build.sh
./scripts/test.sh
```

Preconditions:

- `SIMULATED_TEE=true`;
- container `MODE=1`;
- stable development tunnel only for this phase;
- sanitized indexer configuration exists;
- no production keys.

Run a full WorkProof instruction and prove:

- instruction is emitted on Coston2;
- ext-proxy receives it;
- extension decrypts and verifies it;
- `/action/result/<instructionId>` returns `ActionResponse` rather than 404;
- on-chain contract verifies the signature under the simulated registered `teeId`;
- FAIL does not pay and PASS does pay on the test deployment.

Exit condition:

- round trip passes twice after a clean service restart. Do not run `pre-build --force` merely to repair routing.

### Phase 6 — reproducible release image

**Owners:** F, O, Q  
**Deadline:** 10 August, 16:00 WAT

Freeze the source commit and build the Go image reproducibly:

```powershell
$env:SOURCE_DATE_EPOCH = (git log -1 --format=%ct)
docker compose -f docker-compose.yaml build --no-cache extension-tee
$workproofImageId = docker compose -f docker-compose.yaml images -q extension-tee
if (-not $workproofImageId) { throw 'extension-tee image was not built' }
docker tag $workproofImageId workproof-extension:v0.1.0
docker save workproof-extension:v0.1.0 -o workproof-extension-v0.1.0.tar
docker inspect workproof-extension:v0.1.0 --format '{{range .Config.Env}}{{println .}}{{end}}' | Select-String MODE
docker inspect workproof-extension:v0.1.0 --format '{{index .Config.Labels "tee.launch_policy.allow_env_override"}}'
```

Rules:

- prefer one exact image digest for both preflight and real hardware;
- allow `MODE=0` as a launch override only if the baked launch-policy label permits it;
- calculate and record image digest, source commit, `SOURCE_DATE_EPOCH`, and expected verifier version;
- push/copy by digest; never rebuild at the destination;
- if source changes after this point, rebuild, obtain a new code hash, re-register, and rerun all real E2E evidence.

Exit condition:

- a second clean build from the same source produces the expected reproducible digest/code-hash result, and image labels permit every required launch override.

### Phase 7 — deploy real FCC hardware

**Owners:** O, F, Flare/GCP operator  
**Deadline:** 11 August, 12:00 WAT

Production sequence must remain linear:

1. Activate Coston2 production configuration:

```bash
bash ./scripts/use-chain.sh coston2
```

2. Confirm `.env.coston2` contains no simulated settings:

```text
LOCAL_MODE=false
SIMULATED_TEE=false
CHAIN_ID=114
```

3. Register/deploy the instruction sender and extension once:

```bash
bash ./scripts/pre-build.sh
```

The internal simulated-Coston2 test may reuse the same live-chain instruction
sender and extension ID only when the final contract, extension ID, and frozen
source are unchanged. If any of those changed, perform one intentional clean
final deployment, update the remote workload to the new extension ID, and audit
the old and new machine registrations before calling the one-shot binding. Never
let `config/extension.env` silently mix two deployments.

4. Hand off the exact image digest plus launch values:

- `INITIAL_OWNER`;
- `CHAIN_URL`;
- new `EXTENSION_ID`;
- internal `PROXY_URL`;
- `CHAIN_ID=114`;
- `MODE=0`;
- stable public HTTPS routing to proxy port 6664.

5. Obtain the public proxy URL and update `EXT_PROXY_URL`.
6. Verify `/info` before any registration:

```powershell
curl.exe -s "$env:EXT_PROXY_URL/info" | jq '.machineData'
```

Required values:

- platform begins with the GCP AMD SEV identifier;
- measured code hash is not the known simulated code hash;
- extension ID matches `config/extension.env`;
- initial owner matches;
- chain ID is 114;
- TEE/public key produces the reported `teeId`;
- image digest matches the handoff.

7. Register and promote the real machine:

```bash
bash ./scripts/post-build.sh
bash ./scripts/test.sh
```

8. Query active machines:

```bash
cd go/tools
go run ./cmd/query-tee -ext <extensionId> -rpc "$CHAIN_URL"
```

9. Confirm exactly one live production P0 machine. Pause stale IDs through the authorized manager only after comparing them with `/info`.

**Never run `full-setup.sh` against the remote TEE.** `setExtensionId()` is one-shot and must be bound last after registration is known correct.

Exit condition:

- real `/info`, on-chain registration, availability proof, `test.sh`, and a WorkProof VERIFY instruction all pass with `MODE=0`.

### Phase 8 — relayer and recovery CLI

**Owners:** A, Q  
**Deadline:** 11 August, 18:00 WAT

Relayer behavior:

1. Subscribe to `VerificationDispatched`.
2. Poll only the configured stable proxy for the exact instruction ID.
3. Treat 404 as “not available yet,” not FAIL.
4. Use bounded exponential backoff until contract timeout.
5. Validate the ActionResponse locally before sending a transaction.
6. Submit result/signature to `settleAttempt`.
7. Stop on settlement, invalid proof, timeout, or replaced instruction.
8. Store no secrets and never hold payment tokens.

Create a recovery command:

```bash
pnpm --dir relayer relay --instruction <id> --job <id> --attempt <n>
```

Tests:

- proxy 404 then success;
- malformed JSON;
- wrong proxy signature but valid/invalid TEE signature distinction;
- duplicate relayers race safely;
- transaction replacement and RPC retry;
- chain reorg/finality wait;
- timed-out instruction never settles;
- relayer key has only C2FLR and no admin role.

Exit condition:

- two independent relayer processes racing the same proof result in one settlement and one harmless revert/no-op.

### Phase 9 — web product

**Owners:** A, P, Q  
**Deadline:** 12 August, 12:00 WAT

Required screens:

1. **Landing:** “Lock payment. Verify work privately. Pay automatically.”
2. **Create job:** client, contractor, principal, deadlines, public template, private vectors, clear fees.
3. **Acceptance:** contractor sees immutable public requirements, test manifest, principal, deadline, and risk.
4. **Submit:** contract address, detected code hash, submission block, public smoke-test result.
5. **Job timeline:** funded → accepted → submitted → secure random → FCC verifying → PASS/FAIL → paid/refunded.
6. **Evidence panel:** chain, token, contract, extension, TEE, code hash, random round, instruction, result, balance deltas, explorer links.
7. **Recovery:** manual “Lock randomness,” “Dispatch,” “Relay result,” and “Refund expired” buttons when permissionless automation is unavailable.

UX rules:

- real wallet connection must request/reflect the user's wallet; never label a service connection “Connect wallet”;
- show Coston2 and testnet warnings permanently;
- distinguish “transaction submitted,” “confirmed,” “FCC pending,” and “settled”;
- a 404 result is pending, not failed;
- never display “verified” before on-chain settlement confirms;
- show exact contractor net amount and client total before signature;
- no fake user counts, fake volume, or fake partner logos;
- mobile-friendly but desktop demo is priority.

Exit condition:

- an external user completes the happy path using only on-screen instructions.

### Phase 10 — full real E2E and adversarial test

**Owners:** all; Q signs off  
**Deadline:** 12 August, 20:00 WAT

Run with distinct client and contractor wallets:

**Scenario A — fail, no payment**

1. Client creates and funds a 5 FTestXRP job plus disclosed fee.
2. Contractor accepts.
3. Contractor submits intentionally incorrect contract A.
4. Future secure randomness is locked.
5. Real FCC verifies.
6. Relayer submits signed FAIL.
7. Record unchanged contractor balance and still-locked principal.

**Scenario B — fix, pass, automatic payment**

1. Contractor deploys corrected contract B.
2. Contractor resubmits before deadline.
3. New exact future random round is committed and locked securely.
4. Real FCC verifies.
5. Relayer submits signed PASS.
6. Contract automatically sends exactly 5 FTestXRP to contractor and fee to treasury.
7. Record before/after balances and explorer transaction.

**Scenario C — refund**

1. Create a short-deadline job that never passes.
2. Advance naturally beyond deadline/grace on testnet.
3. Any caller invokes refund.
4. Tokens return only to the client.

**Adversarial suite**

- mutate every verdict field;
- submit from wrong signer;
- replay valid PASS;
- substitute artifact;
- substitute bundle;
- use insecure random round;
- use stale instruction after timeout;
- take relayer offline;
- restart proxy without restarting TEE;
- perform controlled TEE rotation only after jobs are drained;
- confirm no secret canary in browser, logs, proxy, chain, report, or analytics.

Exit condition:

- all three scenarios and all adversarial tests produce expected chain state. Save evidence immediately.

### Phase 11 — external pilot and product evidence

**Owners:** P, A  
**Deadline:** 13 August, 12:00 WAT

Recruit at least:

- one external developer as contractor;
- one external client, hackathon participant, or protocol contributor to review/create a job.

Pilot script:

1. Ask the person to explain WorkProof back in one sentence before using it.
2. Observe without coaching.
3. Record completion time and every point of confusion.
4. Ask whether they would use it, for what job size, and what would stop them.
5. Fix blockers, not cosmetic preferences.
6. Obtain permission before publishing a quote, name, wallet, or recording.

Metrics:

- user explains Lock → Verify → Pay correctly;
- job creation under five minutes;
- no confusion about testnet value;
- no developer assistance on wallet/network/settlement;
- pilot feedback and resulting change recorded in `docs/evidence/pilot.md`.

Never fabricate traction. A single real pilot with a documented lesson is stronger than invented user counts.

Exit condition:

- at least one external user completes or independently reviews the real flow,
  explains it correctly, and produces one documented usability finding and response.

### Phase 12 — security and release audit

**Owners:** Q, C, F, O  
**Deadline:** 13 August, 15:00 WAT

Checklist:

- all contract tests, fuzz, invariants, Go race tests, schema tests, UI tests, and real E2E pass at frozen commit;
- `forge inspect`/ABI review shows no admin outcome or principal-withdraw function;
- dependencies and Docker bases are pinned;
- source and image are reproducible;
- secret scan returns clean;
- logs contain no private bundle plaintext;
- contract source is verified on explorer;
- active TEE list contains only intended production machines;
- stable proxy returns healthy `/info` and the exact real result;
- restart/rotation runbook is tested;
- deployment manifest fields match chain and `/info` independently;
- public limitations are accurate;
- mock/simulation adapters are unreachable in production build/config;
- no placeholder address, fake metric, or unverified claim remains.

Exit condition:

- Q signs `docs/evidence/RELEASE_SIGNOFF.md`; unresolved P0 issue means no release.

### Phase 13 — submission package

**Owners:** P, all reviewers  
**Deadline:** 13 August, 18:00 WAT draft; 14 August, 12:00 WAT submit

Deliverables:

- public repository at frozen commit;
- deployed web application;
- 2–4 minute demo video;
- concise README with one-command local tests and real deployment evidence;
- architecture diagram;
- `deployments/coston2.json`;
- `docs/evidence/demo-run.json`;
- `NEW_WORK.md`;
- threat model and limitations;
- roadmap and pilot feedback;
- DoraHacks submission with bounty selection and every required field.

Do a judge test: give the README and video to someone who has never seen WorkProof. They must answer within one minute:

1. What problem does it solve?
2. Who pays whom?
3. Why is FCC necessary?
4. What proves the demo is real?
5. What happens when work fails?

If any answer is unclear, simplify the submission before adding features.

Exit condition:

- the DoraHacks submission is accepted before the internal deadline, a submission
  receipt/screenshot is stored, and every public link works from a logged-out browser.

---

## 16. Testing matrix

| Layer | Test class | Minimum release bar |
|---|---|---|
| Contracts | Unit | Every state transition and custom error |
| Contracts | Fuzz | Amounts, deadlines, attempts, random-round status, result fields |
| Contracts | Invariant | Solvency, single payout, recipient correctness, refund liveness |
| FCC | Unit/race | Parser, canonicalization, seed, selection, test types, bounds, redaction |
| FCC | Conformance | Official container/wire-format suite passes byte-exactly |
| Cross-language | Golden vectors | TS/Go/Solidity hashes and ABI agree |
| Signature | Golden/adversarial | Native TEE result recovers exact `teeId`; mutations/replays fail |
| Relayer | Unit/integration | 404, duplicate relay, timeout, invalid proof, RPC failure |
| Web | Component/E2E | Wallet/network, fund, accept, submit, statuses, evidence, refund |
| Storage | Integrity/security | Ciphertext/plaintext commitments, size cap, SSRF, missing blob |
| Coston2 | Simulated routing | Internal only; two clean round trips |
| Coston2 | Real hardware | MODE=0 FAIL/PASS/refund with explorer evidence |
| Operations | Restart/rotation | Proxy restart; drain-before-TEE-restart; stale ID paused |
| Security | Secrets | Canary absent everywhere outside TEE plaintext memory |
| User | Pilot | External person completes and understands flow |

### CI order

For every pull request:

```text
format/lint
→ schema golden vectors
→ Foundry unit/fuzz
→ Go unit/race/conformance
→ relayer/web tests
→ secret/dependency scan
→ Docker build
```

Real Coston2/hardware tests run manually on a frozen release commit because they spend gas, depend on external infrastructure, and must not leak keys into CI.

---

## 17. FCC operations runbook

### Healthy deployment definition

- stable public HTTPS proxy;
- `/info` returns expected production machine;
- correct chain/extension/owner;
- platform is GCP AMD SEV;
- measured code hash and image digest recorded;
- one intended active production TEE for P0;
- proxy can reach indexer DB;
- TEE can reach proxy, Coston2 RPC, and ciphertext gateway;
- `test.sh` and WorkProof business E2E both pass.

### 404 result diagnosis

Do not repeatedly re-register.

1. Confirm transaction emitted the expected instruction ID.
2. Confirm extension ID and target TEE in contract events.
3. Query active machines and compare with live `/info` TEE ID.
4. Check ext-proxy saw the instruction.
5. Check whether the TEE fetched the action.
6. Check extension handler status/log without exposing secrets.
7. Check proxy received the signed ActionResponse.
8. Poll the exact `/action/result/<instructionId>` path.
9. Only after identifying the broken hop should services be restarted.
10. If TEE itself restarts, follow rotation; do not pretend the old identity persists.

### TEE restart/rotation

Confidential Space creates a new TEE identity on relaunch. The old on-chain machine remains active until paused.

1. Pause new job creation.
2. Drain jobs encrypted to the current TEE; refund unresolved jobs when rules permit.
3. Record old `/info`, TEE ID, code hash, and active-machine list.
4. Launch exact pinned image digest with correct extension ID and `MODE=0`.
5. Verify new `/info`.
6. Run `post-build.sh` and availability proof for new machine.
7. Run `test.sh` and WorkProof E2E.
8. Query active machines again.
9. Pause the old TEE ID—never the new live ID.
10. Confirm exactly one intended active machine.
11. Re-enable new jobs; new bundles encrypt to the new public key.

There is no `unpause` shortcut; a paused machine returns only through a fresh production/availability process.

### Image change

1. Freeze new source commit.
2. Build reproducibly and record new digest/code hash.
3. Deploy exact digest.
4. Verify launch policy and `/info`.
5. Allow/register new code hash.
6. Run all real E2E tests.
7. Update evidence manifest.

### FCC manager/diamond redeploy

If system registrations are wiped:

1. run `pre-build.sh` to obtain a fresh extension and instruction sender;
2. restart remote workload with the new extension ID;
3. verify `/info` matches;
4. register/promote machine;
5. call one-shot `setExtensionId()` only at the correct final stage;
6. rerun E2E and redeploy WorkProof contracts if their immutable FCC bindings changed.

---

## 18. Evidence and observability

### Deployment manifest

Populate `deployments/coston2.json` from live queries, never manually guessed values:

```text
networkName
chainId
rpc/explorer
sourceCommit
contractAddresses
contractDeploymentTxs
verifiedSourceUrls
resolvedAssetManagerFXRP
resolvedFTestXRP
extensionId
teeId
teePlatform
teeCodeHash
imageDigest
proxyUrl
verifierVersionHash
deployedAt
```

### Demo evidence manifest

For FAIL, PASS, and REFUND, record:

- job and attempt;
- client/contractor public addresses;
- artifact address, block, and code hash;
- bundle/spec commitments, never plaintext;
- random round, security flag, and random hash;
- instruction and transaction IDs;
- TEE ID and result signature;
- report hash;
- outcome;
- before/after token balances;
- settlement/refund explorer URL;
- timestamps.

### Monitoring

Minimum alerts:

- proxy `/info` unavailable;
- live TEE ID changes;
- more than one active production TEE in P0;
- code hash/extension mismatch;
- indexer connection failure;
- instruction pending beyond expected time;
- repeated result 404 near timeout;
- relayer balance low;
- ciphertext gateway unavailable;
- contract accounting mismatch in read-only monitor.

Monitoring can notify humans but cannot sign a verdict or move principal.

---

## 19. Demo and judge narrative

### Opening sentence

“WorkProof lets a business lock payment for smart-contract work, privately verifies the agreed tests in Flare Confidential Compute, and pays the developer automatically only after the work passes.”

### 2–4 minute video structure

1. **Problem (15 seconds):** clients fear bad delivery; contractors fear nonpayment.
2. **Create (30 seconds):** show exact requirements, private-vector commitment, 5 FTestXRP principal, fee, and deadlines.
3. **Fail safely (35 seconds):** submit wrong contract, show secure random round and real FCC, then FAIL with no payment.
4. **Fix and pay (45 seconds):** submit corrected contract, show PASS and automatic contractor balance increase.
5. **Prove reality (40 seconds):** explorer transactions, verified source, extension/TEE IDs, GCP AMD SEV platform, non-simulated code hash, signed result, FTestXRP resolution.
6. **Future (15 seconds):** marketplace/API adapters and production audit—not fake finished features.

Because secure rounds take roughly 90 seconds and FCC routing can take longer, prepare real jobs at adjacent stages for a smooth live demo. The video may time-compress waiting, but show timestamps and explorer links so no result appears fabricated.

### One-screen evidence panel

Use green only for independently confirmed facts:

```text
Payment locked        5.000000 FTestXRP
Artifact              0x… / code hash 0x…
Randomness            Secure / round …
FCC                    GCP AMD SEV / MODE=0
TEE                    0x… / code hash 0x…
Verification           PASS / 12 of 12
Settlement             Paid / tx 0x…
```

### Honest limitations slide

- P0 verifies Coston2 smart-contract deliverables with bounded read/call templates.
- FTestXRP is testnet-only and has no monetary value.
- One active TEE is used; jobs must drain before key rotation.
- Subjective work requires explicit arbitration and is not supported in P0.
- Mainnet requires audit, legal, compliance, and production infrastructure work.

Honesty increases technical credibility; it does not weaken a complete demo.

---

## 20. Calendar: 7–14 August 2026

| Date/time WAT | Must finish | Hard decision |
|---|---|---|
| Aug 7 | Plan freeze; register project; request indexer/GCP access; fund wallets | Do not code optional features |
| Aug 8 12:00 | External dependencies confirmed | If hardware/indexer owner unknown, escalate red risk |
| Aug 8 21:00 | Simulation lab and native FCC signature spike pass | If signature is not contract-verifiable, stop expansion |
| Aug 9 15:00 | Contracts/invariants pass | No UI before solvency and auth are green |
| Aug 9 21:00 | Go verifier/conformance pass | No arbitrary execution scope creep |
| Aug 10 12:00 | Coston2 simulated-routing E2E passes internally | Diagnose every 404 before hardware handoff |
| Aug 10 16:00 | Reproducible image frozen and handed to operator | No rebuild-by-tag |
| Aug 11 12:00 | Real GCP AMD SEV TEE registered and basic E2E passes | If blocked, all hands on real FCC only |
| Aug 11 18:00 | Real WorkProof instruction and signed settlement pass | Kill WorkProof submission if G4/G5 remain red |
| Aug 12 12:00 | Web/relayer complete | Manual permissionless recovery must work |
| Aug 12 20:00 | Real FAIL → fix → PASS → pay and refund evidence | Any fund/control defect blocks release |
| Aug 13 12:00 | External pilot and UX fixes | Remove optional features that cause confusion |
| Aug 13 15:00 | Security/release audit signed | Freeze source/image/contracts |
| Aug 13 18:00 | Video, README, submission draft | No new feature commits |
| Aug 14 09:00 | Independent judge-readability review | Fix docs only unless critical |
| Aug 14 12:00 | Submit and verify links from logged-out browser | Do not wait for an unknown deadline timezone |

---

## 21. Kill criteria and fallback rules

The project is stopped or materially redesigned—not disguised—when any condition below remains true at its deadline.

| Failure | Deadline | Required response |
|---|---|---|
| Native FCC ActionResult cannot be verified in Solidity against official Go vector | Aug 8 21:00 | Get maintainer confirmation/fix; do not substitute backend authority |
| No confirmed real Confidential Space path | Aug 9 12:00 | Escalate; freeze all non-FCC features |
| No real MODE=0 machine/E2E | Aug 11 18:00 | Do not submit a simulated WorkProof as real |
| More than one unknown active P0 TEE | Before funding any final demo job | Pause stale machines and re-audit IDs |
| Contract owner can redirect/settle principal | Any time | Block release and redesign |
| Official FTestXRP cannot be resolved/transferred | Aug 10 | Fix payment rail; mock token is not final evidence |
| Secure historical randomness cannot be locked deterministically | Aug 10 | Fix with Flare guidance; do not use local random in final |
| Hidden-test fairness cannot be explained | Aug 9 | Restrict templates further or drop private-test claim |
| Secret appears in public surface | Any time | Rotate/recreate secrets, remove data, audit entire pipeline |
| External user cannot understand product | Aug 13 | Simplify flow and copy; remove optional screens |
| Final evidence contains placeholders/simulation | Release audit | Block submission until corrected |

Fallback does not mean silently weakening the trust model. Any scope change must update README, demo, threat model, and claim language together.

---

## 22. Post-hackathon path to a full-scale company

### Phase A — private beta (0–3 months)

- independent smart-contract audit;
- 10–20 pilot jobs with agencies/protocols;
- standard verifier templates for ERC interfaces and protocol-specific deliverables;
- stateful EVM sandbox with strict deterministic limits;
- multi-milestone project grouping;
- robust secret lifecycle and controlled TEE rotation;
- redundant relayers and production monitoring;
- clear commercial terms and dispute/arbitration partners;
- measure time-to-acceptance, disputes avoided, false pass/false fail, and cost per verification.

### Phase B — platform (3–9 months)

- organization accounts and approvals;
- API/webhook SDK for freelance and bounty platforms;
- container/API/WASM adapters only after sandbox security review;
- multi-TEE redundancy and quorum/consensus policy;
- Work Receipts and portable performance history;
- invoices, tax/accounting exports, and fiat-value reporting;
- mainnet deployment after audit and legal readiness;
- regulated payment/custody partner if business model or jurisdictions require one.

### Phase C — network (9–18 months)

- verifier-template marketplace with signed/versioned policies;
- specialized independent verification providers;
- arbitration network for explicitly subjective milestones;
- AI assistant for converting prose into a draft public spec, with both parties approving the deterministic result;
- contractor financing/insurance only as separate regulated products with explicit capital—not socialized participant losses.

### Business model

- client-paid success fee;
- fixed verification/compute fee;
- SaaS plans for agencies and marketplaces;
- enterprise API/support;
- template marketplace revenue share.

Do not base revenue on holding client funds longer, hidden spreads, token speculation, or forcing users to absorb other users' defaults.

---

## 23. Final release checklist

### Product

- [ ] One-sentence pitch is accurate.
- [ ] Public rules and private examples are clearly distinguished.
- [ ] Client and contractor amounts/fees/deadlines are visible before signing.
- [ ] FAIL, PASS, timeout, and refund behavior is understandable.
- [ ] External user completed the real flow.

### Contracts

- [ ] Official FTestXRP resolved and recorded.
- [ ] Principal fully pre-funded.
- [ ] Secure future random round committed and locked.
- [ ] Native FCC signature verified against stored TEE ID.
- [ ] All verdict/artifact/spec/random/instruction bindings checked.
- [ ] Fuzz and invariants pass.
- [ ] Admin cannot decide outcomes or seize principal.
- [ ] Source verified on explorer.

### FCC

- [ ] Go conformance suite passes.
- [ ] Exact image digest is frozen and reproducible.
- [ ] Real machine uses `MODE=0`, `SIMULATED_TEE=false`, and chain ID 114.
- [ ] `/info` shows GCP AMD SEV and non-simulated measured code hash.
- [ ] Extension, owner, TEE ID, and code hash match on-chain state.
- [ ] Exactly one intended active P0 machine.
- [ ] Real WorkProof instruction produces retrievable signed result.
- [ ] TEE rotation runbook tested.

### Security/operations

- [ ] No secrets in repository, chain, logs, browser, report, or screenshots.
- [ ] Ciphertext integrity and plaintext commitment both checked.
- [ ] SSRF, resource, time, gas, and response limits enforced.
- [ ] Stable proxy, indexer, RPC, storage, and relayer monitored.
- [ ] 404 and timeout recovery works.
- [ ] Refund remains live while new work is paused.

### Evidence/submission

- [ ] FAIL/no-pay explorer evidence.
- [ ] Corrected PASS/automatic-pay explorer evidence.
- [ ] Expiry/refund evidence.
- [ ] FTestXRP balance deltas.
- [ ] Secure random round/security flag.
- [ ] Extension ID, TEE ID, platform, code hash, and image digest.
- [ ] Git commit and new-work ledger.
- [ ] Pilot evidence and honest limitations.
- [ ] Video/app/repository links work logged out.
- [ ] DoraHacks entry submitted before internal deadline.

---

## 24. Reverification record

### Verified facts used by this plan

- Flare Summer Signal runs through 14 August 2026 and lists two $6,000 bounty pools.
- Confidential Compute Apps is a listed bounty.
- Judging covers usefulness, Flare integration, technical execution, new work, clarity, and future potential.
- FCC uses an on-chain instruction sender, TeeExtensionRegistry/TeeMachineRegistry, ext-proxy, and an extension in a TEE.
- The official scaffold's real deployment path requires production attestation (`MODE=0`) and `SIMULATED_TEE=false`.
- Go is the official scaffold's cross-machine reproducible single-process option.
- FCC `ActionResult` data is byte-sensitive and the node signs a domain-separated result hash.
- Confidential Space relaunch creates a new TEE ID; stale active machines can cause random routing and result 404s.
- `setExtensionId()` is one-shot; remote deployment must not use an all-in-one flow that binds it prematurely.
- Flare secure randomness exposes a security flag and historical round lookup; insecure randomness must not be used.
- Coston2 FTestXRP should be resolved through `AssetManagerFXRP`/`fAsset()` and has 6 decimals.
- Coston2 chain ID is 114 and Solidity should target Cancun.

### Telegram lessons incorporated

- A reachable proxy can still return 404 because the instruction never reached the live TEE or no result was produced.
- Repeated TEE registration can leave several dead machines active.
- Restarts rotate TEE identity; the official solution is controlled rotation and stale-machine pause, not assuming persistent enclave state.
- Indexer database host/port access is an external dependency and must be preflighted early.
- Temporary tunnels are unsuitable for a judge-facing final product.
- Wallet/service connection labels must describe what actually happens.
- Public contract addresses are normal; credentials, private endpoints, secrets, and unredacted configuration are not.
- A project must expose verifiable Flare integration rather than ask users to trust a narrative.

### Unresolved external facts that must be confirmed during execution

- exact DoraHacks deadline timezone;
- current indexer access details supplied privately by Flare;
- named GCP/Flare operator and Confidential Space quota;
- current FCC system registry deployment/configuration after any manager redeploy;
- current Coston2 FTestXRP resolved address at deployment time;
- production domain/TLS ownership;
- legal/compliance requirements for any later mainnet commercial release.

These are explicit tasks and gates, not hidden assumptions.

---

## 25. Primary sources

- Flare Summer Signal official DoraHacks path: <https://dorahacks.io/hackathon/flaresummersignal>
- Current event requirements mirror: <https://www.hackathonradar.com/database/hackathon/93d91cae-47e7-4db4-8734-1a9ed4d3fc9a>
- FCC getting started: <https://dev.flare.network/fcc/guides/getting-started>
- FCC private-key/signing example: <https://dev.flare.network/fcc/guides/sign-extension>
- Official FCC scaffold: <https://github.com/flare-foundation/fce-extension-scaffold>
- Official scaffold deployment steps: <https://github.com/flare-foundation/fce-extension-scaffold/blob/main/docs/deployment-steps.md>
- Normative extension container contract: <https://github.com/flare-foundation/fce-extension-scaffold/blob/main/docs/extension-contract.md>
- Secure randomness: <https://dev.flare.network/network/guides/secure-random-numbers>
- Historical randomness interface: <https://dev.flare.network/network/fsp/solidity-reference/IRelay>
- Flare contract registry: <https://dev.flare.network/network/guides/flare-contracts-registry>
- FAssets reference: <https://dev.flare.network/fassets/reference>
- FXRP Asset Manager resolution: <https://dev.flare.network/fassets/developer-guides/fassets-asset-manager-address-contracts-registry>

---

## 26. Go/no-go authorization

Implementation may begin only after the team accepts these statements:

1. WorkProof's hackathon P0 is an objective smart-contract-delivery verifier, not a universal judge of work.
2. Simulation is an internal validation gate only.
3. The final submission requires real Coston2 FTestXRP, secure Flare randomness, and real MODE=0 FCC hardware.
4. The native TEE ActionResult signature—not a trusted backend assertion—authorizes settlement.
5. Hidden tests contain private examples of public rules, not secret requirements.
6. No admin can create PASS or redirect principal.
7. If the real proof path fails the dated kill gates, the team stops rather than faking completion.

When all seven are accepted, execute Phase 0 and proceed in order.

---

## 27. Plan audit sign-off

This plan received a second pass after drafting. The audit checked scope,
economics, authorization, protocol facts, commands, deadlines, evidence, and
failure recovery.

| Audit area | Result | Evidence in this plan |
|---|---|---|
| Hackathon fit | Pass | One primary bounty, all five judging criteria mapped, submission fields listed |
| Economic conservation | Pass | Full pre-funding, explicit fee payer, no pooled/default loss, solvency invariants |
| Objective fairness | Pass with P0 restriction | Public rules/private examples, approved templates, no subjective AI verdict |
| FCC necessity | Pass | Private bundle decryption and attested deterministic verification |
| Settlement trust | Pass subject to G4 spike | Native TEE result signature, stored exact `teeId`, permissionless untrusted relayer |
| Randomness manipulation | Pass subject to Coston2 test | Future exact round, secure flag required, deterministic insecure-round advance |
| Artifact binding | Pass by design | Chain/address/block/code hash plus result and storage checks |
| Simulation honesty | Pass | Simulation isolated before build and prohibited from final evidence/config |
| Real-hardware path | Pass subject to external access | MODE=0 sequence, digest pin, `/info`, registration, stale-machine runbook |
| Fund recovery | Pass by design | Timeouts do not move principal; expiry/refund remains callable while paused |
| Admin authority | Pass by design | No PASS/refund-recipient override; ABI/invariant release audit |
| Operational failures | Pass | 404 tracing, indexer preflight, proxy stability, TEE rotation, manager redeploy |
| Command executability | Pass after corrections | Valid Go formatting, first-install lockfile flow, resolved Docker image ID |
| Deadline feasibility | High risk but executable | P0-first daily critical path and Aug 11 hardware kill gate |
| Claim accuracy | Pass | FTestXRP testnet value and P0 limitations stated repeatedly |

### Corrections made during audit

- Replaced invalid directory-level `gofmt` commands with `go fmt ./...`.
- Separated first-time `pnpm install` from later frozen-lockfile CI installs.
- Replaced an undefined Docker image placeholder with an explicit Compose image-ID lookup.
- Clarified that the official `ActionResult` signature covers only data, ID,
  submission tag, and status; all fund-critical fields must live inside signed data.
- Added a rule preventing simulated and production extension IDs/configuration
  from being silently mixed.
- Added explicit phase exit conditions for the external pilot and final submission.

### Residual red risks

The plan itself cannot remove three external/implementation risks:

1. access to real GCP Confidential Space and the Flare indexer;
2. byte-exact Solidity reproduction of the pinned FCC signing-domain hash;
3. completing and externally testing the entire real loop in seven days.

They are placed at the front of execution and have dated kill gates. There is no
mocked fallback authorized for any of them.
