package extension

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"extension-scaffold/internal/config"
	"extension-scaffold/internal/verifier"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	"github.com/flare-foundation/go-flare-common/pkg/tee/structs"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"

	"github.com/flare-foundation/tee-node/pkg/processorutils"
)

type Extension struct {
	mu     sync.RWMutex
	Server *http.Server

	greetingCount int
	lastGreeting  string
	farewellCount int
	lastFarewell  string

	// workProofVerifier is nil when WORKPROOF_* env vars are not configured
	// (e.g. plain local dev of the greeting handlers only); processVerify
	// reports a clean status=0 in that case instead of panicking.
	workProofVerifier *verifier.Verifier
}

// --- DO NOT MODIFY: New(), actionHandler() are boilerplate.
func New(extensionPort, signPort int) *Extension {
	e := &Extension{}

	if config.WorkProofRPCURL != "" && config.WorkProofEscrowAddress != "" {
		escrowAddress, err := parseRequiredAddress("WORKPROOF_ESCROW_ADDRESS", config.WorkProofEscrowAddress)
		if err != nil {
			logger.Errorf("workproof verifier not initialized: %v", err)
			return e
		}
		randomNumberV2Addr, err := parseRequiredAddress("WORKPROOF_RANDOM_NUMBER_V2_ADDRESS", config.WorkProofRandomNumberV2Addr)
		if err != nil {
			logger.Errorf("workproof verifier not initialized: %v", err)
			return e
		}
		v, err := verifier.New(verifier.Config{
			ChainID:            114, // Coston2; fixed project-wide (SPEC.md targetChainId), not separately configurable
			RPCURL:             config.WorkProofRPCURL,
			EscrowAddress:      escrowAddress,
			RandomNumberV2Addr: randomNumberV2Addr,
			SignPort:           signPort,
			CiphertextHosts:    config.WorkProofCiphertextHosts,
		})
		if err != nil {
			logger.Errorf("workproof verifier not initialized: %v", err)
		} else {
			e.workProofVerifier = v
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /state", e.stateHandler)
	mux.HandleFunc("POST /action", e.actionHandler)

	e.Server = &http.Server{
		Addr:              fmt.Sprintf(":%d", extensionPort),
		Handler:           mux,
		ReadHeaderTimeout: config.ServerReadHeaderTimeout,
		ReadTimeout:       config.ServerReadTimeout,
		WriteTimeout:      config.ServerWriteTimeout,
		IdleTimeout:       config.ServerIdleTimeout,
	}
	return e
}

func parseRequiredAddress(name, raw string) (common.Address, error) {
	if !common.IsHexAddress(raw) {
		return common.Address{}, fmt.Errorf("%s must be a 20-byte hex address", name)
	}
	addr := common.HexToAddress(raw)
	if addr == (common.Address{}) {
		return common.Address{}, fmt.Errorf("%s must not be the zero address", name)
	}
	return addr, nil
}

// stateHandler() structure is boilerplate but update the State field mapping to match your Extension fields.
func (e *Extension) stateHandler(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	stateResponse := types.StateResponse{
		StateVersion: teeutils.ToHash(config.Version),
		State: types.State{
			GreetingCount:               e.greetingCount,
			LastGreeting:                e.lastGreeting,
			FarewellCount:               e.farewellCount,
			LastFarewell:                e.lastFarewell,
			WorkProofVerifierConfigured: e.workProofVerifier != nil,
		},
	}
	e.mu.RUnlock()

	err := json.NewEncoder(w).Encode(stateResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("sending response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (e *Extension) processAction(action teetypes.Action) (int, []byte) {
	dataFixed, err := processorutils.Parse[instruction.DataFixed](action.Data.Message)
	if err != nil {
		return http.StatusBadRequest, []byte(fmt.Sprintf("decoding fixed data: %v", err))
	}

	switch {
	case dataFixed.OPType == teeutils.ToHash(config.OPTypeGreeting):
		return e.processGreeting(action, dataFixed)

	case dataFixed.OPType == teeutils.ToHash(config.OPTypeWorkProof):
		return e.processWorkProof(action, dataFixed)

	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op type: received %s, expected %s (%s) or %s (%s)",
			dataFixed.OPType.Hex(),
			teeutils.ToHash(config.OPTypeGreeting).Hex(), config.OPTypeGreeting,
			teeutils.ToHash(config.OPTypeWorkProof).Hex(), config.OPTypeWorkProof,
		))
	}
}

// processGreeting routes GREETING instructions by OPCommand.
func (e *Extension) processGreeting(action teetypes.Action, df *instruction.DataFixed) (int, []byte) {
	switch {
	case df.OPCommand == teeutils.ToHash(config.OPCommandSayHello):
		ar := e.processSayHello(action, df)
		b, _ := json.Marshal(ar)
		return http.StatusOK, b

	case df.OPCommand == teeutils.ToHash(config.OPCommandSayGoodbye):
		ar := e.processSayGoodbye(action, df)
		b, _ := json.Marshal(ar)
		return http.StatusOK, b

	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op command: received %s, expected one of [%s (%s), %s (%s)]",
			df.OPCommand.Hex(),
			teeutils.ToHash(config.OPCommandSayHello).Hex(), config.OPCommandSayHello,
			teeutils.ToHash(config.OPCommandSayGoodbye).Hex(), config.OPCommandSayGoodbye,
		))
	}
}

