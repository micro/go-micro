package a2a

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// Push-notification callbacks are the one place the A2A gateway makes an
// outbound HTTP request to an address chosen by a (possibly untrusted) caller:
// tasks/pushNotificationConfig/set records a URL and deliverPush POSTs task
// state to it. Without a guard that is a server-side request forgery vector —
// a caller can aim the gateway at loopback, link-local (cloud metadata), or
// private hosts it would otherwise never reach.
//
// The default policy allows only http/https callbacks whose host does not
// resolve to a loopback, private, link-local, or unspecified address, and the
// guarded HTTP client re-checks the *resolved* IP at dial time so a hostname
// that passes validation cannot be rebound to an internal address before the
// connection is made. Operators who need to reach a trusted in-cluster
// receiver set Options.AllowPushURL to take over the policy.

// pushLookupIP resolves a host to IPs; overridable in tests.
var pushLookupIP = net.LookupIP

// defaultPushURLPolicy is the SSRF-safe policy applied when no AllowPushURL is
// configured. It rejects non-http(s) schemes and hosts that resolve to a
// loopback, private, link-local, multicast, or unspecified address.
func defaultPushURLPolicy(u *url.URL) error {
	return defaultPushURLPolicyWithPrefixes(u, nil)
}

func defaultPushURLPolicyWithPrefixes(u *url.URL, nat64Prefixes []net.IPNet) error {
	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("push callback scheme %q not allowed (want http or https)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("push callback url has no host")
	}
	ips, err := resolvePushHost(host)
	if err != nil {
		return fmt.Errorf("push callback host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("push callback host %q did not resolve", host)
	}
	for _, ip := range ips {
		if blockedPushIP(ip, nat64Prefixes...) {
			return fmt.Errorf("push callback host %q resolves to a blocked address %s", host, ip)
		}
	}
	return nil
}

func resolvePushHost(host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	return pushLookupIP(host)
}

// blockedPushIP reports whether ip is one an outbound push callback must not
// reach: loopback, private (RFC1918 / ULA), link-local (incl. 169.254.169.254
// cloud metadata), multicast, or the unspecified address.
func blockedPushIP(ip net.IP, nat64Prefixes ...net.IPNet) bool {
	if ip == nil ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	// IPv6 transition addresses (6to4, NAT64, Teredo, IPv4-compatible) embed an
	// IPv4 address none of the checks above look at. Unwrap and re-check it so
	// [2002:a9fe:a9fe::1] is treated as 169.254.169.254.
	for _, inner := range embeddedIPv4(ip, nat64Prefixes...) {
		if blockedPushIP(inner) {
			return true
		}
	}
	return false
}

// embeddedIPv4 returns the IPv4 addresses carried inside an IPv6 transition
// address, or nil when it carries none. Teredo yields two: the relay server and
// the (obfuscated) client.
func embeddedIPv4(ip net.IP, nat64Prefixes ...net.IPNet) []net.IP {
	v6 := ip.To16()
	if v6 == nil || ip.To4() != nil {
		return nil
	}
	for _, prefix := range nat64Prefixes {
		if inner := rfc6052IPv4(v6, prefix); inner != nil {
			return []net.IP{inner}
		}
	}
	switch {
	// 6to4 — RFC 3056, 2002::/16, IPv4 in bytes 2-6.
	case v6[0] == 0x20 && v6[1] == 0x02:
		return []net.IP{net.IPv4(v6[2], v6[3], v6[4], v6[5])}
	// NAT64 well-known prefix — RFC 6052, 64:ff9b::/96, IPv4 in the low 32 bits.
	case v6[0] == 0x00 && v6[1] == 0x64 && v6[2] == 0xff && v6[3] == 0x9b && allZeros(v6[4:12]):
		return []net.IP{net.IPv4(v6[12], v6[13], v6[14], v6[15])}
	// NAT64 local-use prefix — RFC 8215, 64:ff9b:1::/48. Embedded IPv4 position
	// depends on the operator's prefix length, so block the whole range.
	case v6[0] == 0x00 && v6[1] == 0x64 && v6[2] == 0xff && v6[3] == 0x9b && v6[4] == 0x00 && v6[5] == 0x01:
		return []net.IP{net.IPv4zero}
	// Teredo — RFC 4380, 2001::/32. Server IPv4 in bytes 4-8, client IPv4 in
	// bytes 12-16 obfuscated by XOR with 0xff.
	case v6[0] == 0x20 && v6[1] == 0x01 && v6[2] == 0x00 && v6[3] == 0x00:
		return []net.IP{
			net.IPv4(v6[4], v6[5], v6[6], v6[7]),
			net.IPv4(v6[12]^0xff, v6[13]^0xff, v6[14]^0xff, v6[15]^0xff),
		}
	// IPv4-compatible — deprecated ::a.b.c.d, not unwrapped by net.IP.To4.
	case allZeros(v6[0:12]):
		return []net.IP{net.IPv4(v6[12], v6[13], v6[14], v6[15])}
	}
	return nil
}

// rfc6052IPv4 extracts the IPv4 address from ip when prefix is a supported
// network-specific RFC 6052 prefix. For prefix lengths up to /64, RFC 6052
// places the reserved u octet at bits 64-71 and splits the IPv4 bits around it.
func rfc6052IPv4(ip net.IP, prefix net.IPNet) net.IP {
	bits, total := prefix.Mask.Size()
	prefixIP := prefix.IP.To16()
	if total != 128 || prefixIP == nil || !prefix.Contains(ip) {
		return nil
	}
	var indexes []int
	switch bits {
	case 32:
		indexes = []int{4, 5, 6, 7}
	case 40:
		indexes = []int{5, 6, 7, 9}
	case 48:
		indexes = []int{6, 7, 9, 10}
	case 56:
		indexes = []int{7, 9, 10, 11}
	case 64:
		indexes = []int{9, 10, 11, 12}
	case 96:
		indexes = []int{12, 13, 14, 15}
	default:
		return nil
	}
	if bits <= 64 && ip[8] != 0 {
		return nil
	}
	return net.IPv4(ip[indexes[0]], ip[indexes[1]], ip[indexes[2]], ip[indexes[3]])
}

func allZeros(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

// pushDialControl runs after DNS resolution, immediately before connect, on the
// resolved address — so it blocks a host that passed URL validation but was
// rebound to an internal IP (DNS rebinding).
func pushDialControl(_, address string, _ syscall.RawConn) error {
	return pushDialControlWithPrefixes("", address, nil, nil)
}

func pushDialControlWithPrefixes(_ string, address string, _ syscall.RawConn, nat64Prefixes []net.IPNet) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("push callback: cannot parse dial address %q", address)
	}
	if blockedPushIP(ip, nat64Prefixes...) {
		return fmt.Errorf("push callback: refusing to connect to blocked address %s", ip)
	}
	return nil
}

