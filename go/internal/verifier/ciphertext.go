package verifier

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"extension-scaffold/internal/config"
)

// CiphertextFetcher fetches ciphertext only from a fixed HTTPS host
// allowlist, with defense against SSRF: every hop (including redirects) is
// scheme/host-checked, and the resolved IP is rejected if it falls in a
// private/loopback/link-local range -- closing the "allowlisted domain
// resolves to an internal IP" (DNS rebinding) gap that a host-string check
// alone would miss. Plan section 11 "Resource limits" /
// THREAT_MODEL.md "SSRF through bundle locator".
//
// Critically, validateURL's resolution is PINNED and reused for the actual
// connection (via dialPinned) rather than left for the transport to
// re-resolve independently at connect time -- resolving twice and
// connecting to whatever the second resolution returns is the exact TOCTOU
// gap DNS rebinding exploits (an attacker's DNS record can point at a safe
// IP for the validation lookup and a private/internal IP moments later for
// the real connection). SNI/Host still reflect the original hostname --
// dialPinned only substitutes the IP address dialed, so TLS certificate
// validation is unaffected.
type CiphertextFetcher struct {
	allowedHosts map[string]bool
	client       *http.Client
	// resolve is net.LookupIP by default; overridable in tests so the
	// private/reserved-IP check can be exercised without depending on real
	// DNS resolution of a fake test hostname.
	resolve func(host string) ([]net.IP, error)

	mu        sync.Mutex
	pinnedIPs map[string]string // lowercased host -> validated IP, set by validateURL, read by dialPinned
}

func NewCiphertextFetcher(allowedHosts []string) *CiphertextFetcher {
	m := make(map[string]bool, len(allowedHosts))
	for _, h := range allowedHosts {
		m[strings.ToLower(h)] = true
	}
	f := &CiphertextFetcher{allowedHosts: m, resolve: net.LookupIP, pinnedIPs: make(map[string]string)}
	f.client = &http.Client{
		Timeout: config.RPCCallTimeout,
		Transport: &http.Transport{
			DialContext: f.dialPinned,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return f.validateURL(req.URL)
		},
	}
	return f
}

func (f *CiphertextFetcher) validateURL(u *url.URL) error {
	if u.Scheme != "https" {
		return fmt.Errorf("ciphertext locator must use https, got %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if !f.allowedHosts[host] {
		return fmt.Errorf("host %q is not in the ciphertext allowlist", host)
	}
	ips, err := f.resolve(host)
	if err != nil {
		return fmt.Errorf("resolving %q: %w", host, err)
	}
	var pinned net.IP
	for _, ip := range ips {
		if isPrivateOrReserved(ip) {
			return fmt.Errorf("host %q resolves to a private/reserved address %s", host, ip)
		}
		if pinned == nil {
			pinned = ip
		}
	}
	if pinned == nil {
		return fmt.Errorf("host %q did not resolve to any address", host)
	}
	f.mu.Lock()
	f.pinnedIPs[host] = pinned.String()
	f.mu.Unlock()
	return nil
}

// dialPinned dials the IP validateURL just validated for this host instead
// of letting the transport perform its own, separate DNS resolution.
// validateURL always runs immediately before the connection that needs it
// (once in Fetch for the initial request, once per hop in CheckRedirect for
// every redirect), so the pinned entry is always fresh.
func (f *CiphertextFetcher) dialPinned(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("parsing dial address %q: %w", addr, err)
	}
	f.mu.Lock()
	pinned, ok := f.pinnedIPs[strings.ToLower(host)]
	f.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no validated IP pinned for host %q -- refusing to dial without validation", host)
	}
	var d net.Dialer
	return d.DialContext(ctx, network, net.JoinHostPort(pinned, port))
}

func isPrivateOrReserved(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// Fetch retrieves ciphertext from locator, enforcing the host allowlist,
// SSRF protections, and a maximum response size.
func (f *CiphertextFetcher) Fetch(ctx context.Context, locator string) ([]byte, error) {
	u, err := url.Parse(locator)
	if err != nil {
		return nil, fmt.Errorf("parsing ciphertext locator: %w", err)
	}
	if err := f.validateURL(u); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, config.RPCCallTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building ciphertext request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching ciphertext: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ciphertext gateway returned status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, int64(config.MaxCiphertextBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading ciphertext body: %w", err)
	}
	if len(data) > config.MaxCiphertextBytes {
		return nil, fmt.Errorf("ciphertext exceeds max size of %d bytes", config.MaxCiphertextBytes)
	}
	return data, nil
}
