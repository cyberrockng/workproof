# Static analysis review (2026-08-10)

An independent audit listed Slither, Semgrep, ShellCheck, Gitleaks, and
TruffleHog as tools it could not run in its own environment. This document
records actually running the ones that could be installed without root
access, what they found, and the disposition of every finding — fixed,
already mitigated, or reviewed and accepted, never silently dismissed.

ShellCheck and TruffleHog could not be installed here either (`apt install`
needs sudo, which this environment doesn't have; no equivalent user-space
binary was found for either). Still an open gap.

## gitleaks (v8.30.1)

Two scans: the raw working tree (`--no-git`, everything present on disk)
and real git history (`--source .`, what's actually reachable via the
public repo). Every finding reviewed:

| Finding | Where | Disposition |
|---|---|---|
| Real deployer/proxy private key | `.env`, `.env.coston2` (working tree only) | Gitignored, never committed (confirmed via `git check-ignore`), wallet holds 0 funds (see `docs/operations/external-dependencies.md`) |
| Hardhat's well-known public dev key (`983760a4...`) | `.env.example` (tracked); also in scaffold git history from before this project's Phase 1 | Intentionally public devnet convention key, not a secret — same key the upstream scaffold itself ships |
| forge-std's well-known public test key (`ac0974be...`) | `docs/evidence/fcc-signature-spike-v1.json`, `go/cmd/fcc-spike/main.go` (both tracked); confirmed identical to `lib/forge-std/test/StdCheats.t.sol`'s own hardcoded constant | Public Foundry-ecosystem test convention, explicitly labeled `privateKeyTestOnly` in the evidence file, not a secret |
| Placeholder hex pattern (`abcdef1234...`) | `tools/pkg/validate/validate_test.go` (scaffold-provided test fixture) | Obviously fake sequential pattern, not a real key |
| Contributor email | `config/coston2/deployed-addresses.json` git history | Normal git authorship metadata from the upstream scaffold, not a secret |

**No real secret was ever committed or pushed.**

## semgrep (v1.172.0, `--config auto`, 615 rules, 245 files)

18 findings across 7 rule categories. Zero findings in `contracts/` (21
Solidity rules ran, 5 files, clean).

Fixed:

- **GitHub Actions mutable tag pinning** (`.github/workflows/ci.yml`, 11
  instances) — every `uses:` now pinned to a full commit SHA (resolved via
  the real GitHub API, not guessed), with the release version as a trailing
  comment. Tags/branches can be silently repointed by the action owner or a
  compromised account (real incidents: tj-actions/changed-files,
  reviewdog/action-setup).
- **pnpm workspace hardening** (`pnpm-workspace.yaml`) — added
  `blockExoticSubdeps: true`, `minimumReleaseAge: 10080` (7 days),
  `trustPolicy: no-downgrade`. These require pnpm >=10.16-10.26; the
  project is currently pinned to pnpm 9.15.9 (`ci.yml`), so they're
  forward-compatible settings that take effect once that pin is bumped —
  confirmed pnpm 9.15.9 parses and silently ignores the unrecognized keys
  rather than erroring (`pnpm install --frozen-lockfile` and `pnpm -r test`
  both still exit 0 with them present) before adding them.

Reviewed, not fixed (out of WorkProof's active scope — inherited,
unmodified scaffold code for languages/tooling this project doesn't use):

- `go/internal/extension/utils.go`'s `w.Write(body)` (XSS-pattern rule) —
  this writes a JSON API response body, not HTML into a browser context;
  the rule doesn't distinguish content type. Also scaffold "DO NOT MODIFY"
  boilerplate.
- `python/base/node.py` dynamic urllib use — `python/` is scaffold-owned,
  inherited, and unused (WorkProof is Go-only).
- `testing/scripts/install.sh` curl-pipe-bash (×2) — scaffold-owned dev
  environment bootstrap tooling, not part of the deployed system's attack
  surface.

## slither (v0.11.6) against `contracts/WorkProofEscrow.sol`

64 raw results across 12 detector categories; the large majority are in
*vendored* `lib/flare-foundry-periphery-package` interfaces (naming
convention, unindexed event params, wide pragma ranges) that this project
doesn't own or modify. The findings actually in `WorkProofEscrow.sol`,
reviewed individually:

- **`reentrancy-eth`** in `dispatchVerification`: state (`instructionId`,
  `dispatchedAt`, `timeoutAt`) is written *after* the external
  `TEE_EXTENSION_REGISTRY.sendInstructions` call. Real CEI-ordering
  violation in the raw pattern slither detects — but `dispatchVerification`
  is `nonReentrant`, and `instructionId` specifically can't be written
  *before* the call because it's the call's own return value. Already has a
  dedicated, passing regression test that arms a malicious extension
  registry to attempt exactly this reentrancy
  (`testMaliciousExtensionRegistryCannotReenterDispatch`,
  `test/WorkProofEscrow.t.sol`). **Accepted: mitigated by the reentrancy
  guard, not by literal CEI ordering, and proven by a real adversarial
  test, not just asserted.**
- **`uninitialized-local`** (`lockRandomness`'s `randomNumber`/`isSecure`) —
  declared before a `try/catch` block so they're usable after it; a
  required Solidity scoping pattern, not a real uninitialized-value bug
  (Solidity value types are always zero-initialized regardless). No action.
- **`unused-return`** (`lockRandomness` ignores
  `RandomNumberV2Interface.getRandomNumberHistorical`'s third return value,
  the result timestamp) — deliberately discarded via an unnamed return
  parameter (`returns (uint256 rn, bool secure, uint256)`), the idiomatic
  Solidity way to say "intentionally unused," not a careless drop. The
  timestamp isn't needed anywhere in the escrow's logic. No action.
- **`calls-loop`/`costly-loop`** in `setExtensionId()` — scaffold "DO NOT
  MODIFY" function (see `contracts/WorkProofEscrow.sol`'s own comment and
  `docs/evidence/phase3-production-contracts.md`); the loop bound is
  `[FIRST_PUBLIC_EXTENSION_ID, nextPublicExtensionId())` and the function
  can only ever execute once per contract (`require(_extensionId == 0)`),
  so there's no recurring or attacker-amplifiable gas cost. No action.
- **`timestamp`** (deadline comparisons throughout) — the same
  `block.timestamp`-in-comparisons pattern `forge lint` already flags
  project-wide; an accepted, inherent tradeoff for deadline-based logic
  measured in hours (validator timestamp manipulation is bounded to a few
  seconds, irrelevant at this scale). Not new, already reviewed throughout
  this project.
- **`solc-version`** — about the *vendored* dependencies' wide pragma
  ranges (`>=0.7.6<0.9` etc.), not `WorkProofEscrow.sol`'s own compiler
  version, which is pinned exactly (`foundry.toml`, `solc = "0.8.35"`).
- **`naming-convention`/`immutable-states`** — cosmetic (constant-style
  naming for immutable registry references; `owner` could technically be
  `immutable` but is intentionally mutable for potential future ownership
  transfer, which the contract doesn't currently implement but doesn't rule
  out either). No functional impact.

**Net result: zero un-mitigated, actionable findings in
`WorkProofEscrow.sol` itself.**