// newPushGuardClient creates the HTTP client used for default-policy push
// delivery. Its dialer refuses connections to blocked addresses at connect
// time.
func newPushGuardClient(nat64Prefixes []net.IPNet) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
				Control: func(network, address string, conn syscall.RawConn) error {
					return pushDialControlWithPrefixes(network, address, conn, nat64Prefixes)
				},
			}).DialContext,
		},
	}
}

// checkPushURL validates a callback URL against the dispatcher's effective
// policy (Options.AllowPushURL, or the default SSRF-safe policy).
func (d *dispatcher) checkPushURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid push callback url: %w", err)
	}
	policy := d.allowPushURL
	if policy == nil {
		policy = defaultPushURLPolicy
	}
	return policy(u)
}

func (d *dispatcher) setNAT64Prefixes(prefixes []net.IPNet) {
	d.nat64Prefixes = append([]net.IPNet(nil), prefixes...)
	if d.guardPushDial {
		d.allowPushURL = func(u *url.URL) error {
			return defaultPushURLPolicyWithPrefixes(u, d.nat64Prefixes)
		}
		d.pushHTTPClient = newPushGuardClient(d.nat64Prefixes)
	}
}

// pushClient is the HTTP client deliverPush uses: the guarded client under the
// default policy, or the default client when an operator has taken over the
// policy via Options.AllowPushURL (they own the trust decision then).
func (d *dispatcher) pushClient() *http.Client {
	if d.guardPushDial {
		return d.pushHTTPClient
	}
	return http.DefaultClient
}
