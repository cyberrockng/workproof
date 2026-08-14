package verifier

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gowebpki/jcs"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/contracts/workproofescrow"
	"extension-scaffold/pkg/types"
)

// realEngineVersionHash is the ACTUAL running engine version, computed once
// as keccak256(config.WorkProofEngineVersion) -- matching the byte-proven
// convention in pkg/types/workproof_instruction_test.go and every test
// fixture's ENGINE_HASH = keccak256("engine-v1") (NOT Solidity's raw
// bytes32(string) encoding, which is what OP_TYPE/OP_COMMAND use instead --
// those are two different, non-interchangeable encodings in this codebase).
// This is the only trustworthy source for VerdictOutcome.EngineVersionHash
// -- instr.EngineVersionHash is client/job input, and blindly echoing it
// back (the pre-fix bug) makes the "engine version" binding pure theater:
// it would always match whatever a job was created with, never reveal that
// a *different* engine version actually performed the verification.
var realEngineVersionHash = keccak256([]byte(config.WorkProofEngineVersion))

// Config wires the verifier to its real dependencies. No field here is
// ever a mock/placeholder in the production path -- only test/ doubles
// stand in for these during local unit tests.
type Config struct {
	ChainID            uint64
	RPCURL             string
	EscrowAddress      common.Address
	RandomNumberV2Addr common.Address
	SignPort           int
	CiphertextHosts    []string
}

type Verifier struct {
	cfg       Config
	eth       *ethclient.Client
	escrow    *workproofescrow.WorkProofEscrowCaller
	decrypter *NodeDecrypter
	fetcher   *CiphertextFetcher
}

func New(cfg Config) (*Verifier, error) {
	eth, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("dialing chain RPC: %w", err)
	}
	escrow, err := workproofescrow.NewWorkProofEscrowCaller(cfg.EscrowAddress, eth)
	if err != nil {
		return nil, fmt.Errorf("binding escrow contract: %w", err)
	}
	return &Verifier{
		cfg:       cfg,
		eth:       eth,
		escrow:    escrow,
		decrypter: NewNodeDecrypter(cfg.SignPort, nil),
		fetcher:   NewCiphertextFetcher(cfg.CiphertextHosts),
	}, nil
}

// HandlerFailure signals a status=0 case: the extension could not even
// establish a trustworthy job/attempt identity to bind a verdict to, so no
// verdict is produced at all (plan Phase 4 task 12: "such a result cannot
// settle funds").
type HandlerFailure struct{ Err error }

func (e *HandlerFailure) Error() string { return e.Err.Error() }
func (e *HandlerFailure) Unwrap() error { return e.Err }

func fail(format string, args ...any) error {
	return &HandlerFailure{Err: fmt.Errorf(format, args...)}
}

