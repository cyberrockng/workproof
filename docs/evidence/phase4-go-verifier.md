# Phase 4 — Go FCC verifier

This documents `go/internal/verifier` (the WorkProof VERIFY handler) as it
stands after Phase 4. It does not itself constitute a completed security
audit or proof of real end-to-end dispatch against a registered production
TEE machine — that remains blocked on Phase 0's external dependencies and is
Phase 5/7 scope.

## What was built

- `go/internal/config/config.go`: `OPTypeWorkProof`/`OPCommandVerify`
  constants matching `contracts/WorkProofEscrow.sol`'s `OP_TYPE`/`OP_COMMAND`
  exactly; `WorkProofEngineVersion`; resource limits from plan section 11
  (max bundle/ciphertext bytes, vector/selection counts, RPC/attempt
  timeouts, gas cap, response cap); `WORKPROOF_*` env vars for RPC URL,
  escrow address, RandomNumberV2 address, and the ciphertext gateway
  allowlist (not part of the scaffold's generic env surface — the greeting
  handlers never needed chain access at all).
- `go/pkg/types/workproof.go`: Go types + hand-written `abi.Argument`
  definitions for `WorkProofInstruction` and `VerdictV1`, mirroring the
  Solidity structs field-for-field. **Proven byte-exact against real
  Solidity `abi.encode(...)` output** (`workproof_test.go`,
  `workproof_instruction_test.go`) — not assumed compatible.
- `go/pkg/contracts/workproofescrow/`: **abigen-generated** (not
  hand-written) read bindings for `WorkProofEscrow.sol`, produced by
  `scripts/generate-workproof-bindings.sh` from the real `forge build`
  output ABI — the same recipe the scaffold's own
  `scripts/generate-bindings.sh` uses for `HelloWorldInstructionSender`.
  Regenerate after any contract ABI change.
- `go/internal/verifier/`: the verification engine itself —
  `bundle.go`/`validate.go` (bundle types, JCS/RFC 8785 canonicalization via
  the real `gowebpki/jcs` library, full schema validation matching
  `packages/schema/schemas/workproof-bundle-v1.schema.json`);
  `decrypt.go` (real `POST /decrypt` client to tee-node's local `SIGN_PORT`,
  base64 wire format per `docs/extension-contract.md` §3 — explicitly
  flagged there as "the single most common porting mistake"); `ciphertext.go`
  (HTTPS-only, host-allowlisted fetch with SSRF/DNS-rebinding defense: every
  hop including redirects is scheme/host-checked, and the resolved IP is
  rejected if private/loopback/reserved); `vectors.go` (independent
  re-fetch of the historical random number, `TestSeed` derivation exactly
  per plan section 10 step 8, deterministic Fisher-Yates selection, and
  execution of all 5 P0 vector types via real `eth_call`/`eth_getCode`/
  `eth_getStorageAt`); `verifier.go` (orchestration: instruction decode →
  on-chain job cross-check → artifact code-hash recompute → random-number
  cross-check → bundle fetch/decrypt/commitment-check → vector selection/
  execution → `VerdictV1` + report hash).
- `go/internal/extension/extension.go`: `processWorkProof`/`processVerify`
  wired into the existing `processAction` OPType/OPCommand routing, exactly
  matching the scaffold's own `processGreeting`/`processSayHello` pattern.

## A real gap found and fixed mid-phase: missing ciphertext commitment

While wiring the bundle-fetch step, discovered that `WorkProofInstruction`
(and `JobTerms`) had no ciphertext hash/locator field at all — despite
SPEC.md's Job Terms explicitly listing "ciphertext hash and content-addressed
locator" as required. Went back into the already-shipped Phase 3 contract and
added `bytes32 ciphertextHash` to `JobTerms`/`WorkProofInstruction`/
`createJob`, deliberately **not** a full on-chain locator string — the fetch
URL is `https://{engine-configured allowlisted gateway}/{ciphertextHash}`,
so "content-addressed" already implies the path without paying for an
expensive on-chain string. This is exactly the kind of thing the "mandatory
source of truth" discipline is meant to catch: re-reading SPEC.md while
implementing Phase 4 surfaced a real omission in already-"complete" Phase 3
work, and it was fixed rather than worked around.

