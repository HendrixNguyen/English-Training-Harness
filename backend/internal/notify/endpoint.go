package notify

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
)

// ErrForbiddenEndpoint means a push endpoint is not something this server
// will dial: wrong scheme, malformed, or pointing at a private, loopback,
// link-local or otherwise internal address. UpdateSettings wraps it in
// ErrInvalidRequest (→ 400); Tick treats it like ErrSubscriptionGone (prune).
//
// This is the SSRF boundary: endpoint is client-supplied and the worker POSTs
// to it daily, so it is validated when stored AND enforced on the resolved
// address at dial time (guardDial) — DNS can change between the two.
var ErrForbiddenEndpoint = errors.New("notify: forbidden push endpoint")

// MaxEndpointLength bounds the stored URL. Real push endpoints are 100–300
// bytes; 2048 is the conventional URL ceiling.
const MaxEndpointLength = 2048

// ValidateEndpoint is the syntactic, no-network check: https, parses, has a
// host, no userinfo, bounded, and an IP-literal host is not a forbidden
// address. Hostnames are deliberately NOT resolved here — that decision
// belongs to guardDial, on the address actually being connected to.
func ValidateEndpoint(raw string) error {
	if len(raw) > MaxEndpointLength {
		return fmt.Errorf("%w: longer than %d bytes", ErrForbiddenEndpoint, MaxEndpointLength)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrForbiddenEndpoint, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q, want https", ErrForbiddenEndpoint, u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("%w: userinfo not allowed", ErrForbiddenEndpoint)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%w: no host", ErrForbiddenEndpoint)
	}
	if ip, err := netip.ParseAddr(host); err == nil && forbiddenAddr(ip) {
		return fmt.Errorf("%w: %s is not a public address", ErrForbiddenEndpoint, ip)
	}
	return nil
}

// Explicit IPv4 ranges the netip predicates do not cover.
var forbiddenPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),     // "this" network
	netip.MustParsePrefix("100.64.0.0/10"), // CGNAT (RFC 6598)
	netip.MustParsePrefix("192.0.0.0/24"),  // IETF protocol assignments
	netip.MustParsePrefix("198.18.0.0/15"), // benchmarking (RFC 2544)
	netip.MustParsePrefix("240.0.0.0/4"),   // reserved + broadcast
}

// forbiddenAddr reports whether ip is an address this server must never
// dial for a push endpoint. IPv4-mapped IPv6 is unmapped first so
// ::ffff:10.0.0.5 is judged as 10.0.0.5.
func forbiddenAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, p := range forbiddenPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