// Verify runs the full WorkProof VERIFY flow (plan section 15 Phase 4 tasks
// 3-10) and returns a fully-formed VerdictV1. A *HandlerFailure return means
// status=0 (identity/binding could not be established at all); any other
// non-nil error is also treated as status=0 out of caution. Once identity
// is established, verification-layer problems (bad bundle, unreachable
// artifact, etc.) are reported as a valid Outcome=Inconclusive verdict
// instead, since that is specifically the "infrastructure error" outcome
// SPEC.md's outcome table defines -- not a handler failure.
// instructionId is the real ActionResult.ID (tee-node's ActionData.ID,
// echoed back into every ActionResult -- see buildResult in
// internal/extension/utils.go) that tee-node actually signs over
// (FccVerdict.sol's actionResultHash includes it directly). It is NOT part
// of the ABI-encoded WorkProofInstruction payload -- the registry assigns it
// at dispatch time -- so it must be threaded in from the caller rather than
// read off instr. It must end up in the returned VerdictV1.Id.InstructionId
// exactly, or WorkProofEscrow.sol's _decodeAndAuthenticate recovers the
// wrong signer (it reconstructs the digest using v.id.instructionId
// extracted from the decoded data) and settlement reverts for every result.
func (v *Verifier) Verify(ctx context.Context, instructionId [32]byte, instr types.WorkProofInstruction) (types.VerdictV1, error) {
	ctx, cancel := context.WithTimeout(ctx, config.AttemptTotalTimeout)
	defer cancel()

	if instr.ChainId == nil || instr.ChainId.Uint64() != v.cfg.ChainID {
		return types.VerdictV1{}, fail("instruction chainId %v does not match configured chain %d", instr.ChainId, v.cfg.ChainID)
	}
	if instr.EscrowAddress != v.cfg.EscrowAddress {
		return types.VerdictV1{}, fail("instruction escrowAddress %s does not match configured escrow %s", instr.EscrowAddress, v.cfg.EscrowAddress)
	}
	if instr.JobId == nil {
		return types.VerdictV1{}, fail("instruction missing jobId")
	}
	// This verifier cannot honestly produce a verdict for a job that expects
	// a different engine version -- a configuration-level rejection, not a
	// job-specific processing failure, so it belongs alongside the
	// chainId/escrowAddress checks above rather than becoming a signed
	// Inconclusive verdict.
	if instr.EngineVersionHash != realEngineVersionHash {
		return types.VerdictV1{}, fail("instruction engineVersionHash %x does not match this verifier's real running version %x", instr.EngineVersionHash, realEngineVersionHash)
	}

	// Re-read the real on-chain job state; never trust the relayed
	// instruction's claims about job identity without cross-checking them
	// (SPEC.md: "the extension re-reads job/attempt at the dispatch block
	// and rejects a mismatch").
	job, err := v.escrow.GetJob(nil, instr.JobId)
	if err != nil {
		return types.VerdictV1{}, fail("reading on-chain job state: %w", err)
	}
	if err := crossCheckInstruction(instr, job); err != nil {
		return types.VerdictV1{}, fail("instruction does not match on-chain job state: %w", err)
	}
	// The routing layer determined this ActionResult corresponds to this
	// dispatch, but never trust that without an explicit cross-check against
	// the same on-chain-stored instructionId settleAttempt will itself
	// require to match (defense in depth, same discipline as every other
	// crossCheckInstruction field).
	if instructionId != job.Current.InstructionId {
		return types.VerdictV1{}, fail("actionResult id %x does not match on-chain dispatched instructionId %x", instructionId, job.Current.InstructionId)
	}

	artifactBlock := instr.ArtifactBlock
	recomputedCodeHash, err := v.recomputeArtifactCodeHash(ctx, instr.ArtifactAddress, artifactBlock)
	if err != nil {
		return types.VerdictV1{}, fail("reading artifact code: %w", err)
	}
	if recomputedCodeHash != instr.ArtifactCodeHash {
		return types.VerdictV1{}, fail("recomputed artifact code hash does not match instruction-committed value")
	}

	randomNumber, isSecure, err := FetchRandomNumber(ctx, v.eth, v.cfg.RandomNumberV2Addr, instr.RandomRound)
	if err != nil {
		return types.VerdictV1{}, fail("reading historical random number: %w", err)
	}
	if !isSecure {
		return types.VerdictV1{}, fail("random round %v is not secure -- lockRandomness should never have locked this round", instr.RandomRound)
	}
	if keccak256AbiEncodedUint(randomNumber) != instr.RandomValueHash {
		return types.VerdictV1{}, fail("recomputed randomValueHash does not match instruction-committed value")
	}

	identity := verdictIdentityFrom(instr, instructionId)

	// From here on, problems become Outcome=Inconclusive, not a handler
	// failure: identity/binding is already sound, so a signed Inconclusive
	// verdict is honest and lets the job retry rather than getting stuck.
	bundle, bundleErr := v.fetchAndDecryptBundle(ctx, instr)
	if bundleErr != nil {
		return v.inconclusiveVerdict(instr, identity, "bundle: "+bundleErr.Error())
	}

	seed, err := TestSeed(randomNumber, instr.EscrowAddress, instr.JobId, instr.Attempt, instr.SpecHash, instr.ArtifactCodeHash)
	if err != nil {
		return v.inconclusiveVerdict(instr, identity, "deriving test seed: "+err.Error())
	}
	selected := SelectVectors(seed, len(bundle.Vectors), bundle.SelectionCount)

	outcomes := make([]VectorOutcome, 0, len(selected))
	allPassed := true
	anySkipped := false
	for _, idx := range selected {
		// bundle.TimeoutMsPerCall is schema-validated but was previously
		// never actually enforced -- only the overall AttemptTotalTimeout
		// bounded anything, so one slow/hanging call could silently consume
		// the entire attempt's budget instead of just failing its own vector.
		callCtx, cancel := context.WithTimeout(ctx, time.Duration(bundle.TimeoutMsPerCall)*time.Millisecond)
		o := ExecuteVector(callCtx, v.eth, instr.ArtifactAddress, artifactBlock, bundle.GasLimitPerCall, bundle.MaxResponseBytes, bundle.Vectors[idx])
		cancel()
		outcomes = append(outcomes, o)
		if o.Skipped {
			anySkipped = true
		} else if !o.Passed {
			allPassed = false
		}
	}

	outcome := types.OutcomePass
	switch {
	case anySkipped:
		outcome = types.OutcomeInconclusive
	case !allPassed:
		outcome = types.OutcomeFail
	}

	reportHash, err := reportHashFor(outcomes)
	if err != nil {
		return v.inconclusiveVerdict(instr, identity, "building report: "+err.Error())
	}

	// Deliberately the TEE host's real wall-clock time, not a derived/
	// deterministic value -- IssuedAt is a genuine timestamp, so it's
	// expected to vary run-to-run. WorkProofEscrow.sol's
	// _checkOutcomeBinding bounds it to [dispatchedAt, block.timestamp],
	// which turns a drifted TEE clock into a safe settlement rejection
	// (liveness), never acceptance of a falsely-timestamped verdict. See
	// THREAT_MODEL.md "TEE clock drift" / "Residual Risks".
	now := uint64(time.Now().Unix())
	return types.VerdictV1{
		Id: identity,
		Result: types.VerdictOutcome{
			ArtifactCodeHash:  instr.ArtifactCodeHash,
			RandomRound:       instr.RandomRound,
			RandomValueHash:   instr.RandomValueHash,
			EngineVersionHash: realEngineVersionHash,
			Outcome:           uint8(outcome),
			PassedCount:       countPassed(outcomes),
			ExecutedCount:     uint32(len(outcomes)),
			ReportHash:        reportHash,
			IssuedAt:          now,
			ExpiresAt:         instr.ExpiresAt,
		},
	}, nil
}

