package verifier

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"
)

// Regression tests for a real bug: validateURL resolved and checked DNS,
// but the actual http.Client connection used to resolve the hostname
// AGAIN, independently, at connect time -- the classic DNS-rebinding TOCTOU
// gap (an attacker's DNS record can answer "safe" for the validation lookup
// and "private/internal" moments later for the real connection). The fix
// pins the IP validateURL just approved and makes the transport's
// DialContext use exactly that IP, never re-resolving.

func TestDialPinnedUsesThePinnedIPNotFreshDNSResolution(t *testing.T) {
	// A real local listener stands in for "the IP validateURL already
	// approved and pinned". dialPinned must connect to exactly this pinned
	// IP without performing any DNS lookup of its own.
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			conn.Close()
			close(accepted)
		}
	}()

	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	f.resolve = func(string) ([]net.IP, error) {
		t.Error("dialPinned must not call resolve -- it must use the already-pinned IP, not re-resolve")
		return nil, nil
	}
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("splitting listener address: %v", err)
	}
	f.pinnedIPs["gateway.example.com"] = host

	conn, err := f.dialPinned(context.Background(), "tcp", net.JoinHostPort("gateway.example.com", port))
	if err != nil {
		t.Fatalf("dialPinned: %v", err)
	}
	conn.Close()

	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("listener never accepted a connection -- dialPinned did not dial the pinned IP")
	}
}

func TestDialPinnedRejectsHostWithNoPinnedIP(t *testing.T) {
	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	// No prior validateURL call for this host, so nothing is pinned.
	if _, err := f.dialPinned(context.Background(), "tcp", "gateway.example.com:443"); err == nil {
		t.Fatal("expected an error dialing a host with no pinned IP -- refusing to dial without validation is the whole point of the fix")
	}
}

func TestValidateURLPinsTheValidatedIP(t *testing.T) {
	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	f.resolve = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	u, err := url.Parse("https://gateway.example.com/x")
	if err != nil {
		t.Fatalf("parsing test URL: %v", err)
	}
	if err := f.validateURL(u); err != nil {
		t.Fatalf("validateURL: %v", err)
	}

	f.mu.Lock()
	pinned, ok := f.pinnedIPs["gateway.example.com"]
	f.mu.Unlock()
	if !ok {
		t.Fatal("validateURL did not pin an IP for the validated host")
	}
	if pinned != "93.184.216.34" {
		t.Fatalf("pinned IP = %q, want %q", pinned, "93.184.216.34")
	}
}
