package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Outcome mirrors contracts/WorkProofEscrow.sol's Outcome enum ABI encoding
// (uint8, declaration order Pass/Fail/Inconclusive).
type Outcome uint8

const (
	OutcomePass Outcome = iota
	OutcomeFail
	OutcomeInconclusive
)

// Field naming note (applies to every struct below): go-ethereum's abi.Pack
// matches Go struct fields to ABI tuple components by name, capitalizing
// only the ABI name's first rune (abi.ToCamelCase: "chainId" -> "ChainId",
// "id" -> "Id") -- NOT idiomatic Go acronym casing ("ChainID", "Id" would
// naturally be written "ID"). abicoder.Decode is positional and would
// tolerate either spelling, but Encode is not: using "ChainID"/"JobID"/
// "InstructionID"/"ID" here breaks round-trip encoding with "field not
// found in struct" (caught by TestVerdictV1DecodesRealSolidityEncoding's
// round-trip assertion). Do not "fix" this casing back to idiomatic Go.

// WorkProofInstruction mirrors contracts/WorkProofEscrow.sol's
// WorkProofInstruction struct field-for-field, in declaration order --
// abicoder.Decode maps tuple components to struct fields positionally, not
// by name, so field order here must exactly match the Solidity struct.
type WorkProofInstruction struct {
	ChainId           *big.Int
	EscrowAddress     common.Address
	JobId             *big.Int
	Attempt           uint64
	SpecHash          [32]byte
	PrivateBundleHash [32]byte
	CiphertextHash    [32]byte
	ArtifactAddress   common.Address
	ArtifactBlock     *big.Int
	ArtifactCodeHash  [32]byte
	RandomRound       *big.Int
	RandomValueHash   [32]byte
	EngineVersionHash [32]byte
	ExpiresAt         uint64
}

// VerdictIdentity mirrors WorkProofEscrow.sol's VerdictIdentity sub-struct.
type VerdictIdentity struct {
	SchemaVersion     uint8
	EscrowAddress     common.Address
	ChainId           *big.Int
	JobId             *big.Int
	Attempt           uint64
	InstructionId     [32]byte
	SpecHash          [32]byte
	PrivateBundleHash [32]byte
	ArtifactAddress   common.Address
	ArtifactBlock     *big.Int
}

// VerdictOutcome mirrors WorkProofEscrow.sol's VerdictOutcome sub-struct.
type VerdictOutcome struct {
	ArtifactCodeHash  [32]byte
	RandomRound       *big.Int
	RandomValueHash   [32]byte
	EngineVersionHash [32]byte
	Outcome           uint8
	PassedCount       uint32
	ExecutedCount     uint32
	ReportHash        [32]byte
	IssuedAt          uint64
	ExpiresAt         uint64
}

// VerdictV1 mirrors WorkProofEscrow.sol's VerdictV1 struct (VerdictIdentity
// + VerdictOutcome). The Solidity-side nested-struct split was done purely
// to dodge an IR stack-depth limit on abi.decode; since both sub-structs are
// fully static (no dynamic bytes/string/array members), the ABI wire
// encoding is byte-identical to a flat 20-field tuple, and this Go type
// produces that same encoding without needing to know about the split.
type VerdictV1 struct {
	Id     VerdictIdentity
	Result VerdictOutcome
}

// WorkProofInstructionArg is the abi.Argument for WorkProofInstruction,
// matching contracts/WorkProofEscrow.sol's struct exactly.
var WorkProofInstructionArg abi.Argument

// VerdictV1Arg is the abi.Argument for VerdictV1, matching
// contracts/WorkProofEscrow.sol's struct exactly.
var VerdictV1Arg abi.Argument

func init() {
	instructionTy, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "chainId", Type: "uint256"},
		{Name: "escrowAddress", Type: "address"},
		{Name: "jobId", Type: "uint256"},
		{Name: "attempt", Type: "uint64"},
		{Name: "specHash", Type: "bytes32"},
		{Name: "privateBundleHash", Type: "bytes32"},
		{Name: "ciphertextHash", Type: "bytes32"},
		{Name: "artifactAddress", Type: "address"},
		{Name: "artifactBlock", Type: "uint256"},
		{Name: "artifactCodeHash", Type: "bytes32"},
		{Name: "randomRound", Type: "uint256"},
		{Name: "randomValueHash", Type: "bytes32"},
		{Name: "engineVersionHash", Type: "bytes32"},
		{Name: "expiresAt", Type: "uint64"},
	})
	if err != nil {
		panic(err)
	}
	WorkProofInstructionArg = abi.Argument{Type: instructionTy}

	identityComponents := []abi.ArgumentMarshaling{
		{Name: "schemaVersion", Type: "uint8"},
		{Name: "escrowAddress", Type: "address"},
		{Name: "chainId", Type: "uint256"},
		{Name: "jobId", Type: "uint256"},
		{Name: "attempt", Type: "uint64"},
		{Name: "instructionId", Type: "bytes32"},
		{Name: "specHash", Type: "bytes32"},
		{Name: "privateBundleHash", Type: "bytes32"},
		{Name: "artifactAddress", Type: "address"},
		{Name: "artifactBlock", Type: "uint256"},
	}
	outcomeComponents := []abi.ArgumentMarshaling{
		{Name: "artifactCodeHash", Type: "bytes32"},
		{Name: "randomRound", Type: "uint256"},
		{Name: "randomValueHash", Type: "bytes32"},
		{Name: "engineVersionHash", Type: "bytes32"},
		{Name: "outcome", Type: "uint8"},
		{Name: "passedCount", Type: "uint32"},
		{Name: "executedCount", Type: "uint32"},
		{Name: "reportHash", Type: "bytes32"},
		{Name: "issuedAt", Type: "uint64"},
		{Name: "expiresAt", Type: "uint64"},
	}
	verdictTy, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "id", Type: "tuple", Components: identityComponents},
		{Name: "result", Type: "tuple", Components: outcomeComponents},
	})
	if err != nil {
		panic(err)
	}
	VerdictV1Arg = abi.Argument{Type: verdictTy}
}
