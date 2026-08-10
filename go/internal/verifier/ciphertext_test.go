package verifier

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"extension-scaffold/internal/config"
)

func TestIsPrivateOrReservedRejectsPrivateRanges(t *testing.T) {
	private := []string{"127.0.0.1", "10.0.0.5", "192.168.1.1", "172.16.0.1", "169.254.1.1", "0.0.0.0"}
	for _, ip := range private {
		if !isPrivateOrReserved(net.ParseIP(ip)) {
			t.Errorf("%s should be classified as private/reserved", ip)
		}
	}
}

func TestIsPrivateOrReservedAllowsPublicRanges(t *testing.T) {
	public := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"}
	for _, ip := range public {
		if isPrivateOrReserved(net.ParseIP(ip)) {
			t.Errorf("%s is a public address and should not be rejected", ip)
		}
	}
}

func TestValidateURLRejectsNonHTTPS(t *testing.T) {
	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	u, _ := url.Parse("http://gateway.example.com/abc")
	if err := f.validateURL(u); err == nil {
		t.Fatal("expected rejection for non-https scheme")
	}
}

func TestValidateURLRejectsNonAllowlistedHost(t *testing.T) {
	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	u, _ := url.Parse("https://evil.example.com/abc")
	if err := f.validateURL(u); err == nil {
		t.Fatal("expected rejection for a host not in the allowlist")
	}
}

func TestValidateURLRejectsHostResolvingToPrivateIP(t *testing.T) {
	// Even an allowlisted hostname must be rejected if it resolves to a
	// private/internal address (DNS rebinding defense) -- injected resolver
	// so this doesn't depend on real DNS.
	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	f.resolve = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	u, _ := url.Parse("https://gateway.example.com/abc")
	if err := f.validateURL(u); err == nil {
		t.Fatal("expected rejection when an allowlisted host resolves to a private IP")
	}
}

func TestValidateURLAcceptsAllowlistedPublicHost(t *testing.T) {
	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	f.resolve = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	u, _ := url.Parse("https://gateway.example.com/abc")
	if err := f.validateURL(u); err != nil {
		t.Fatalf("expected an allowlisted host resolving publicly to be accepted, got: %v", err)
	}
}

// dialingTransportTo builds an http.Transport that trusts srv's TLS
// certificate (via srv.Client()'s pre-configured transport) but redirects
// every actual TCP dial to srv's real listener address, regardless of the
// (fake, allowlisted-for-the-test) hostname in the request URL. This lets
// Fetch's real https-scheme/allowlist/DNS-resolution checks run for real
// while still reaching a local httptest.Server -- only the final socket
// dial is substituted, the same DI seam production code uses via
// CiphertextFetcher.resolve.
func dialingTransportTo(t *testing.T, srv *httptest.Server) *http.Transport {
	t.Helper()
	base, ok := srv.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("httptest server's client transport is not *http.Transport")
	}
	clone := base.Clone()
	clone.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	return clone
}

func TestFetchHappyPathAndSizeCap(t *testing.T) {
	payload := []byte("real-ciphertext-bytes")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	f.resolve = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil // passes the private-IP check
	}
	f.client = &http.Client{Transport: dialingTransportTo(t, srv)}

	got, err := f.Fetch(context.Background(), "https://gateway.example.com/some-ciphertext-hash")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Fetch returned %q, want %q", got, payload)
	}
}

func TestFetchRejectsResponseOverMaxCiphertextBytes(t *testing.T) {
	oversized := make([]byte, config.MaxCiphertextBytes+1)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(oversized)
	}))
	defer srv.Close()

	f := NewCiphertextFetcher([]string{"gateway.example.com"})
	f.resolve = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	f.client = &http.Client{Transport: dialingTransportTo(t, srv)}

	_, err := f.Fetch(context.Background(), "https://gateway.example.com/x")
	if err == nil {
		t.Fatalf("expected rejection for a response exceeding config.MaxCiphertextBytes (%d bytes)", config.MaxCiphertextBytes)
	}
	if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") ||
		strings.Contains(err.Error(), "tls:") || strings.Contains(err.Error(), "x509:") {
		t.Fatalf("rejection happened for the wrong reason (TLS/transport failure, not the size cap): %v", err)
	}
}