// inconclusiveVerdict still binds every field _checkOutcomeBinding checks
// (WorkProofEscrow.sol runs that check unconditionally, for every outcome,
// not only Pass) -- an under-bound Inconclusive verdict would be rejected
// as InvalidVerdict by the contract rather than accepted into Retryable.
func (v *Verifier) inconclusiveVerdict(instr types.WorkProofInstruction, identity types.VerdictIdentity, detail string) (types.VerdictV1, error) {
	reportHash, err := reportHashFor([]VectorOutcome{{ID: "infrastructure", Skipped: true, Detail: detail}})
	if err != nil {
		return types.VerdictV1{}, fail("building inconclusive report: %w", err)
	}
	return types.VerdictV1{
		Id: identity,
		Result: types.VerdictOutcome{
			ArtifactCodeHash:  instr.ArtifactCodeHash,
			RandomRound:       instr.RandomRound,
			RandomValueHash:   instr.RandomValueHash,
			EngineVersionHash: realEngineVersionHash,
			Outcome:           uint8(types.OutcomeInconclusive),
			PassedCount:       0,
			ExecutedCount:     1,
			ReportHash:        reportHash,
			IssuedAt:          uint64(time.Now().Unix()),
			ExpiresAt:         instr.ExpiresAt,
		},
	}, nil
}

