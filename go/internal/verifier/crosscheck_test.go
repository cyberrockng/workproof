package verifier

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"extension-scaffold/pkg/contracts/workproofescrow"
	"extension-scaffold/pkg/types"
)

func matchingInstructionAndJob() (types.WorkProofInstruction, workproofescrow.WorkProofEscrowJob) {
	instr := types.WorkProofInstruction{
		ChainId:           big.NewInt(114),
		EscrowAddress:     common.HexToAddress("0x1000000000000000000000000000000000000001"),
		JobId:             big.NewInt(7),
		Attempt:           2,
		SpecHash:          keccak256([]byte("spec")),
		PrivateBundleHash: keccak256([]byte("bundle")),
		CiphertextHash:    keccak256([]byte("ciphertext")),
		ArtifactAddress:   common.HexToAddress("0x2000000000000000000000000000000000000002"),
		ArtifactBlock:     big.NewInt(123),
		ArtifactCodeHash:  keccak256([]byte("code")),
		RandomRound:       big.NewInt(456),
		RandomValueHash:   keccak256([]byte("random")),
		EngineVersionHash: realEngineVersionHash,
		ExpiresAt:         999_999_999_999,
	}
	job := workproofescrow.WorkProofEscrowJob{
		Terms: workproofescrow.WorkProofEscrowJobTerms{
			SpecHash:          instr.SpecHash,
			PrivateBundleHash: instr.PrivateBundleHash,
			CiphertextHash:    instr.CiphertextHash,
			EngineVersionHash: instr.EngineVersionHash,
			GraceEnds:         instr.ExpiresAt,
		},
		Current: workproofescrow.WorkProofEscrowAttemptState{
			Attempt:          instr.Attempt,
			ArtifactAddress:  instr.ArtifactAddress,
			ArtifactBlock:    instr.ArtifactBlock,
			ArtifactCodeHash: instr.ArtifactCodeHash,
			RandomRound:      instr.RandomRound,
			RandomValueHash:  instr.RandomValueHash,
		},
		State: workProofStateVerifying,
	}
	return instr, job
}

func TestCrossCheckInstructionRejectsNonVerifyingState(t *testing.T) {
	instr, job := matchingInstructionAndJob()
	job.State = 3
	if err := crossCheckInstruction(instr, job); err == nil {
		t.Fatal("expected non-verifying job state to be rejected")
	}
}

func TestCrossCheckInstructionRejectsSettledJob(t *testing.T) {
	instr, job := matchingInstructionAndJob()
	job.Settled = true
	if err := crossCheckInstruction(instr, job); err == nil {
		t.Fatal("expected settled job to be rejected")
	}
}

func TestCrossCheckInstructionRejectsExpiryMismatch(t *testing.T) {
	instr, job := matchingInstructionAndJob()
	job.Terms.GraceEnds = instr.ExpiresAt + 1
	if err := crossCheckInstruction(instr, job); err == nil {
		t.Fatal("expected expiry mismatch to be rejected")
	}
}

func TestChainIDMatchesRejectsUint64Truncation(t *testing.T) {
	huge := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(114))
	if chainIDMatches(huge, 114) {
		t.Fatal("huge chain id ending in 114 must not match Coston2")
	}
	if !chainIDMatches(big.NewInt(114), 114) {
		t.Fatal("exact Coston2 chain id should match")
	}
}
