package types

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	abicoder "github.com/flare-foundation/go-flare-common/pkg/abicoder"
)

// TestWorkProofInstructionDecodesRealSolidityEncoding cross-verifies the Go
// ABI definition against bytes actually produced by Solidity's
// abi.encode(instr) (test/GenInstructionVector.t.sol), not hand-derived.
func TestWorkProofInstructionDecodesRealSolidityEncoding(t *testing.T) {
	dataHex := "00000000000000000000000000000000000000000000000000000000000000720000000000000000000000001234567890123456789012345678901234567890000000000000000000000000000000000000000000000000000000000000002a00000000000000000000000000000000000000000000000000000000000000038befe5913be4edba4ceda78f9be44236197ccc73196bea55cafeff0bfecbc0618d0a20ebe6abc1658d0e8a4c557ee1a5d10567f4586f168af2e389c34ad9c8b9883c4514cd77782c55f9622ab859977d7e7076c89806247409850100dce38bff0000000000000000000000004a1af2c21763d225fdacb9e070c2234ad834feae0000000000000000000000000000000000000000000000000000000000087a239d2202aa3fa84102814b1f5946491a0126e5b16bfedb867369fd904fc211d16e00000000000000000000000000000000000000000000000000000000000f120629fe47912cc540a72dff4779db65a9d86a13eb11150e6b52287e4c4a47ca9b6273e768e07bbd1d343ebac8e803e994fc0af15a4a3a14597c7e3d9d40793cb20a000000000000000000000000000000000000000000000000000000006554180f"
	data, err := hex.DecodeString(dataHex)
	if err != nil {
		t.Fatalf("decoding hex fixture: %v", err)
	}

	var instr WorkProofInstruction
	if err := abicoder.DecodeTo(WorkProofInstructionArg, data, &instr); err != nil {
		t.Fatalf("DecodeTo: %v", err)
	}

	if instr.ChainId.Cmp(big.NewInt(114)) != 0 {
		t.Errorf("ChainId = %s, want 114", instr.ChainId)
	}
	if instr.EscrowAddress != common.HexToAddress("0x1234567890123456789012345678901234567890") {
		t.Errorf("EscrowAddress mismatch: %s", instr.EscrowAddress.Hex())
	}
	if instr.JobId.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("JobId = %s, want 42", instr.JobId)
	}
	if instr.Attempt != 3 {
		t.Errorf("Attempt = %d, want 3", instr.Attempt)
	}
	if common.BytesToHash(instr.SpecHash[:]) != crypto.Keccak256Hash([]byte("spec-2")) {
		t.Errorf("SpecHash mismatch")
	}
	if common.BytesToHash(instr.PrivateBundleHash[:]) != crypto.Keccak256Hash([]byte("bundle-2")) {
		t.Errorf("PrivateBundleHash mismatch")
	}
	if common.BytesToHash(instr.CiphertextHash[:]) != crypto.Keccak256Hash([]byte("ciphertext-2")) {
		t.Errorf("CiphertextHash mismatch")
	}
	if instr.ArtifactAddress != common.HexToAddress("0x4A1af2C21763D225FdAcb9E070c2234Ad834FeAe") {
		t.Errorf("ArtifactAddress mismatch")
	}
	if instr.ArtifactBlock.Cmp(big.NewInt(555555)) != 0 {
		t.Errorf("ArtifactBlock = %s, want 555555", instr.ArtifactBlock)
	}
	if common.BytesToHash(instr.ArtifactCodeHash[:]) != crypto.Keccak256Hash([]byte("code-2")) {
		t.Errorf("ArtifactCodeHash mismatch")
	}
	if instr.RandomRound.Cmp(big.NewInt(987654)) != 0 {
		t.Errorf("RandomRound = %s, want 987654", instr.RandomRound)
	}
	if common.BytesToHash(instr.RandomValueHash[:]) != crypto.Keccak256Hash([]byte("random-2")) {
		t.Errorf("RandomValueHash mismatch")
	}
	if common.BytesToHash(instr.EngineVersionHash[:]) != crypto.Keccak256Hash([]byte("workproof-verifier-v1")) {
		t.Errorf("EngineVersionHash mismatch")
	}
	if instr.ExpiresAt != 1700009999 {
		t.Errorf("ExpiresAt = %d, want 1700009999", instr.ExpiresAt)
	}

	// Round-trip: re-encoding must reproduce the exact original Solidity bytes.
	encoded, err := abicoder.Encode(WorkProofInstructionArg, instr)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if hex.EncodeToString(encoded) != dataHex {
		t.Errorf("round-trip mismatch:\ngot:  0x%s\nwant: 0x%s", hex.EncodeToString(encoded), dataHex)
	}
}
