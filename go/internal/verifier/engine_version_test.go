package verifier

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"extension-scaffold/internal/config"
)

// Regression test for a real bug: EngineVersionHash was never actually
// computed from the running verifier's own config.WorkProofEngineVersion --
// verifier.go used to copy instr.EngineVersionHash straight into every
// verdict, so the "engine version" binding was pure theater (always echoed
// whatever a job was created with, never reflected the code that actually
// ran). This also regression-tests the encoding itself: EngineVersionHash
// is keccak256(WorkProofEngineVersion), the same convention already proven
// against real Solidity output in pkg/types/workproof_instruction_test.go
// (crypto.Keccak256Hash([]byte("workproof-verifier-v1"))) -- NOT Solidity's
// raw bytes32(string) encoding that OP_TYPE/OP_COMMAND use, which would be
// a silently wrong, non-interchangeable value here.
func TestRealEngineVersionHashMatchesByteProvenConvention(t *testing.T) {
	want := crypto.Keccak256Hash([]byte(config.WorkProofEngineVersion))
	if realEngineVersionHash != want {
		t.Fatalf("realEngineVersionHash = %x, want keccak256(%q) = %x", realEngineVersionHash, config.WorkProofEngineVersion, want)
	}
}
