package validator

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// getConnection attempts to connect to a host with one of the common email ports.
func getConnection(ctx context.Context, dialer *net.Dialer, mxHost string) (net.Conn, error) {
	var conn net.Conn
	var err error

	const dialTimeout = 100 * time.Millisecond

	ports := []string{"25", "587", "2525", "465"}
	for _, port := range ports {
		port := port

		// @todo Should we check multiple ports, and do this in parallel?
		// @todo Do we want to force ipv4/6?
		// @todo Configure timeouts specifically for this expensive step?

		var dialErr error

		ctx, cancel := context.WithTimeout(ctx, dialTimeout)
		conn, dialErr = dialer.DialContext(ctx, "tcp", mxHost+":"+port)
		cancel()

		if dialErr == nil {
			break
		}

		if !strings.Contains(dialErr.Error(), "connection refused") {
			err = fmt.Errorf("%s "+mxHost+":"+port+" %w", err, dialErr)
		}
	}

	return conn, err
}

// fetchMXHosts collects up to 10 MX hosts for a given domain
func fetchMXHosts(ctx context.Context, resolver *net.Resolver, domain string) ([]string, error) {

	mxs, err := resolver.LookupMX(ctx, domain)
	if err != nil {
		return []string{}, fmt.Errorf("MX lookup failed %w", err)
	}

	if len(mxs) == 0 {
		return []string{}, fmt.Errorf("no MX records found %w", err)
	}

	// Reading an external source, limiting to a liberal amount
	var allocateMax = 10
	if l := len(mxs); l < 10 {
		allocateMax = l
	}

	var collected = make([]string, 0, allocateMax)
	for _, mx := range mxs[:allocateMax] {

		// Hosts might end on a "." (which isn't bad) or consist solely out of a "." (which is bad) this produces a canonical test basis
		host := strings.TrimRight(mx.Host, ".")
		if MightBeAHostOrIP(host) {
			collected = append(collected, host)
		}
	}

	if len(collected) == 0 {
		err = fmt.Errorf("tried %d MX host(s), all were invalid %w", len(mxs), ErrInvalidHost)
	}

	return collected, err
}

// MightBeAHostOrIP is a very rudimentary check to see if the argument could be either a host name or IP address
// It aims on speed and not for correctness. It's intended to weed-out bogus responses such as '.'
//nolint:gocyclo
func MightBeAHostOrIP(h string) bool {

	// Normally we can assume that host names have a tld or consists at least out of 4 characters
	lastCharIndex := len(h) - 1
	if 4 >= lastCharIndex || lastCharIndex >= 253 {
		return false
	}

	var dotCount uint8
	for i, c := range h {
		switch {
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 45 /* dash - */ :
		case c == 46 && 0 < i && i < lastCharIndex /* dot . */ :
			dotCount++
		default:
			return false
		}
	}

	// We need at least one dot for a domain to be valid
	return dotCount > 0
}
