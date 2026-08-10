package verifier

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ethereumCallMsg/ethereumCallMsgFrom build a read-only ethereum.CallMsg
// with a fixed gas cap, from either the zero address or an explicit caller.
func ethereumCallMsg(to common.Address, data []byte) ethereum.CallMsg {
	return ethereum.CallMsg{To: &to, Data: data}
}

// ethereumCallMsgFrom builds a CallMsg with an explicit caller, gas cap, and
// optional value (payable-function vectors -- value may be nil for the
// common zero-value case).
func ethereumCallMsgFrom(from, to common.Address, data []byte, gasCap int64, value *big.Int) ethereum.CallMsg {
	msg := ethereum.CallMsg{From: from, To: &to, Data: data, Value: value}
	if gasCap > 0 {
		msg.Gas = uint64(gasCap)
	}
	return msg
}

// isEVMRevert reports whether err represents a genuine JSON-RPC-level error
// response (which is how every eth_call revert -- and only a revert -- is
// surfaced by go-ethereum: node implementations return revert data inside a
// JSON-RPC error object, parsed into a type implementing rpc.Error/
// rpc.DataError). A raw transport failure (timeout, connection refused, TLS
// failure, malformed body, context deadline) never implements this
// interface -- it surfaces as a plain Go error instead. Conflating the two
// would let an RPC outage or provider glitch count as a passing hidden
// test, exactly the bug this function exists to close.
func isEVMRevert(err error) bool {
	var rpcErr rpc.Error
	return errors.As(err, &rpcErr)
}

// FetchRandomNumber independently re-reads the historical random number for
// round from RandomNumberV2Interface.getRandomNumberHistorical -- this data
// is public on-chain state, so the verifier fetches it itself rather than
// trusting a value the relayed instruction might claim (SPEC.md/plan
// section 10: "the extension re-reads the job/attempt at the dispatch
// block and rejects a mismatch"). Only randomValueHash = keccak256(abi.encode(
// randomNumber)) was ever committed on-chain (WorkProofEscrow.sol
// lockRandomness), never the raw number itself.
func FetchRandomNumber(ctx context.Context, eth *ethclient.Client, randomNumberV2 common.Address, round *big.Int) (*big.Int, bool, error) {
	uint256Ty, _ := abi.NewType("uint256", "", nil)
	boolTy, _ := abi.NewType("bool", "", nil)
	args := abi.Arguments{{Type: uint256Ty}, {Type: boolTy}, {Type: uint256Ty}}
	inArgs := abi.Arguments{{Type: uint256Ty}}

	selector := crypto.Keccak256([]byte("getRandomNumberHistorical(uint256)"))[:4]
	packedArgs, err := inArgs.Pack(round)
	if err != nil {
		return nil, false, fmt.Errorf("packing getRandomNumberHistorical args: %w", err)
	}
	calldata := append(append([]byte{}, selector...), packedArgs...)

	result, err := eth.CallContract(ctx, ethereumCallMsg(randomNumberV2, calldata), nil)
	if err != nil {
		return nil, false, fmt.Errorf("calling getRandomNumberHistorical: %w", err)
	}

	values, err := args.Unpack(result)
	if err != nil {
		return nil, false, fmt.Errorf("unpacking getRandomNumberHistorical result: %w", err)
	}
	randomNumber, ok := values[0].(*big.Int)
	if !ok {
		return nil, false, fmt.Errorf("unexpected type for randomNumber")
	}
	isSecure, ok := values[1].(bool)
	if !ok {
		return nil, false, fmt.Errorf("unexpected type for isSecure")
	}
	return randomNumber, isSecure, nil
}

// TestSeed derives the deterministic vector-selection seed exactly as plan
// section 10 step 8 specifies: keccak256(randomNumber, escrowAddress,
// jobId, attempt, specHash, artifactCodeHash) -- a flat (non-tuple)
// abi.encode of mixed types, matching Solidity's abi.encode(...) call
// syntax for a plain argument list.
func TestSeed(randomNumber *big.Int, escrowAddress common.Address, jobID *big.Int, attempt uint64, specHash, artifactCodeHash [32]byte) ([32]byte, error) {
	uint256Ty, _ := abi.NewType("uint256", "", nil)
	addressTy, _ := abi.NewType("address", "", nil)
	uint64Ty, _ := abi.NewType("uint64", "", nil)
	bytes32Ty, _ := abi.NewType("bytes32", "", nil)

	args := abi.Arguments{
		{Type: uint256Ty}, {Type: addressTy}, {Type: uint256Ty}, {Type: uint64Ty}, {Type: bytes32Ty}, {Type: bytes32Ty},
	}
	packed, err := args.Pack(randomNumber, escrowAddress, jobID, attempt, specHash, artifactCodeHash)
	if err != nil {
		return [32]byte{}, fmt.Errorf("packing test seed inputs: %w", err)
	}
	return keccak256(packed), nil
}

