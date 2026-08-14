package verifier

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"extension-scaffold/pkg/types"
)

func TestInfrastructureInconclusiveVerdictHasSettlementSafeCounts(t *testing.T) {
	instr := types.WorkProofInstruction{
		ArtifactAddress: common.HexToAddress("0x1000000000000000000000000000000000000001"),
		ArtifactBlock:   big.NewInt(123),
		RandomRound:     big.NewInt(456),
		ExpiresAt:       999,
	}
	identity := types.VerdictIdentity{
		SchemaVersion: 1,
		EscrowAddress: common.HexToAddress("0x2000000000000000000000000000000000000002"),
		ChainId:       big.NewInt(114),
		JobId:         big.NewInt(7),
		Attempt:       1,
	}

	verdict, err := (&Verifier{}).inconclusiveVerdict(instr, identity, "test infrastructure failure")
	if err != nil {
		t.Fatalf("inconclusiveVerdict returned error: %v", err)
	}
	if verdict.Result.Outcome != uint8(types.OutcomeInconclusive) {
		t.Fatalf("outcome = %d, want Inconclusive", verdict.Result.Outcome)
	}
	if verdict.Result.ExecutedCount == 0 {
		t.Fatal("infrastructure inconclusive verdict must represent the skipped infrastructure result")
	}
	if verdict.Result.PassedCount >= verdict.Result.ExecutedCount {
		t.Fatalf("inconclusive counts must not look like PASS: passed=%d executed=%d", verdict.Result.PassedCount, verdict.Result.ExecutedCount)
	}
}
