// Package config contains configuration values and defaults used by the extension.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Version = "0.1.0"

	OPTypeGreeting      = "GREETING"
	OPCommandSayHello   = "SAY_HELLO"
	OPCommandSayGoodbye = "SAY_GOODBYE"

	// OPTypeWorkProof/OPCommandVerify must match contracts/WorkProofEscrow.sol's
	// OP_TYPE/OP_COMMAND constants exactly (both bytes32("WORKPROOF")/
	// bytes32("VERIFY")). Plan section 15 Phase 4 task 1.
	OPTypeWorkProof = "WORKPROOF"
	OPCommandVerify = "VERIFY"

	// WorkProofEngineVersion is hashed as keccak256(WorkProofEngineVersion)
	// (see internal/verifier.realEngineVersionHash) into every
	// VerdictV1.EngineVersionHash -- NOT Solidity's raw bytes32(string)
	// encoding (that's what OP_TYPE/OP_COMMAND use instead; the two
	// encodings are not interchangeable in this codebase). This exact
	// keccak256 convention is byte-proven against real Solidity
	// abi.encode(...) output in pkg/types/workproof_instruction_test.go. A
	// job's EngineVersionHash is frozen at createJob time and checked
	// byte-exact at settlement (WorkProofEscrow.sol _checkOutcomeBinding),
	// so bump this whenever verifier behavior or the wire format changes
	// (plan Phase 4 task 13) -- an old job pinned to a stale version simply
	// can never settle against a newer engine, by design.
	WorkProofEngineVersion = "workproof-verifier-v1"

	TimeoutShutdown = 5 * time.Second

	ServerReadHeaderTimeout = 5 * time.Second
	ServerReadTimeout       = 30 * time.Second
	ServerWriteTimeout      = 30 * time.Second
	ServerIdleTimeout       = 60 * time.Second
)

// WorkProof resource limits (plan section 11 "Resource limits").
const (
	MaxBundleBytes            = 64 * 1024
	MaxCiphertextBytes        = 128 * 1024
	MaxVectorCount            = 128
	MaxSelectionCount         = 32
	RPCCallTimeout            = 5 * time.Second
	AttemptTotalTimeout       = 30 * time.Second
	CallGasCap          int64 = 2_000_000
	MaxResponseBytes          = 8 * 1024
	MaxActionBodyBytes        = 1 << 20
)

// Defaults.
var (
	ExtensionPort = 8080
	SignPort      = 9090

	// WorkProof-specific: not part of the generic scaffold's env surface
	// (docs/extension-contract.md's env table only documents the vars
	// tee-node itself consumes plus EXTENSION_PORT/SIGN_PORT), because only
	// the WorkProof handler needs its own chain RPC access and contract
	// address -- the greeting handlers never called out to the chain at all.
	WorkProofRPCURL             string
	WorkProofEscrowAddress      string
	WorkProofRandomNumberV2Addr string
	WorkProofCiphertextHosts    []string
)

// Environment variables override defaults.
func init() {
	ep := os.Getenv("EXTENSION_PORT")
	sp := os.Getenv("SIGN_PORT")

	if ep != "" {
		if v, err := strconv.Atoi(ep); err == nil {
			ExtensionPort = v
		}
	}
	if sp != "" {
		if v, err := strconv.Atoi(sp); err == nil {
			SignPort = v
		}
	}

	WorkProofRPCURL = os.Getenv("WORKPROOF_RPC_URL")
	WorkProofEscrowAddress = os.Getenv("WORKPROOF_ESCROW_ADDRESS")
	WorkProofRandomNumberV2Addr = os.Getenv("WORKPROOF_RANDOM_NUMBER_V2_ADDRESS")
	if hosts := os.Getenv("WORKPROOF_CIPHERTEXT_HOSTS"); hosts != "" {
		WorkProofCiphertextHosts = strings.Split(hosts, ",")
	}
}