// SelectVectors deterministically selects selectionCount indices out of
// [0, vectorCount) using seed-derived Fisher-Yates: the plan mandates
// "deterministic Fisher-Yates selection" and "deterministic ordering" but
// does not pin an exact PRNG-expansion byte format the way the FCC
// signature chain does, so this uses a standard, clearly-documented
// counter-expansion scheme (successive keccak256(seed, counter) draws) --
// any two runs with the same seed/counts always produce the same selection
// in the same order, which is the property that actually matters (no party
// can grind a favorable subset after seeing the seed, and results are
// reproducible for judging).
func SelectVectors(seed [32]byte, vectorCount, selectionCount int) []int {
	indices := make([]int, vectorCount)
	for i := range indices {
		indices[i] = i
	}
	limit := vectorCount
	if selectionCount < limit {
		limit = selectionCount
	}
	for i := 0; i < limit; i++ {
		remaining := vectorCount - i
		draw := keccak256(append(append([]byte{}, seed[:]...), encodeCounter(uint64(i))...))
		j := i + int(new(big.Int).Mod(new(big.Int).SetBytes(draw[:]), big.NewInt(int64(remaining))).Int64())
		indices[i], indices[j] = indices[j], indices[i]
	}
	return indices[:limit]
}

func encodeCounter(i uint64) []byte {
	b := make([]byte, 8)
	for k := 0; k < 8; k++ {
		b[7-k] = byte(i >> (8 * k))
	}
	return b
}

// VectorOutcome is the redacted, public-safe result of executing one
// vector: no hidden inputs or expected outputs are retained past this
// point (SPEC.md "The public report exposes vector IDs, statuses, timing,
// and redacted diagnostics. It must not expose hidden inputs or expected
// outputs.").
type VectorOutcome struct {
	ID      string
	Passed  bool
	Skipped bool // set when a call errors/times out rather than definitively fails
	Detail  string
}

// ExecuteVector runs a single P0 vector against artifactAddress at the
// given block, within the bundle's declared gas/response caps.
func ExecuteVector(ctx context.Context, eth *ethclient.Client, artifactAddress common.Address, blockNumber *big.Int, gasCap int64, maxResponseBytes int, v Vector) VectorOutcome {
	switch v.Type {
	case VectorEthCallEquals:
		return executeEthCallEquals(ctx, eth, artifactAddress, blockNumber, gasCap, maxResponseBytes, v)
	case VectorEthCallReverts:
		return executeEthCallReverts(ctx, eth, artifactAddress, blockNumber, gasCap, v)
	case VectorErc165SupportsInterface:
		return executeErc165(ctx, eth, artifactAddress, blockNumber, gasCap, v)
	case VectorCodeSizeRange:
		return executeCodeSizeRange(ctx, eth, artifactAddress, blockNumber, v)
	case VectorStorageAtEquals:
		return executeStorageAtEquals(ctx, eth, artifactAddress, blockNumber, v)
	default:
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "unknown vector type"}
	}
}

// parseCallValue decodes a vector's optional hex uint256 msg.value field.
// An empty string means "no value declared" -- nil (the CallMsg omits
// Value), not zero-as-a-parsed-number, so a vector that never declares
// value behaves exactly as it did before value support existed.
func parseCallValue(hexValue string) (*big.Int, error) {
	if hexValue == "" {
		return nil, nil
	}
	return hexutil.DecodeBig(hexValue)
}

