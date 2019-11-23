package validators

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
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

// ValidateFull performs a series of checks, from cheap to expensive and provides a fairly accurate result
func ValidateFull(dialer *net.Dialer) Validator {
	v := NewSMValidator(dialer)
	return func(ctx context.Context, e types.EmailParts) Result {
		a, err := v.CheckBasic(ctx, e)
		return Result{
			Error:       err,
			Timings:     a.Timings,
			Validations: a.Validations,
		}
	}
}

// ValidateMX validates if the domain of the address has MX records. This is a more basic check than ValidateMXAndRCPT,
// using them both together doesn't make a lot of sense. Use this for less precision and if speed is more important
// you could use this for an initial check instead
func ValidateMX() Validator {

	resolver := &net.Resolver{}
	return func(ctx context.Context, e types.EmailParts) Result {
		result := Result{
			Timings: make(types.Timings, 0, 1),
			Error:   nil,
		}

		var mxs []string
		result = validateStep(result, "LookupMX", VFMXLookup, func() error {
			var err error
			mxs, err = fetchMXHosts(ctx, resolver, e.Domain)
			return err
		})

		if result.Error != nil || len(mxs) == 0 {
			return result
		}

		result.Validations.MarkAsValid()
		return result
	}
}

// ValidateSyntax performs the most basic of checks: Does the syntax seem somewhat plausible.
func ValidateSyntax() Validator {
	return func(ctx context.Context, e types.EmailParts) Result {
		result := Result{
			Timings: make(types.Timings, 0, 1),
			Error:   nil,
		}

		validateStep(result, "Structure", VFSyntax, func() error {
			var err error
			_, err = mail.ParseAddress(e.Address)
			return err
		})

		if result.Error != nil {
			return result
		}

		result.Validations.MarkAsValid()
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
		return result
	}

	result.Validations = result.Validations.MergeWithNext(flag)
	return result
}
