package inspector

import (
	"context"
	"fmt"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/types"
)

// New creates a new Checker and applies any specified functional Option argument
func New(options ...Option) Checker {
	mc := Checker{}

	for _, o := range options {
		o(&mc)
	}

	return mc
}

// Checker holds the type that can perform the e-mail address checks
type Checker struct {
	validators []Validator
}

// Check runs various validators on the input and produces a Result
func (c Checker) Check(ctx context.Context, email string) Result {
	e, err := types.NewEmailParts(email)
	if err != nil {
		return newErrorResult(err)
	}

	var result = Result{
		Timings: make(validator.Timings, 0, len(c.validators)),
	}

	for _, v := range c.validators {
		r := v(ctx, e)

		// Set Validations used
		result.Validations = result.Validations.MergeWithNext(r.Validations)

		// Append to timers
		for _, t := range r.Timings {
			result.Timings.Add(t.Label, t.Duration)
		}

		// Wrap the error
		if r.Error != nil {
			result.Error = wrapError(result.Error, r.Error)
		}

		if ctx.Err() != nil {
			result.Error = wrapError(result.Error, ctx.Err())
		}

		// Since we're working incrementally we can bail, since we'll never recover to a valid state
		if result.Error != nil || !result.IsValid() {
			return result
		}
	}

	return result
}

func newErrorResult(err error) Result {
	return Result{
		Timings: validator.Timings{},
		Error:   err,
	}
}

// wrapError wraps an error with the parent error and ignores the parent when it's nil
func wrapError(parent error, new error) error {
	if parent == nil {
		return new
	}

	return fmt.Errorf("%s %w", parent, new)
}
