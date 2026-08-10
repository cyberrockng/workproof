package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Regression tests for a real gap: executeErc165 used to accept any
// response with result[31]==1 as "supports=true", regardless of what the
// other 31 bytes contained or how long the response actually was -- a
// non-canonically-encoded bool (garbage in the padding, or extra/missing
// trailing bytes) would be silently treated as a valid true/false rather
// than rejected as malformed. Fixed via go-ethereum's own strict bool ABI
// decoder (accounts/abi.readBool semantics).

func newFakeErc165Server(t *testing.T, resultHex string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonrpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("server: decoding request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_call" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"%s"}`, resultHex)
			return
		}
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0"}`))
	}))
}

func runErc165(t *testing.T, resultHex string, expected bool) VectorOutcome {
	t.Helper()
	srv := newFakeErc165Server(t, resultHex)
	defer srv.Close()
	eth := dialFakeServer(t, srv)

	v := Vector{ID: "erc165-canon", Type: VectorErc165SupportsInterface, InterfaceID: "0x01ffc9a7", Expected: &expected}
	return executeErc165(context.Background(), eth, common.HexToAddress("0x2"), big.NewInt(1), 250000, v)
}

func TestExecuteErc165AcceptsCanonicalTrue(t *testing.T) {
	o := runErc165(t, "0x0000000000000000000000000000000000000000000000000000000000000001", true)
	if o.Skipped || !o.Passed {
		t.Fatalf("canonical true (0x00...01) must pass: %+v", o)
	}
}

func TestExecuteErc165AcceptsCanonicalFalse(t *testing.T) {
	o := runErc165(t, "0x0000000000000000000000000000000000000000000000000000000000000000", false)
	if o.Skipped || !o.Passed {
		t.Fatalf("canonical false (0x00...00) must pass: %+v", o)
	}
}

func TestExecuteErc165RejectsGarbagePaddedTruthyValue(t *testing.T) {
	// Non-zero byte in the padding (not just the last byte) -- this is
	// exactly the bug: the old result[31]==1 check would have accepted
	// this as supports=true since the last byte happens to be 0x01, even
	// though a real Solidity bool return is never encoded this way.
	o := runErc165(t, "0x0100000000000000000000000000000000000000000000000000000000000001", true)
	if !o.Skipped {
		t.Fatalf("a non-canonical bool encoding must be Skipped (inconclusive), not silently accepted as true: %+v", o)
	}
}

func TestExecuteErc165RejectsNonBoolLastByte(t *testing.T) {
	// Last byte is 0xff, not 0 or 1 -- readBool-equivalent strictness must
	// reject this outright rather than treating any non-zero last byte as
	// truthy.
	o := runErc165(t, "0x00000000000000000000000000000000000000000000000000000000000000ff", true)
	if !o.Skipped {
		t.Fatalf("a last byte outside {0,1} must be rejected as non-canonical: %+v", o)
	}
}

func TestExecuteErc165RejectsWrongLength(t *testing.T) {
	o := runErc165(t, "0x01", true)
	if !o.Skipped {
		t.Fatalf("a response shorter than one ABI word must be rejected, not lenient-accepted: %+v", o)
	}
}