func verdictIdentityFrom(instr types.WorkProofInstruction, instructionId [32]byte) types.VerdictIdentity {
	return types.VerdictIdentity{
		SchemaVersion:     1,
		EscrowAddress:     instr.EscrowAddress,
		ChainId:           instr.ChainId,
		JobId:             instr.JobId,
		Attempt:           instr.Attempt,
		InstructionId:     instructionId,
		SpecHash:          instr.SpecHash,
		PrivateBundleHash: instr.PrivateBundleHash,
		ArtifactAddress:   instr.ArtifactAddress,
		ArtifactBlock:     instr.ArtifactBlock,
	}
}

func crossCheckInstruction(instr types.WorkProofInstruction, job workproofescrow.WorkProofEscrowJob) error {
	if job.Terms.SpecHash != instr.SpecHash {
		return fmt.Errorf("specHash mismatch")
	}
	if job.Terms.PrivateBundleHash != instr.PrivateBundleHash {
		return fmt.Errorf("privateBundleHash mismatch")
	}
	if job.Terms.CiphertextHash != instr.CiphertextHash {
		return fmt.Errorf("ciphertextHash mismatch")
	}
	if job.Terms.EngineVersionHash != instr.EngineVersionHash {
		return fmt.Errorf("engineVersionHash mismatch")
	}
	if job.Current.Attempt != instr.Attempt {
		return fmt.Errorf("attempt mismatch: on-chain %d, instruction %d", job.Current.Attempt, instr.Attempt)
	}
	if job.Current.ArtifactAddress != instr.ArtifactAddress {
		return fmt.Errorf("artifactAddress mismatch")
	}
	if job.Current.ArtifactBlock.Cmp(instr.ArtifactBlock) != 0 {
		return fmt.Errorf("artifactBlock mismatch")
	}
	if job.Current.ArtifactCodeHash != instr.ArtifactCodeHash {
		return fmt.Errorf("artifactCodeHash mismatch")
	}
	if job.Current.RandomRound.Cmp(instr.RandomRound) != 0 {
		return fmt.Errorf("randomRound mismatch")
	}
	if job.Current.RandomValueHash != instr.RandomValueHash {
		return fmt.Errorf("randomValueHash mismatch")
	}
	return nil
}

func (v *Verifier) recomputeArtifactCodeHash(ctx context.Context, artifact common.Address, block *big.Int) ([32]byte, error) {
	code, err := v.eth.CodeAt(ctx, artifact, block)
	if err != nil {
		return [32]byte{}, err
	}
	return keccak256(code), nil
}

