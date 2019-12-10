package inspector

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
)

var (
	ErrValueTooGreat = errors.New("value too great")
)

// Validator is the type all validators must conform to
type Validator func(ctx context.Context, e types.EmailParts) Result

// Result is the validation result
type Result struct {
	Error error
	validator.Timings
	validations.Validations
}

// ValidateSyntax performs a series of checks, from cheap to expensive and provides a fairly accurate result
func ValidateSyntax(dialer *net.Dialer) Validator {
	v := validator.NewEmailAddressValidator(dialer)
	return func(ctx context.Context, e types.EmailParts) Result {
		a, err := v.CheckWithSyntax(ctx, e)
		return Result{
			Error:       err,
			Timings:     a.Timings,
			Validations: a.Validations,
		}
	}
}

// ValidateMaxLength performs a byte length check on the entire address. Plays nice when max == 0
func ValidateMaxLength(max uint64) Validator {
	var result = Result{
		Error:       nil,
		Timings:     make(validator.Timings, 0),
		Validations: 0,
	}

	if max == 0 {
		result := result
		result.MarkAsValid()
		// noop
		return func(ctx context.Context, e types.EmailParts) Result {
			return result
		}
	}

	return func(ctx context.Context, e types.EmailParts) Result {
		result := result
		if uint64(len(e.Address)) > max {
			result.Error = fmt.Errorf("%w max of %d bytes", ErrValueTooGreat, max)
			result.MarkAsInvalid()
			return result
		}

		result.MarkAsValid()
		return result
	}
}
