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

	"github.com/Dynom/ERI/inspector/types"
)

const (

	// Validation Flags, these flags represent successful validation steps. Depending on how far you want to go, you can
	// classify a validation as valid enough, for your use-case.
	VFValid       Validations = 1 << iota // The e-mail is considered valid (1) or not (0)
	VFSyntax      Validations = 1 << iota // e-mail address follows a (reasonably) valid syntax
	VFMXLookup    Validations = 1 << iota // e-mail domain has MX records
	VFHostConnect Validations = 1 << iota // MX accepts connections
	VFValidRCPT   Validations = 1 << iota // MX acknowledges that the RCPT exists
)

var (
	ErrInvalidHost   = errors.New("invalid host")
	DefaultRecipient = "eri@tysug.net"
)

// Validator is the type all validators must conform to
type Validator func(ctx context.Context, e types.EmailParts) Result

// Validations holds the validation steps performed.
type Validations uint64

// IsValid returns true if the Validations are considered successful
func (v Validations) IsValid() bool {
	return v&VFValid == 1
}

// Merge appends to Validations are returns the result. If the new validations do not consider the validation successful
// it will mark the new Validations as unsuccessful as well.
func (v Validations) Merge(new Validations) Validations {

	v.MarkAsInvalid()
	return v | new
}

// MarkAsInvalid clears the CFValid bit and marks the Validations as invalid
func (v *Validations) MarkAsInvalid() {
	*v &^= VFValid
}

// MarkAsValid sets the CFValid bit and marks the Validations as valid
func (v *Validations) MarkAsValid() {
	*v |= VFValid
}

// Result is the validation result
type Result struct {
	Error error
	Timings
	Validations
}

type Timings []Timing

func (t *Timings) Add(l string, d time.Duration) {
	*t = append(*t, Timing{Label: l, Duration: d})
}

type Timing struct {
	Label    string
	Duration time.Duration
}

// ValidateMXAndRCPT validates if the mailbox exists. You can control the timeout by using context
func ValidateMXAndRCPT(recipient string) Validator {
	var start time.Time

	resolver := net.Resolver{}
	dialer := net.Dialer{}

	return func(ctx context.Context, e types.EmailParts) Result {

		dialer := dialer
		if deadline, set := ctx.Deadline(); set {
			dialer.Deadline = deadline
		}

		result := Result{
			Timings: make(Timings, 0, 4),
			Error:   nil,
		}

		start = time.Now()
		mxHost, err := fetchMXHost(ctx, &resolver, e.Domain)
		result.Timings.Add("LookupMX", time.Since(start))

		if err != nil {
			result.Validations.MarkAsInvalid()

			result.Error = err
			return result
		}

		result.Validations |= VFMXLookup

		ports := []string{"25", "587", "2525", "465"}
		var conn net.Conn
		for _, port := range ports {
			var err error

			// @todo Should we check multiple ports, and do this in parallel?
			// @todo Do we want to force ipv4/6?

			start = time.Now()
			conn, err = dialer.DialContext(ctx, "tcp", mxHost+":"+port)
			result.Timings.Add("Dial "+port, time.Since(start))

			if err != nil && strings.Contains(err.Error(), "connection refused") {
				continue
			}
		}

		result.Validations |= VFHostConnect

		if conn == nil {
			result.Validations.MarkAsInvalid()

			result.Error = fmt.Errorf("dailing MX host %q failed %w", mxHost, err)
			return result
		}

		defer func() {
			_ = conn.Close()
		}()

		if err := ctx.Err(); err != nil {
			result.Error = err
			return result
		}

		client, err := smtp.NewClient(conn, e.Domain)

		if err != nil {
			// @todo should this reflect badly on the verification process?
			result.Error = fmt.Errorf("unable to create an SMTP client %w", err)
			return result
		}

		defer func() {
			_ = client.Quit()
		}()

		if err := ctx.Err(); err != nil {
			result.Validations.MarkAsInvalid()

			result.Error = err
			return result
		}

		start = time.Now()
		err = client.Mail(recipient)
		result.Timings.Add("Mail", time.Since(start))

		if err != nil {
			result.Validations.MarkAsInvalid()

			result.Error = fmt.Errorf("sending MAIL to host failed %w", err)
			return result
		}

		result.Validations |= VFValidRCPT

		if err := ctx.Err(); err != nil {
			result.Validations.MarkAsInvalid()

			result.Error = err
			return result
		}

		start = time.Now()
		err = client.Rcpt(e.Address)
		if err != nil {
			result.Error = fmt.Errorf("sending RCPT to host failed %w", err)
		}

		result.Timings.Add("Rcpt", time.Since(start))

		result.Validations |= VFValidRCPT

		// Flag the validation as Valid
		result.Validations |= VFValid

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
			Timings: make(Timings, 0, 1),
			Error:   nil,
		}

		start = time.Now()
		_, err := fetchMXHost(ctx, &resolver, e.Domain)
		result.Timings.Add("LookupMX", time.Since(start))
		result.Validations |= VFMXLookup

		if err != nil {
			result.Validations.MarkAsInvalid()
			result.Error = err
			return result
		}

		result.Validations |= VFValid

		return result
	}
}

// ValidateSyntax performs the most basic of checks: Does the syntax seem somewhat plausible.
func ValidateSyntax() Validator {
	return func(ctx context.Context, e types.EmailParts) Result {
		var start time.Time

		result := Result{
			Timings: make(Timings, 0, 1),
			Error:   nil,
		}

		start = time.Now()
		_, err := mail.ParseAddress(e.Address)
		result.Timings.Add("Structure", time.Since(start))
		result.Validations = 1 | VFSyntax

		if err != nil {
			result.Error = err
			return result
		}

		result.Validations = 1 | VFValid
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
