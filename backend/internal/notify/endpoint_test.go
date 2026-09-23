package notify

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
)

// hostileEndpoints is the reviewer's table plus every range forbiddenAddr
// names. Each must be refused by ValidateEndpoint (URL layer) — the IP-literal
// ones are also refused by guardDial in TestGuardDialRefusesPrivateAddresses.
var hostileEndpoints = map[string]string{
	"http scheme":            "http://fcm.googleapis.com/fcm/send/abc",
	"no scheme":              "fcm.googleapis.com/fcm/send/abc",
	"file scheme":            "file:///etc/passwd",
	"userinfo":               "https://user:pw@fcm.googleapis.com/fcm/send/abc",
	"no host":                "https:///fcm/send/abc",
	"not a url":              "not a url at all",
	"spec placeholder":       "push_subscription_endpoint_string",
	"ipv4 loopback":          "https://127.0.0.1/x",
	"ipv4 loopback high":     "https://127.255.255.254/x",
	"ipv6 loopback":          "https://[::1]/x",
	"link-local metadata":    "https://169.254.169.254/latest/meta-data/",
	"ipv6 link-local":        "https://[fe80::1]/x",
	"rfc1918 10/8":           "https://10.0.0.5/x",
	"rfc1918 172.16/12":      "https://172.16.0.1/x",
	"rfc1918 172.31":         "https://172.31.255.254/x",
	"rfc1918 192.168/16":     "https://192.168.1.1/x",
	"cgnat 100.64/10":        "https://100.64.0.1/x",
	"cgnat 100.127":          "https://100.127.255.254/x",
	"ipv6 ula fc00::/7":      "https://[fd00::1]/x",
	"ipv6 ula fc":            "https://[fc00::1]/x",
	"unspecified v4":         "https://0.0.0.0/x",
	"unspecified v6":         "https://[::]/x",
	"this-network 0/8":       "https://0.1.2.3/x",
	"ietf protocol 192.0.0":  "https://192.0.0.1/x",
	"benchmark 198.18/15":    "https://198.19.0.1/x",
	"reserved 240/4":         "https://240.0.0.1/x",
	"broadcast":              "https://255.255.255.255/x",
	"multicast v4":           "https://224.0.0.1/x",
	"multicast v6":           "https://[ff02::1]/x",
	"ipv4-mapped private":    "https://[::ffff:10.0.0.5]/x",
	"ipv4-mapped loopback":   "https://[::ffff:127.0.0.1]/x",
	"ipv4-mapped link-local": "https://[::ffff:169.254.169.254]/x",
	"too long":               "https://fcm.googleapis.com/fcm/send/" + strings.Repeat("a", MaxEndpointLength),
}

// realEndpoints are the shapes the four browser push services actually issue
// (tokens shortened). All must pass the URL layer.
var realEndpoints = []string{
	"https://fcm.googleapis.com/fcm/send/dA1b2C3d4E5:APA91bHqZ-example",
	"https://updates.push.services.mozilla.com/wpush/v2/gAAAAABk-example",
	"https://web.push.apple.com/QGtuZXhhbXBsZQ-example",
	"https://wns2-par02p.notify.windows.com/w/?token=AwYAAAB-example",
	"https://push.example/ep1",                  // the service_test fixture
	"https://ntfy.example.org/up/abc123",        // self-hosted UnifiedPush distributor
	"https://fcm.googleapis.com:443/fcm/send/x", // explicit port
}

func TestValidateEndpointRefusesHostileURLs(t *testing.T) {
	for name, raw := range hostileEndpoints {
		err := ValidateEndpoint(raw)
		if !errors.Is(err, ErrForbiddenEndpoint) {
			t.Errorf("%s (%q): err = %v, want ErrForbiddenEndpoint", name, raw, err)
		}
	}
}

func TestValidateEndpointAcceptsRealPushServiceURLs(t *testing.T) {
	for _, raw := range realEndpoints {
		if err := ValidateEndpoint(raw); err != nil {
			t.Errorf("%q: unexpected %v", raw, err)
		}
	}
	// A DNS name is a dial-time decision, not a subscribe-time one (Design §3).
	if err := ValidateEndpoint("https://localhost/x"); err != nil {
		t.Errorf("localhost is decided at the dial, not here: %v", err)
	}
}

func TestForbiddenAddrBoundaries(t *testing.T) {
	forbidden := []string{
		"127.0.0.1", "::1", "169.254.169.254", "169.254.0.1", "fe80::1",
		"10.0.0.0", "10.255.255.255", "172.16.0.0", "172.31.255.255", "192.168.0.0", "192.168.255.255",
		"100.64.0.0", "100.127.255.255", "fc00::", "fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"0.0.0.0", "::", "0.255.255.255", "192.0.0.8", "198.18.0.1", "198.19.255.255", "240.0.0.1", "255.255.255.255",
		"224.0.0.1", "ff02::1", "::ffff:10.0.0.5", "::ffff:127.0.0.1",
	}
	permitted := []string{
		"8.8.8.8", "1.1.1.1", "142.250.31.188", "172.15.255.255", "172.32.0.0", "100.63.255.255", "100.128.0.0",
		"198.17.255.255", "198.20.0.0", "9.255.255.255", "11.0.0.0", "2001:4860:4860::8888", "2a00:1450:4001:80b::200a",
		"::ffff:8.8.8.8",
	}
	for _, s := range forbidden {
		if !forbiddenAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be forbidden", s)
		}
	}
	for _, s := range permitted {
		if forbiddenAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be permitted", s)
		}
	}
}
