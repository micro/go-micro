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
		if blockedPushIP(ip) {
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
func blockedPushIP(ip net.IP) bool {
	return ip == nil ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// pushDialControl runs after DNS resolution, immediately before connect, on the
// resolved address — so it blocks a host that passed URL validation but was
// rebound to an internal IP (DNS rebinding).
func pushDialControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("push callback: cannot parse dial address %q", address)
	}
	if blockedPushIP(ip) {
		return fmt.Errorf("push callback: refusing to connect to blocked address %s", ip)
	}
	return nil
}

// pushGuardClient is the HTTP client used for default-policy push delivery. Its
// dialer refuses connections to blocked addresses at connect time.
var pushGuardClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
			Control: pushDialControl,
		}).DialContext,
	},
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

// pushClient is the HTTP client deliverPush uses: the guarded client under the
// default policy, or the default client when an operator has taken over the
// policy via Options.AllowPushURL (they own the trust decision then).
func (d *dispatcher) pushClient() *http.Client {
	if d.guardPushDial {
		return pushGuardClient
	}
	return http.DefaultClient
}
