package verifier

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// This is a direct regression test for a real bug: executeEthCallReverts
// used to treat ANY non-nil error from eth.CallContract as a passing
// revert, indiscriminately conflating a genuine EVM revert with a
// transport/RPC-level failure (timeout, outage, malformed response). A
// private bundle containing only ETH_CALL_REVERTS vectors against a target
// with a flaky/adversarial RPC path could force a false PASS. These tests
// run a fake JSON-RPC server so both cases are exercised against the real
// go-ethereum client/rpc parsing path, not simulated in Go directly.

type jsonrpcRequest struct {
	Method string `json:"method"`
}

func newFakeRPCServer(t *testing.T, handleEthCall func(w http.ResponseWriter)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonrpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("server: decoding request: %v", err)
		}
		switch req.Method {
		case "eth_call":
			handleEthCall(w)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0"}`))
		}
	}))
}

func dialFakeServer(t *testing.T, srv *httptest.Server) *ethclient.Client {
	t.Helper()
	eth, err := ethclient.DialContext(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("dialing fake RPC server: %v", err)
	}
	t.Cleanup(eth.Close)
	return eth
}

func TestExecuteEthCallRevertsAcceptsARealJSONRPCRevert(t *testing.T) {
	srv := newFakeRPCServer(t, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		// A real go-ethereum-compatible JSON-RPC error response for a
		// reverted eth_call -- this is what a genuine EVM revert looks like
		// on the wire, distinct from an HTTP-level failure.
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":3,"message":"execution reverted: nope","data":"0x08c379a0"}}`))
	})
	defer srv.Close()
	eth := dialFakeServer(t, srv)

	v := Vector{ID: "revert-1", Type: VectorEthCallReverts, Calldata: "0xdeadbeef", Caller: "0x0000000000000000000000000000000000000001"}
	outcome := executeEthCallReverts(context.Background(), eth, common.HexToAddress("0x2"), big.NewInt(1), 100000, v)

	if outcome.Skipped {
		t.Fatalf("a genuine JSON-RPC revert must not be Skipped: %+v", outcome)
	}
	if !outcome.Passed {
		t.Fatalf("a genuine JSON-RPC revert with no selector/pattern constraint must Pass: %+v", outcome)
	}
}

func TestExecuteEthCallRevertsSkipsATransportFailureRatherThanPassing(t *testing.T) {
	srv := newFakeRPCServer(t, func(w http.ResponseWriter) {
		// A transport/infrastructure-level failure: an upstream gateway
		// error page, not a JSON-RPC error response. Real RPC outages and
		// adversarial providers look like this, not like a clean revert.
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>502 Bad Gateway</html>"))
	})
	defer srv.Close()
	eth := dialFakeServer(t, srv)

	v := Vector{ID: "revert-2", Type: VectorEthCallReverts, Calldata: "0xdeadbeef", Caller: "0x0000000000000000000000000000000000000001"}
	outcome := executeEthCallReverts(context.Background(), eth, common.HexToAddress("0x2"), big.NewInt(1), 100000, v)

	if outcome.Passed {
		t.Fatalf("a transport failure must never be counted as a passing revert -- this is exactly the bug being regression-tested: %+v", outcome)
	}
	if !outcome.Skipped {
		t.Fatalf("a transport failure must be Skipped (inconclusive), not Passed or a hard Fail: %+v", outcome)
	}
}

func TestExecuteEthCallRevertsRejectsWhenCallSucceeds(t *testing.T) {
	srv := newFakeRPCServer(t, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000001"}`))
	})
	defer srv.Close()
	eth := dialFakeServer(t, srv)

	v := Vector{ID: "revert-3", Type: VectorEthCallReverts, Calldata: "0xdeadbeef", Caller: "0x0000000000000000000000000000000000000001"}
	outcome := executeEthCallReverts(context.Background(), eth, common.HexToAddress("0x2"), big.NewInt(1), 100000, v)

	if outcome.Passed || outcome.Skipped {
		t.Fatalf("a call that succeeds (does not revert) must Fail, not Pass or Skip: %+v", outcome)
	}
}

func TestParseCallValue(t *testing.T) {
	v, err := parseCallValue("")
	if err != nil || v != nil {
		t.Fatalf("empty value must decode to nil, nil: got %v, %v", v, err)
	}

	v, err = parseCallValue("0x2386f26fc10000") // 0.01 ether in wei
	if err != nil {
		t.Fatalf("parseCallValue: %v", err)
	}
	want := big.NewInt(10000000000000000)
	if v.Cmp(want) != 0 {
		t.Fatalf("parseCallValue = %s, want %s", v, want)
	}

	if _, err := parseCallValue("not-hex"); err == nil {
		t.Fatal("expected an error decoding a malformed value")
	}
}

func TestEthereumCallMsgFromSetsValue(t *testing.T) {
	from := common.HexToAddress("0x1")
	to := common.HexToAddress("0x2")

	msgNoValue := ethereumCallMsgFrom(from, to, []byte{0xaa}, 50000, nil)
	if msgNoValue.Value != nil {
		t.Fatalf("expected nil Value when none is declared, got %v", msgNoValue.Value)
	}

	value := big.NewInt(12345)
	msgWithValue := ethereumCallMsgFrom(from, to, []byte{0xaa}, 50000, value)
	if msgWithValue.Value == nil || msgWithValue.Value.Cmp(value) != 0 {
		t.Fatalf("expected Value=%s to be threaded onto the CallMsg, got %v", value, msgWithValue.Value)
	}
}