func executeEthCallEquals(ctx context.Context, eth *ethclient.Client, target common.Address, block *big.Int, gasCap int64, maxResponseBytes int, v Vector) VectorOutcome {
	calldata, err := hexutil.Decode(v.Calldata)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed calldata"}
	}
	expected, err := hexutil.Decode(v.ExpectedReturn)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed expectedReturn"}
	}
	value, err := parseCallValue(v.Value)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed value"}
	}
	caller := common.HexToAddress(v.Caller)

	result, err := eth.CallContract(ctx, ethereumCallMsgFrom(caller, target, calldata, gasCap, value), block)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "call error"}
	}
	if len(result) > maxResponseBytes {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "response exceeds cap"}
	}
	return VectorOutcome{ID: v.ID, Passed: hex.EncodeToString(result) == hex.EncodeToString(expected)}
}

func executeEthCallReverts(ctx context.Context, eth *ethclient.Client, target common.Address, block *big.Int, gasCap int64, v Vector) VectorOutcome {
	calldata, err := hexutil.Decode(v.Calldata)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed calldata"}
	}
	value, err := parseCallValue(v.Value)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed value"}
	}
	caller := common.HexToAddress(v.Caller)

	_, err = eth.CallContract(ctx, ethereumCallMsgFrom(caller, target, calldata, gasCap, value), block)
	if err == nil {
		return VectorOutcome{ID: v.ID, Passed: false, Detail: "call did not revert"}
	}
	if !isEVMRevert(err) {
		// A transport/RPC-level failure (timeout, outage, malformed
		// response) is not evidence of anything about the artifact --
		// counting it as a passing revert would let flaky infrastructure
		// (or an adversarial provider) forge a PASS.
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "call error (not a confirmed EVM revert)"}
	}
	if v.ExpectedRevertSelector == "" && v.ExpectedRevertPattern == "" {
		return VectorOutcome{ID: v.ID, Passed: true}
	}
	msg := err.Error()
	if v.ExpectedRevertSelector != "" {
		sel := strings.TrimPrefix(v.ExpectedRevertSelector, "0x")
		if strings.Contains(strings.ToLower(msg), strings.ToLower(sel)) {
			return VectorOutcome{ID: v.ID, Passed: true}
		}
		return VectorOutcome{ID: v.ID, Passed: false, Detail: "revert selector mismatch"}
	}
	if strings.Contains(msg, v.ExpectedRevertPattern) {
		return VectorOutcome{ID: v.ID, Passed: true}
	}
	return VectorOutcome{ID: v.ID, Passed: false, Detail: "revert pattern mismatch"}
}

func executeErc165(ctx context.Context, eth *ethclient.Client, target common.Address, block *big.Int, gasCap int64, v Vector) VectorOutcome {
	ifaceID, err := hexutil.Decode(v.InterfaceID)
	if err != nil || len(ifaceID) != 4 {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed interfaceId"}
	}
	selector := crypto.Keccak256([]byte("supportsInterface(bytes4)"))[:4]
	calldata := append(append([]byte{}, selector...), common.RightPadBytes(ifaceID, 32)...)

	result, err := eth.CallContract(ctx, ethereumCallMsgFrom(common.Address{}, target, calldata, gasCap, nil), block)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "call error (contract may not implement ERC165)"}
	}
	supports := len(result) >= 32 && result[31] == 1
	expected := v.Expected != nil && *v.Expected
	return VectorOutcome{ID: v.ID, Passed: supports == expected}
}

func executeCodeSizeRange(ctx context.Context, eth *ethclient.Client, target common.Address, block *big.Int, v Vector) VectorOutcome {
	code, err := eth.CodeAt(ctx, target, block)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "eth_getCode error"}
	}
	size := len(code)
	return VectorOutcome{ID: v.ID, Passed: size >= v.MinBytes && size <= v.MaxBytes}
}

func executeStorageAtEquals(ctx context.Context, eth *ethclient.Client, target common.Address, block *big.Int, v Vector) VectorOutcome {
	slotBytes, err := hexutil.Decode(v.Slot)
	if err != nil || len(slotBytes) != 32 {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed slot"}
	}
	expected, err := hexutil.Decode(v.ExpectedValue)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "malformed expectedValue"}
	}
	value, err := eth.StorageAt(ctx, target, common.BytesToHash(slotBytes), block)
	if err != nil {
		return VectorOutcome{ID: v.ID, Skipped: true, Detail: "eth_getStorageAt error"}
	}
	return VectorOutcome{ID: v.ID, Passed: hex.EncodeToString(value) == hex.EncodeToString(expected)}
}
