package validator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Dynom/ERI/types"
)

var (
	ErrInvalidHost        = errors.New("invalid host")
	ErrEmailAddressSyntax = errors.New("invalid syntax")
)

func getNewArtifact(ctx context.Context, ep types.EmailParts, options ...ArtifactFn) Artifact {
	a := Artifact{
		Validations: 0,
		Steps:       0,
		Timings:     make(Timings, 10),
		email:       ep,
		mx:          nil,
		ctx:         ctx,
		dialer: &net.Dialer{
			Timeout: time.Second * 60,
		},
		conn: nil,
	}

	for _, opt := range options {
		opt(&a)
	}

	return a
}

func WithDialer(dialer *net.Dialer) ArtifactFn {
	if dialer.Resolver == nil {
		dialer.Resolver = net.DefaultResolver
	}

	return func(artifact *Artifact) {
		artifact.dialer = dialer
		artifact.resolver = dialer.Resolver
	}
}

// getConnection attempts to connect to a host with one of the common email ports.
func getConnection(ctx context.Context, dialer DialContext, mxHost string) (net.Conn, error) {
	var conn net.Conn
	var err error

	ports := []string{"25", "587", "2525", "465"}
	for _, port := range ports {

		// @todo Should we check multiple ports, and do this in parallel?
		// @todo Do we want to force ipv4/6?

		var dialErr error
		conn, dialErr = dialer.DialContext(ctx, "tcp", mxHost+":"+port)

		if dialErr == nil {
			break
		}

		if !strings.Contains(dialErr.Error(), "connection refused") {
			err = fmt.Errorf("%s "+mxHost+":"+port+" %w", err, dialErr)
		}
	}

	// When dialing fails, we could end up with a nil connection, which indicates that all options have been exhausted
	// and that we've ended up with no reachable hosts
	if conn == nil {
		return conn, fmt.Errorf("no connection possible %w", ErrInvalidHost)
	}

	return conn, err
}

// fetchMXHosts collects up to N MX hosts for a given domain
func fetchMXHosts(ctx context.Context, resolver LookupMX, domain string) ([]string, error) {
	mxs, err := resolver.LookupMX(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("MX lookup failed %w", err)
	}

	if len(mxs) == 0 {
		return nil, fmt.Errorf("no MX records found %w", err)
	}

	// Reading an external source, limiting to a liberal amount
	allocateMax := 3
	if l := len(mxs); l < allocateMax {
		allocateMax = l
	}

	collected := make([]string, 0, allocateMax)
	for _, mx := range mxs[:allocateMax] {
		if MightBeAHostOrIP(mx.Host) {
			collected = append(collected, mx.Host)
		}
	}

	if len(collected) == 0 {
		err = fmt.Errorf("tried %d out of %d MX host(s), all were invalid %w", len(mxs), allocateMax, ErrInvalidHost)
	}

	return collected, err
}

// MightBeAHostOrIP is a very rudimentary check to see if the argument could be either a host name or IP address
// It aims at speed and not correctness. It's intended to weed-out bogus responses such as '.'
//
//nolint:gocyclo
func MightBeAHostOrIP(h string) bool {
	// @todo merge with looksLikeValidDomain() -- Do we want regex fallback?

	// Normally we can assume that host names have a tld or consists at least out of 4 characters
	lastCharIndex := len(h) - 1
	if 3 >= lastCharIndex || lastCharIndex >= 253 {
		return false
	}

	var dotCount uint8
	for i, c := range h {
		switch {
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 46 && 0 < i /* dot . */ :
			dotCount++

		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case c == 45 /* dash - */ :
		default:
			return false
		}
	}

	// We need at least one dot for a domain to be valid
	return dotCount > 0
}

// Note: These explicitly exclude several, otherwise legal, characters (such as: 0x00A0).
//
//	NBSP is a frequently occurring erroneous character in e-mail addresses (possibly introduced by a copy & paste
//	from rich formatted documents) and not expected to be desired.
var (
	reLocal  = regexp.MustCompile(`(?i)\A(?:(?:[\p{L}\p{N}]|[!#$%&'*+\-/=?^_\x60{|}~]|[\x{00C0}-\x{1FFF}\x{2070}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}])+(?:\.(?:[\p{L}\p{N}]|[!#$%&'*+\-/=?^_\x60{|}~]|[\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}])+)*)\z`)
	reDomain = regexp.MustCompile(`(?i)\A(?:[\p{L}\p{N}\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}](?:[\p{L}\p{N}\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}-]*[\p{L}\p{N}\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}])?\.)+[\p{L}\p{N}\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}](?:[\p{L}\p{N}\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}-]*[\p{L}\p{N}\x{00A1}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFEF}])?\z`)
)

//nolint:gocyclo
func looksLikeValidLocalPart(local string) bool {
	lastIndexPos := len(local)
	if 1 > lastIndexPos || lastIndexPos > 63 {
		return false
	}

	var tryRegex bool
	for i, c := range local {
		switch {
		default:
			if c > utf8.RuneSelf {
				tryRegex = true
				break
			}

			return false

		case 97 <= c && c <= 122 /* a-z */ :
		case c == 46 && 0 < i && i < lastIndexPos /* . not first or last */ :
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :

		case c == 33 /* ! */ :
		case c == 35 /* # */ :
		case c == 36 /* $ */ :
		case c == 37 /* % */ :
		case c == 38 /* & */ :
		case c == 39 /* ' */ :
		case c == 42 /* * */ :
		case c == 43 /* + */ :
		case c == 45 /* - */ :
		case c == 47 /* / */ :
		case c == 61 /* = */ :
		case c == 63 /* ? */ :
		case c == 94 /* ^ */ :
		case c == 95 /* _ */ :
		case c == 96 /* ` */ :
		case c == 123 /* { */ :
		case c == 124 /* | */ :
		case c == 125 /* } */ :
		case c == 126 /* ~ */ :
		}
	}

	if tryRegex {
		return reLocal.MatchString(local)
	}

	return true
}

//nolint:gocyclo
func looksLikeValidDomain(domain string) bool {
	lastIndexPos := len(domain) - 1

	// Normally we can assume that host names have a tld and/or consists at least out of 4 characters
	if 3 >= lastIndexPos || lastIndexPos >= 253 {
		return false
	}

	var tryRegex bool
	var hasDot bool
	for i, c := range domain {
		switch {
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 46 && 0 < i && i < lastIndexPos /* dot . */ :
			hasDot = true
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case c == 45 && 0 < i && i < lastIndexPos /* dash - */ :

		default:
			if c > utf8.RuneSelf {
				tryRegex = true
				break
			}

			return false
		}
	}

	// We (might) have unicode characters, falling back on full-pattern-matching
	if tryRegex {
		return reDomain.MatchString(domain)
	}

	return hasDot
}

// wrapError wraps an error with the parent error and ignores the parent when it's nil
func wrapError(parent, new error) error {
	if parent == nil {
		return new
	}

	return fmt.Errorf("%s %w", parent, new)
}