// processSayHello handles SAY_HELLO instructions: returns a greeting and tracks count.
func (e *Extension) processSayHello(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
	var req types.SayHelloRequest
	dec := json.NewDecoder(bytes.NewReader(df.OriginalMessage))
	dec.DisallowUnknownFields()
	err := dec.Decode(&req)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	if req.Name == "" {
		return buildResult(action, df, nil, 0, fmt.Errorf("name must not be empty"))
	}

	e.mu.Lock()
	e.greetingCount++
	greetingNumber := e.greetingCount
	greeting := fmt.Sprintf("Hello, %s! Welcome to Flare Confidential Compute.", req.Name)
	e.lastGreeting = greeting
	e.mu.Unlock()

	resp := types.SayHelloResponse{
		Greeting:       greeting,
		GreetingNumber: greetingNumber,
	}
	data, _ := json.Marshal(resp)

	return buildResult(action, df, data, 1, nil)
}

// processSayGoodbye handles SAY_GOODBYE instructions: returns a farewell and tracks count.
func (e *Extension) processSayGoodbye(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
	var req types.SayGoodbyeRequest
	err := structs.DecodeTo(types.SayGoodbyeMessageArg, df.OriginalMessage, &req)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	if req.Name == "" {
		return buildResult(action, df, nil, 0, fmt.Errorf("name must not be empty"))
	}

	e.mu.Lock()
	e.farewellCount++
	farewellNumber := e.farewellCount
	farewell := fmt.Sprintf("Goodbye, %s! Reason: %s", req.Name, req.Reason)
	e.lastFarewell = farewell
	e.mu.Unlock()

	resp := types.SayGoodbyeResponse{
		Farewell:       farewell,
		FarewellNumber: farewellNumber,
	}
	data, _ := json.Marshal(resp)

	return buildResult(action, df, data, 1, nil)
}

// processWorkProof routes WORKPROOF instructions by OPCommand. P0 supports
// only VERIFY.
func (e *Extension) processWorkProof(action teetypes.Action, df *instruction.DataFixed) (int, []byte) {
	switch {
	case df.OPCommand == teeutils.ToHash(config.OPCommandVerify):
		ar := e.processVerify(action, df)
		b, _ := json.Marshal(ar)
		return http.StatusOK, b

	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op command: received %s, expected %s (%s)",
			df.OPCommand.Hex(), teeutils.ToHash(config.OPCommandVerify).Hex(), config.OPCommandVerify,
		))
	}
}

// processVerify handles VERIFY instructions: decodes the WorkProofInstruction
// (plan Phase 4 task 2), runs the full verify flow, and returns a
// status=1 result with an ABI-encoded VerdictV1 in data for any completed
// verification -- including business-level FAIL/INCONCLUSIVE outcomes
// (task 11) -- or status=0 only when a trustworthy verdict could not be
// formed at all (task 12; see verifier.HandlerFailure). actionHandler is
// DO-NOT-MODIFY boilerplate that does not thread the HTTP request's context
// down through processAction, so this uses a background context bounded by
// Verify's own internal AttemptTotalTimeout instead.
func (e *Extension) processVerify(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
	if e.workProofVerifier == nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("workproof verifier not configured on this node"))
	}

	var instr types.WorkProofInstruction
	if err := structs.DecodeTo(types.WorkProofInstructionArg, df.OriginalMessage, &instr); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding WorkProofInstruction: %w", err))
	}

	verdict, err := e.workProofVerifier.Verify(context.Background(), [32]byte(action.Data.ID), instr)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("verify: %w", err))
	}

	data, err := structs.Encode(types.VerdictV1Arg, verdict)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("encoding VerdictV1: %w", err))
	}

	return buildResult(action, df, data, 1, nil)
}
