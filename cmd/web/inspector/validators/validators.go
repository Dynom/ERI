package validators

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/Dynom/ERI/cmd/web/types"
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
	Validations
}

func ValidateFull() Validator {
	r := &net.Resolver{}
	d := &net.Dialer{}
	v := NewSMValidator(r, d)
	return func(ctx context.Context, e types.EmailParts) Result {
		a, err := v.CheckEmailAddress(ctx, e.Address)
		return Result{
			Error:       err,
			Timings:     a.Timings,
			Validations: a.Validations,
		}
	}
}

// ValidateMXAndRCPT validates if the mailbox exists. You can control the timeout by using context
func ValidateMXAndRCPT(recipient string) Validator {
	resolver := net.Resolver{}
	dialer := &net.Dialer{}

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

		var mxHosts []string
		result = validateStep(result, "LookupMX", VFMXLookup, func() error {
			var err error
			mxHosts, err = fetchMXHosts(ctx, &resolver, e.Domain)

			return err
		})

		if result.Error != nil {
			return result
		}

		var conn net.Conn
		result = validateStep(result, "Dial", VFHostConnect, func() error {
			var err error
			conn, err = getConnection(ctx, dialer, mxHosts[0])
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

		result = validateStep(result, "Mail", VFValidRCPT, func() error {
			return client.Mail(recipient)
		})

		if result.Error != nil {
			return result
		}

		result = validateStep(result, "Rcpt", VFValidRCPT|VFValid, func() error {
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
		_, err := fetchMXHosts(ctx, &resolver, e.Domain)
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
			Timings: make(types.Timings, 0, 1),
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

// validateStep encapsulates some boilerplate. If the callback returns with a nil error, the flags are applied
func validateStep(result Result, stepName string, flag Validations, fn func() error) Result {
	start := time.Now()
	err := fn()
	result.Timings.Add(stepName, time.Since(start))

	if err != nil {
		result.Error = fmt.Errorf("step %s failed, error: %w", stepName, err)
	} else {
		result.Validations = result.Validations.MergeWithNext(flag)
	}

	return result
}