Consequence: touched every `createJob` call site (14 across
`test/WorkProofEscrow.t.sol` + `test/RealRegistryFork.t.sol`), regenerated
the abigen bindings, and added a second real cross-language ABI proof
(`workproof_instruction_test.go`) for the now-14-field `WorkProofInstruction`.
Full local (90 tests) + real Coston2 fork (7 tests) suites re-verified green
after the change.

## A second real finding: the Solidity optimizer was never enabled

Adding `ciphertextHash` dropped `WorkProofEscrow`'s deployed bytecode margin
to a critical **44 bytes** under the EIP-170 limit. Investigating led to
discovering `foundry.toml` never set `optimizer = true` — `via-ir = true`
alone does not enable it, and forge's default was `optimizer = false`.
Enabling `optimizer = true` / `optimizer_runs = 200` dropped the contract
from 24,532 to **10,455 bytes** (14,121-byte margin) with zero semantic
change (verified, not assumed: the full 90-test local+spike suite and the
real Coston2 fork suite were rerun and stayed green after the change).

## Cross-language ABI proofs (not hand-derived, not assumed)

Both directions of the wire protocol are proven against real Solidity
`abi.encode(...)` output, generated via throwaway Foundry test fixtures
(`console.logBytes`), never hand-transcribed:

- `TestWorkProofInstructionDecodesRealSolidityEncoding`: Go correctly
  decodes a real Solidity-encoded `WorkProofInstruction`, and re-encoding
  reproduces the exact original bytes.
- `TestVerdictV1DecodesRealSolidityEncoding`: same proof for `VerdictV1`.

A real Go/ABI interop bug was caught while building these: go-ethereum's
struct-to-tuple `abi.Pack` matches Go struct fields to ABI component names by
capitalizing only the first rune (`"chainId"` → `"ChainId"`, `"id"` →
`"Id"`) — **not** idiomatic Go acronym casing. Using `ChainID`/`JobID`/
`InstructionID`/`ID` broke round-trip re-encoding with "field not found in
struct" even though `abicoder.Decode` (positional) tolerated either
spelling. Documented in-code so this isn't "fixed" back to idiomatic casing
by a future refactor.

## Test coverage

`go test ./... -race -cover`: all packages pass, race-clean.
`internal/verifier`: 32.5% statement coverage — deliberately concentrated on
the parts that are genuinely unit-testable without a live chain:

- Full bundle schema validation (`bundle_test.go`): valid bundle accepted;
  wrong format version, wrong chain, selection>vectorCount, vectorCount
  mismatch, gas limit over the engine cap, duplicate vector IDs, and
  cross-type field leakage (e.g. a CODE_SIZE_RANGE vector carrying
  ETH_CALL fields) all rejected; all 5 P0 vector types individually accepted
  when well-formed.
- JCS canonicalization determinism and key-order independence
  (`bundle_test.go`) — a real RFC 8785 library (`gowebpki/jcs`), not a
  hand-rolled canonicalizer.
- Fisher-Yates selection determinism, no-duplicates/in-range, and
  selection-count capping (`vectors_test.go`); `TestSeed` determinism and
  sensitivity to every binding input.
- The real `/decrypt` client's base64 wire format (`decrypt_test.go`,
  via `httptest`) — proving the exact mistake the doc warns about
  (accidentally sending hex) would be caught.
- Ciphertext fetcher SSRF defenses (`ciphertext_test.go`, via
  `httptest.NewTLSServer`): non-https rejected, non-allowlisted host
  rejected, an allowlisted host resolving to a private IP rejected
  (DNS-rebinding defense, using an injected resolver so the test doesn't
  depend on real DNS), a genuine happy-path fetch through a TLS-trusted
  local server, and a real over-cap response rejected -- with an explicit
  guard against the exact false-positive-test bug this session already hit
  twice before (a test "passing" because of an unrelated TLS/transport
  failure rather than the logic under test).