// fetchAndDecryptBundle implements plan section 11 step 9 exactly: "The FCC
// extension downloads ciphertext, checks ciphertextHash, decrypts through
// the node's local /decrypt interface, canonicalizes again, and checks
// privateBundleHash." The fetch URL is deterministic --
// https://{primary configured gateway}/{ciphertextHash} -- since
// "content-addressed" already implies the path (see the JobTerms.ciphertextHash
// doc comment in WorkProofEscrow.sol).
func (v *Verifier) fetchAndDecryptBundle(ctx context.Context, instr types.WorkProofInstruction) (*WorkProofBundle, error) {
	if len(v.cfg.CiphertextHosts) == 0 {
		return nil, fmt.Errorf("no ciphertext gateway configured")
	}
	locator := fmt.Sprintf("https://%s/%s", v.cfg.CiphertextHosts[0], hex.EncodeToString(instr.CiphertextHash[:]))

	ciphertext, err := v.fetcher.Fetch(ctx, locator)
	if err != nil {
		return nil, fmt.Errorf("fetching ciphertext: %w", err)
	}
	if keccak256(ciphertext) != instr.CiphertextHash {
		return nil, fmt.Errorf("fetched ciphertext does not match committed ciphertextHash")
	}

	plaintext, err := v.decrypter.Decrypt(ctx, ciphertext)
	if err != nil {
		// Never wrap/log plaintext-derived content here (SPEC.md section 11:
		// plaintext must never enter a log).
		return nil, fmt.Errorf("decrypting bundle: %w", err)
	}
	if err := checkBundleSize(plaintext); err != nil {
		return nil, err
	}

	var bundle WorkProofBundle
	if err := json.Unmarshal(plaintext, &bundle); err != nil {
		return nil, fmt.Errorf("bundle is not valid JSON")
	}
	if err := bundle.Validate(); err != nil {
		return nil, fmt.Errorf("bundle failed schema validation: %w", err)
	}
	// bundle.Validate() only format-checks PublicSpecHash (it has no access
	// to the on-chain instruction). Without this cross-check a client could
	// submit a private bundle whose tests have nothing to do with the
	// public spec the job was actually created against -- PrivateBundleHash
	// alone only proves the bundle wasn't tampered with in transit, not that
	// it corresponds to the right public agreement.
	if err := checkPublicSpecHash(bundle.PublicSpecHash, instr.SpecHash); err != nil {
		return nil, err
	}

	_, hash, err := bundle.CanonicalizeAndHash()
	if err != nil {
		return nil, fmt.Errorf("canonicalizing bundle: %w", err)
	}
	if hash != instr.PrivateBundleHash {
		return nil, fmt.Errorf("canonicalized bundle does not match committed privateBundleHash")
	}

	return &bundle, nil
}

// checkBundleSize enforces config.MaxBundleBytes -- a resource limit that
// was previously validated as a schema constant but never actually
// enforced against anything. The ciphertext size is capped separately
// (config.MaxCiphertextBytes, in ciphertext.go), but nothing previously
// bounded the decrypted plaintext itself.
func checkBundleSize(plaintext []byte) error {
	if len(plaintext) > config.MaxBundleBytes {
		return fmt.Errorf("decrypted bundle exceeds max size of %d bytes", config.MaxBundleBytes)
	}
	return nil
}

// checkPublicSpecHash binds a decrypted bundle's declared publicSpecHash to
// the real on-chain-committed instr.SpecHash. bundle.Validate() only
// format-checks it as a well-formed bytes32 hex string -- this is the only
// place that can actually verify it against the agreement the job was
// created against, since the bundle is only ever visible after decryption.
func checkPublicSpecHash(bundlePublicSpecHash string, instrSpecHash [32]byte) error {
	decoded, err := hexutil.Decode(bundlePublicSpecHash)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("bundle publicSpecHash malformed")
	}
	if [32]byte(decoded) != instrSpecHash {
		return fmt.Errorf("bundle publicSpecHash does not match the job's committed specHash")
	}
	return nil
}

func countPassed(outcomes []VectorOutcome) uint32 {
	var n uint32
	for _, o := range outcomes {
		if o.Passed {
			n++
		}
	}
	return n
}

// PublicReport is the redacted, judge/UI-safe view of a verification run:
// no hidden inputs or expected outputs, only vector IDs/pass-fail/detail
// (SPEC.md "VerdictV1 Binding" / section 11).
type PublicReport struct {
	Vectors []PublicVectorResult `json:"vectors"`
}

type PublicVectorResult struct {
	ID      string `json:"id"`
	Passed  bool   `json:"passed"`
	Skipped bool   `json:"skipped"`
}

func reportHashFor(outcomes []VectorOutcome) ([32]byte, error) {
	report := PublicReport{Vectors: make([]PublicVectorResult, len(outcomes))}
	for i, o := range outcomes {
		report.Vectors[i] = PublicVectorResult{ID: o.ID, Passed: o.Passed, Skipped: o.Skipped}
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return [32]byte{}, err
	}
	canonical, err := jcs.Transform(raw)
	if err != nil {
		return [32]byte{}, err
	}
	return keccak256(canonical), nil
}

func keccak256AbiEncodedUint(n *big.Int) [32]byte {
	padded := make([]byte, 32)
	n.FillBytes(padded)
	return keccak256(padded)
}
