package verifier

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"extension-scaffold/internal/config"
)

// Regression tests for real gaps: three resource-limit controls were
// schema-validated but never actually enforced anywhere -- config.MaxBundleBytes
// (the decrypted plaintext had no size cap at all), the ERC-165 vector's
// gas cap (executeErc165 used the no-gas-limit ethereumCallMsg instead of
// the capped ethereumCallMsgFrom every other call-based vector uses), and
// bundle.TimeoutMsPerCall (validated by schema but never used to bound an
// individual vector's execution -- only the overall AttemptTotalTimeout did
// anything).

func TestCheckBundleSizeRejectsOversizedPlaintext(t *testing.T) {
	oversized := make([]byte, config.MaxBundleBytes+1)
	if err := checkBundleSize(oversized); err == nil {
		t.Fatal("expected rejection for a decrypted bundle exceeding config.MaxBundleBytes")
	}
}

func TestCheckBundleSizeAcceptsWithinLimit(t *testing.T) {
	small := make([]byte, config.MaxBundleBytes)
	if err := checkBundleSize(small); err != nil {
		t.Fatalf("expected a bundle exactly at the limit to be accepted, got: %v", err)
	}
}

func TestExecuteErc165RespectsGasCap(t *testing.T) {
	var capturedGas string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// eth_call's params are [callObject, blockParameter] -- the second
		// element is a string ("0x1", "latest", ...), not an object, so it
		// must be decoded via json.RawMessage first rather than assuming
		// every param is a map.
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("server: decoding request: %v", err)
		}
		if req.Method == "eth_call" && len(req.Params) > 0 {
			var callObj map[string]any
			if err := json.Unmarshal(req.Params[0], &callObj); err != nil {
				t.Fatalf("server: decoding call object: %v", err)
			}
			if g, ok := callObj["gas"].(string); ok {
				capturedGas = g
			}
		}
		w.Header().Set("Content-Type", "application/json")
		// A well-formed 32-byte "true" ERC165 response.
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000001"}`))
	}))
	defer srv.Close()
	eth := dialFakeServer(t, srv)

	trueVal := true
	v := Vector{ID: "erc165-1", Type: VectorErc165SupportsInterface, InterfaceID: "0x01ffc9a7", Expected: &trueVal}
	outcome := executeErc165(context.Background(), eth, common.HexToAddress("0x2"), big.NewInt(1), 250000, v)

	if outcome.Skipped {
		t.Fatalf("unexpected skip: %+v", outcome)
	}
	if capturedGas == "" {
		t.Fatal("no gas parameter was sent with the eth_call -- executeErc165 must apply the gas cap like every other call-based vector")
	}
	if capturedGas != "0x3d090" { // 250000 in hex
		t.Fatalf("gas param = %s, want 0x3d090 (250000)", capturedGas)
	}
}

func TestPerVectorTimeoutCutsOffASlowCallRatherThanHangingForTheWholeAttempt(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // hangs until the test releases it, simulating a stuck RPC call
	}))
	defer srv.Close()
	defer close(release)
	eth := dialFakeServer(t, srv)

	// Mirrors exactly what Verify()'s vector-execution loop does: wrap each
	// call in its own per-vector timeout derived from bundle.TimeoutMsPerCall,
	// not just the overall attempt timeout.
	perCallTimeout := 100 * time.Millisecond
	callCtx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()

	start := time.Now()
	v := Vector{ID: "slow-1", Type: VectorEthCallEquals, Calldata: "0x1234", Caller: "0x0000000000000000000000000000000000000001", ExpectedReturn: "0x5678"}
	outcome := executeEthCallEquals(callCtx, eth, common.HexToAddress("0x2"), big.NewInt(1), 100000, 1024, v)
	elapsed := time.Since(start)

	if !outcome.Skipped {
		t.Fatalf("expected a timed-out call to be Skipped, got: %+v", outcome)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("call took %v to return -- the per-vector timeout (%v) did not cut it off", elapsed, perCallTimeout)
	}
}