**Honest gap, not hidden**: `Verify()`'s full orchestration (on-chain job
cross-check, artifact code-hash recompute, random-number re-fetch, vector
execution against a real deployed artifact) needs live or realistically
mocked chain RPC and is not unit-tested end-to-end in this phase. This is
deliberately scoped to Phase 5 ("Coston2 simulated-attestation integration
test"), which exists specifically to prove this; building a full JSON-RPC
mock server here would risk the same "does the mock even match the real
protocol" question this session has repeatedly had to verify against real
source for everything else, and Phase 5's real fork/proxy is the actual
authority on that question, not a hand-rolled mock.

## Secret canary / no-plaintext-in-logs check

Reviewed every `fmt.Errorf`/potential log path in `internal/verifier` for
plaintext or ciphertext content leaking into error strings or the
`ActionResult.log` field (which is part of the wire response). None found:
error messages are static descriptions or reference only committed hashes
(public, not secret) and vector `ID`/`Type` labels (public metadata per
SPEC.md's own distinction — "hidden inputs or expected outputs" refers to
`calldata`/`expectedReturn`/`interfaceId`/`slot`/`expectedValue`, not vector
labels). `PublicReport`/`reportHashFor` includes only `{id, passed, skipped}`
per vector, deliberately excluding every hidden-input/expected-output field.
`decrypt.go` explicitly does not log response bodies on error paths.

## Verification commands run (real exit codes, not truncated)

- `go build ./...` — exit 0
- `gofmt -l .` — exit 0, no files listed
- `go vet ./...` — exit 0
- `go test ./... -race -cover` — exit 0, all packages pass
- `bash ./scripts/test-conformance.sh go` — exit 0, 16/16 existing fixtures
  pass (GREETING-only; no WORKPROOF conformance fixtures exist yet, since
  they would need either a live chain or a mock-verifier run mode that
  hasn't been built — an honest, tracked gap, not silently skipped)

## Post-audit corrections (2026-08-09)

An independent audit found four release-blocking defects and several
high-severity gaps after this phase was originally reported complete. Every
finding checked against source was accurate. Full list and fixes in
`NEW_WORK.md` "Post-audit remediation"; the ones this document's original
claims most directly need correcting for:

- **C1 (the most severe)**: `verdictIdentityFrom` never populated
  `VerdictIdentity.InstructionId` -- every real Go-produced verdict had a
  zero instructionId, so `WorkProofEscrow.sol`'s signature recovery (which
  reconstructs the digest using `v.id.instructionId` extracted from the
  decoded data) always recovered the wrong signer. **No Go-generated result
  could ever have settled**, despite this document's cross-language ABI
  proofs (still accurate on their own terms -- byte-exact encode/decode was
  genuinely proven) and 32.5% coverage figure implying more functional
  completeness than existed. Fixed by threading the real `ActionResult.ID`
  through from `processVerify`.
- **C4**: `executeEthCallReverts` treated any RPC/transport error
  identically to a genuine EVM revert -- an RPC outage could force a false
  Pass on an ETH_CALL_REVERTS vector. `msg.value` was validated by schema
  but silently never applied to calls.
- High-severity: `bundle.PublicSpecHash` was never actually compared to the
  real on-chain `instr.SpecHash`; `EngineVersionHash` was echoed from
  client input rather than reflecting the verifier's real running code; the
  ciphertext fetcher's DNS-rebinding defense re-resolved DNS at connect
  time instead of pinning the validated IP (a real TOCTOU gap despite the
  original doc comment's claim to have closed it); `config.MaxBundleBytes`/
  `TimeoutMsPerCall`/the ERC-165 vector's gas cap were validated but never
  enforced.
- Coverage climbed from 32.5% to 43.6% during remediation, driven by real
  new tests for each fix above (fake-RPC-server tests distinguishing
  genuine reverts from transport failures, DNS-pinning tests using a real
  local listener, resource-limit tests, an engine-version regression test)
  -- not from padding existing paths.
- The deployment pipeline finding (C2 -- `pre-build.sh` deployed the
  scaffold's sample `HelloWorldInstructionSender`, never `WorkProofEscrow`)
  is Phase 5/tools-scope, not this phase's own claims, but is the reason
  Phase 5 could not have used any deployment this phase's tooling produced
  either. See `NEW_WORK.md`.

## Known limitations / not yet done

- No WORKPROOF/VERIFY conformance fixtures in `testdata/conformance/` (see
  above).
- `Verify()` end-to-end is unproven against real/mocked chain RPC (Phase 5
  scope, as designed).
- Real end-to-end dispatch — an actual signed `ActionResult` arriving from a
  real registered production TEE machine — remains blocked on Phase 0's
  external dependencies (GCP Confidential Space + registration).
- `PublicReport` does not yet include per-vector timing, which SPEC.md's
  report description mentions ("vector IDs, statuses, timing"); only
  ID/passed/skipped are currently included.
