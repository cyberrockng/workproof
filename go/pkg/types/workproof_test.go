package types

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	abicoder "github.com/flare-foundation/go-flare-common/pkg/abicoder"
)

// TestVerdictV1DecodesRealSolidityEncoding cross-verifies the Go ABI
// definition against bytes actually produced by Solidity's abi.encode(v)
// (test/GenVerdictVector.t.sol), not hand-derived or assumed compatible.
func TestVerdictV1DecodesRealSolidityEncoding(t *testing.T) {
	dataHex := "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000001234567890123456789012345678901234567890000000000000000000000000000000000000000000000000000000000000007200000000000000000000000000000000000000000000000000000000000000070000000000000000000000000000000000000000000000000000000000000002e3579b421888727274c8dfc9bf9185594c1c84dea3844f69556fc5d3dcb108fca9b21596bdda2cf62f7fa138a5fceb65c0d7c13b7836ab2ae89143e4df7c5811a916eeadfac2e468108ab83bb8b3872e52cdfba8f979aa7526f75e1a37849bd10000000000000000000000004a1af2c21763d225fdacb9e070c2234ad834feae00000000000000000000000000000000000000000000000000000000000f423fedab425a4d17b30d323b2fbb52419dafb1d6e21005138f8b086a2b05a328c9f7000000000000000000000000000000000000000000000000000000000001e2406ed040f474a3b0b0df1940b844f391704c779173f365f3db910f7dde29094cc273e768e07bbd1d343ebac8e803e994fc0af15a4a3a14597c7e3d9d40793cb20a0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000055566ad18f48e77044b8f87c0753c69b75f06f85fc6308f1ad98218ee0b25a3a4000000000000000000000000000000000000000000000000000000006553f100000000000000000000000000000000000000000000000000000000006553ff10"
	data, err := hex.DecodeString(dataHex)
	if err != nil {
		t.Fatalf("decoding hex fixture: %v", err)
	}

	var v VerdictV1
	if err := abicoder.DecodeTo(VerdictV1Arg, data, &v); err != nil {
		t.Fatalf("DecodeTo: %v", err)
	}

	if v.Id.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", v.Id.SchemaVersion)
	}
	if v.Id.EscrowAddress != common.HexToAddress("0x1234567890123456789012345678901234567890") {
		t.Errorf("EscrowAddress = %s, want 0x1234567890123456789012345678901234567890", v.Id.EscrowAddress.Hex())
	}
	if v.Id.ChainId.Cmp(big.NewInt(114)) != 0 {
		t.Errorf("ChainId = %s, want 114", v.Id.ChainId)
	}
	if v.Id.JobId.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("JobId = %s, want 7", v.Id.JobId)
	}
	if v.Id.Attempt != 2 {
		t.Errorf("Attempt = %d, want 2", v.Id.Attempt)
	}
	if common.BytesToHash(v.Id.InstructionId[:]) != common.HexToHash("0xe3579b421888727274c8dfc9bf9185594c1c84dea3844f69556fc5d3dcb108fc") {
		t.Errorf("InstructionId mismatch")
	}
	if common.BytesToHash(v.Id.SpecHash[:]) != common.HexToHash("0xa9b21596bdda2cf62f7fa138a5fceb65c0d7c13b7836ab2ae89143e4df7c5811") {
		t.Errorf("SpecHash mismatch")
	}
	if common.BytesToHash(v.Id.PrivateBundleHash[:]) != common.HexToHash("0xa916eeadfac2e468108ab83bb8b3872e52cdfba8f979aa7526f75e1a37849bd1") {
		t.Errorf("PrivateBundleHash mismatch")
	}
	if v.Id.ArtifactAddress != common.HexToAddress("0x4A1af2C21763D225FdAcb9E070c2234Ad834FeAe") {
		t.Errorf("ArtifactAddress mismatch")
	}
	if v.Id.ArtifactBlock.Cmp(big.NewInt(999999)) != 0 {
		t.Errorf("ArtifactBlock = %s, want 999999", v.Id.ArtifactBlock)
	}
	if common.BytesToHash(v.Result.ArtifactCodeHash[:]) != common.HexToHash("0xedab425a4d17b30d323b2fbb52419dafb1d6e21005138f8b086a2b05a328c9f7") {
		t.Errorf("ArtifactCodeHash mismatch")
	}
	if v.Result.RandomRound.Cmp(big.NewInt(123456)) != 0 {
		t.Errorf("RandomRound = %s, want 123456", v.Result.RandomRound)
	}
	if common.BytesToHash(v.Result.RandomValueHash[:]) != common.HexToHash("0x6ed040f474a3b0b0df1940b844f391704c779173f365f3db910f7dde29094cc2") {
		t.Errorf("RandomValueHash mismatch")
	}
	if common.BytesToHash(v.Result.EngineVersionHash[:]) != common.HexToHash("0x73e768e07bbd1d343ebac8e803e994fc0af15a4a3a14597c7e3d9d40793cb20a") {
		t.Errorf("EngineVersionHash mismatch")
	}
	if v.Result.Outcome != 1 {
		t.Errorf("Outcome = %d, want 1 (Fail)", v.Result.Outcome)
	}
	if v.Result.PassedCount != 3 {
		t.Errorf("PassedCount = %d, want 3", v.Result.PassedCount)
	}
	if v.Result.ExecutedCount != 5 {
		t.Errorf("ExecutedCount = %d, want 5", v.Result.ExecutedCount)
	}
	if common.BytesToHash(v.Result.ReportHash[:]) != common.HexToHash("0x5566ad18f48e77044b8f87c0753c69b75f06f85fc6308f1ad98218ee0b25a3a4") {
		t.Errorf("ReportHash mismatch")
	}
	if v.Result.IssuedAt != 1700000000 {
		t.Errorf("IssuedAt = %d, want 1700000000", v.Result.IssuedAt)
	}
	if v.Result.ExpiresAt != 1700003600 {
		t.Errorf("ExpiresAt = %d, want 1700003600", v.Result.ExpiresAt)
	}

	// Round-trip: re-encoding must reproduce the exact original Solidity bytes.
	encoded, err := abicoder.Encode(VerdictV1Arg, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if hex.EncodeToString(encoded) != dataHex {
		t.Errorf("round-trip mismatch:\ngot:  0x%s\nwant: 0x%s", hex.EncodeToString(encoded), dataHex)
	}
}
