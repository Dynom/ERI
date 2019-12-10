package inspector

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/Dynom/ERI/types"
)

func Test_Check(t *testing.T) {
	validationResult := validations.Validations(0)
	validationResult |= validations.VFValid
	validationResult |= validations.VFMXLookup

	insp := New(WithValidators(
		validateStub(validationResult),
	))

	result := insp.Check(context.Background(), "foo@bar")

	if result.Error != nil {
		t.Errorf("Error: %+v", result.Error)
	}

	if !result.IsValid() {
		t.Errorf("Expecting a valid resulit, instead I got %+v", result)
	}
}

// validateStub is a stub validator
func validateStub(v validations.Validations) Validator {
	var err error
	if v&validations.VFValid == 0 {
		err = errors.New("validateStub returning an error")
	}

	return func(ctx context.Context, e types.EmailParts) Result {
		return Result{
			Error:       err,
			Timings:     make(validator.Timings, 0),
			Validations: v,
		}
	}
}

func TestChecker_CheckIncrementalValidators(t *testing.T) {
	tests := []struct {
		name          string
		validators    []Validator
		shouldBeValid bool
	}{
		{
			name:          "single validator",
			shouldBeValid: true,
			validators: []Validator{
				validateStub(0 | validations.VFValid), // Valid
			},
		},
		{
			// Validators are incremental, once we have a failure, we can't recover
			name:          "multi validator, start invalid end with valid",
			shouldBeValid: false,
			validators: []Validator{
				validateStub(0), // Invalid
				validateStub(0 | validations.VFSyntax | validations.VFValid), // Valid
			},
		},
		{
			name:          "multi validator, start valid end with invalid",
			shouldBeValid: false,
			validators: []Validator{
				validateStub(0 | validations.VFSyntax | validations.VFValid),    // Valid
				validateStub(0 | validations.VFSyntax | validations.VFMXLookup), // Invalid
			},
		},
	}

	email := "foo@example.org"
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Checker{
				validators: tt.validators,
			}

			if got := c.Check(ctx, email); got.IsValid() != tt.shouldBeValid {
				t.Errorf("Check() = %+v, got %t want %t", got, got.IsValid(), tt.shouldBeValid)
			}
		})
	}
}
