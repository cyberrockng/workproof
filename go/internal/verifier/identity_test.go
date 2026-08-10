package verifier

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"extension-scaffold/pkg/types"
)

// This is a direct regression test for a real bug: verdictIdentityFrom used
// to build VerdictIdentity without ever setting InstructionId, leaving it
// the zero value for every real result. WorkProofEscrow.sol's
// _decodeAndAuthenticate reconstructs the signature digest using
// v.id.instructionId extracted from the decoded VerdictV1 (FccVerdict.sol's
// actionResultHash takes id directly) -- if that doesn't match the real
// ActionResult.ID tee-node actually signed with, ecrecover returns the
// wrong signer and settlement reverts unconditionally. See
// test/WorkProofEscrow.t.sol's settle helper (line ~167), which already
// proves the Solidity side accepts a correctly-bound instructionId and
// testMutationRejected_instructionId, which proves it rejects a wrong one --
// this test proves the Go side now actually produces the correct value.
func TestVerdictIdentityFromThreadsRealInstructionId(t *testing.T) {
	instr := types.WorkProofInstruction{
		ChainId:           big.NewInt(114),
		EscrowAddress:     common.HexToAddress("0x1111111111111111111111111111111111111111"),
		JobId:             big.NewInt(7),
		Attempt:           2,
		SpecHash:          [32]byte{0x01},
		PrivateBundleHash: [32]byte{0x02},
		CiphertextHash:    [32]byte{0x03},
		ArtifactAddress:   common.HexToAddress("0x2222222222222222222222222222222222222222"),
		ArtifactBlock:     big.NewInt(1000),
		ArtifactCodeHash:  [32]byte{0x04},
	}
	instructionId := [32]byte{0xde, 0xad, 0xbe, 0xef}

	identity := verdictIdentityFrom(instr, instructionId)

	if identity.InstructionId != instructionId {
		t.Fatalf("InstructionId not threaded through: got %x, want %x", identity.InstructionId, instructionId)
	}
	if identity.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", identity.SchemaVersion)
	}
	if identity.EscrowAddress != instr.EscrowAddress {
		t.Errorf("EscrowAddress mismatch")
	}
	if identity.ChainId.Cmp(instr.ChainId) != 0 {
		t.Errorf("ChainId mismatch")
	}
	if identity.JobId.Cmp(instr.JobId) != 0 {
		t.Errorf("JobId mismatch")
	}
	if identity.Attempt != instr.Attempt {
		t.Errorf("Attempt mismatch")
	}
	if identity.SpecHash != instr.SpecHash {
		t.Errorf("SpecHash mismatch")
	}
	if identity.PrivateBundleHash != instr.PrivateBundleHash {
		t.Errorf("PrivateBundleHash mismatch")
	}
	if identity.ArtifactAddress != instr.ArtifactAddress {
		t.Errorf("ArtifactAddress mismatch")
	}
	if identity.ArtifactBlock.Cmp(instr.ArtifactBlock) != 0 {
		t.Errorf("ArtifactBlock mismatch")
	}

	// A zero instructionId (the pre-fix bug) must never be silently accepted
	// as "correct" by this test -- guard against a regression where someone
	// "fixes" this by defaulting instructionId to the zero value again.
	if identity.InstructionId == ([32]byte{}) {
		t.Fatalf("InstructionId is the zero value -- this is exactly the bug being regression-tested")
	}
}
