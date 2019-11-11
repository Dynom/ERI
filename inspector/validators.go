package inspector

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/Dynom/ERI/types"
)

const (
	DefaultRecipient = "eri@tysug.net"
)

var (
	ErrInvalidHost = errors.New("invalid host")
)

// Validator is the type all validators must conform to
type Validator func(ctx context.Context, e types.EmailParts) Result

// Result is the validation result
type Result struct {
	Error error
	types.Timings
	types.Validations
}

// ValidateMXAndRCPT validates if the mailbox exists. You can control the timeout by using context
func ValidateMXAndRCPT(recipient string) Validator {
	resolver := net.Resolver{}
	dialer := net.Dialer{}

	return func(ctx context.Context, e types.EmailParts) Result {
		dialer := dialer
		if deadline, set := ctx.Deadline(); set {
			dialer.Deadline = deadline
		}

		result := Result{
			Timings: make(types.Timings, 0, 4),
		}

		// Assuming the worst
		result.Validations.MarkAsInvalid()

		var mxHost string
		result = validateStep(result, "LookupMX", types.VFMXLookup, func() error {
			var err error
			mxHost, err = fetchMXHost(ctx, &resolver, e.Domain)

			return err
		})

		if result.Error != nil {
			return result
		}

		var conn net.Conn
		result = validateStep(result, "Dial", types.VFHostConnect, func() error {
			var err error
			conn, err = getConnection(ctx, dialer, mxHost)
			return err
		})

		if result.Error != nil || conn == nil {
			return result
		}

		defer func() {
			_ = conn.Close()
		}()

		client, err := smtp.NewClient(conn, e.Domain)

		if err != nil {
			// @todo should this reflect badly on the verification process?
			result.Error = fmt.Errorf("unable to create an SMTP client %w", err)
			return result
		}

		defer func() {
			_ = client.Quit()
		}()

		result = validateStep(result, "Mail", types.VFValidRCPT, func() error {
			return client.Mail(recipient)
		})

		if result.Error != nil {
			return result
		}

		result = validateStep(result, "Rcpt", types.VFValidRCPT|types.VFValid, func() error {
			return client.Rcpt(e.Address)
		})

		return result
	}
}

// ValidateMX validates if the domain of the address has MX records. This is a more basic check than ValidateMXAndRCPT,
// using them both together doesn't make a lot of sense. Use this for less precision and if speed is more important
// you could use this for an initial check instead
func ValidateMX() Validator {

	resolver := net.Resolver{}

	return func(ctx context.Context, e types.EmailParts) Result {
		var start time.Time

		result := Result{
			Timings: make(types.Timings, 0, 1),
			Error:   nil,
		}

		start = time.Now()
		_, err := fetchMXHost(ctx, &resolver, e.Domain)
		result.Timings.Add("LookupMX", time.Since(start))
		result.Validations |= types.VFMXLookup

		if err != nil {
			result.Validations.MarkAsInvalid()
			result.Error = err
			return result
		}

		result.Validations |= types.VFValid

		return result
	}
}

// ValidateSyntax performs the most basic of checks: Does the syntax seem somewhat plausible.
func ValidateSyntax() Validator {
	return func(ctx context.Context, e types.EmailParts) Result {
		var start time.Time

		result := Result{
			Timings: make(types.Timings, 0, 1),
			Error:   nil,
		}

		start = time.Now()
		_, err := mail.ParseAddress(e.Address)
		result.Timings.Add("Structure", time.Since(start))
		result.Validations = 1 | types.VFSyntax

		if err != nil {
			result.Error = err
			return result
		}

		result.Validations = 1 | types.VFValid
		return result
	}
}

// mightBeAHostOrIP is a very rudimentary check to see if the argument could be either a host name or IP address
// It aims on speed and not for correctness. It's intended to weed-out bogus responses such as '.'
//nolint:gocyclo
func mightBeAHostOrIP(h string) bool {

	// Normally we can assume that host names have a tld or consists at least out of 4 characters
	if l := len(h); 4 >= l || l >= 253 {
		return false
	}

	for _, c := range h {
		switch {
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 45 /* dash - */ :
		case c == 46 /* dot . */ :
		default:
			return false
		}
	}

	return true
}

// fetchMXHost returns 0 or 1 value that resembles a hostname/ip of the MX records, sorted by preference
func fetchMXHost(ctx context.Context, resolver *net.Resolver, domain string) (string, error) {

	mxs, err := resolver.LookupMX(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("MX lookup failed %w", err)
	}

	if len(mxs) == 0 {
		return "", fmt.Errorf("no MX records found %w", err)
	}

	for _, mx := range mxs {
		if mightBeAHostOrIP(mx.Host) {

			// Hosts may end with a ".", whilst still valid, it might produce problems with lookups
			return strings.TrimRight(mx.Host, "."), nil
		}
	}

	return "", fmt.Errorf("tried %d MX host(s), all were invalid %w", len(mxs), ErrInvalidHost)
}

func getConnection(ctx context.Context, dialer net.Dialer, mxHost string) (net.Conn, error) {
	var conn net.Conn
	var err error

	ports := []string{"25", "587", "2525", "465"}
	for _, port := range ports {
		port := port

		// @todo Should we check multiple ports, and do this in parallel?
		// @todo Do we want to force ipv4/6?

		var dialErr error
		conn, dialErr = dialer.DialContext(ctx, "tcp", mxHost+":"+port)
		if dialErr == nil {
			break
		}

		if !strings.Contains(dialErr.Error(), "connection refused") {
			err = fmt.Errorf("%s %w", err, dialErr)
		}
	}

	return conn, err
}

// validateStep encapsulates some boilerplate. If the callback returns with a nil error, the flags are applied
func validateStep(result Result, stepName string, flag types.Validations, fn func() error) Result {
	start := time.Now()
	err := fn()
	result.Timings.Add(stepName, time.Since(start))

	if err != nil {
		result.Error = fmt.Errorf("step %s failed, error: %w", stepName, err)
	} else {
		result.Validations |= flag
	}

	return result
}
