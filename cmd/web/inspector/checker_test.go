package inspector

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynom/ERI/cmd/web/inspector/validators"

	"github.com/Dynom/ERI/cmd/web/types"
)

func Test_Check(t *testing.T) {
	validationResult := types.Validations(0)
	validationResult |= types.VFValid
	validationResult |= types.VFMXLookup

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
func validateStub(v types.Validations) validators.Validator {
	var err error
	if v&types.VFValid == 0 {
		err = errors.New("validateStub returning an error")
	}

	return func(ctx context.Context, e types.EmailParts) validators.Result {
		return validators.Result{
			Error:       err,
			Timings:     make(types.Timings, 0),
			Validations: v,
		}
	}
}

func TestChecker_CheckIncrementalValidators(t *testing.T) {
	tests := []struct {
		name          string
		validators    []validators.Validator
		shouldBeValid bool
	}{
		{
			name:          "single validator",
			shouldBeValid: true,
			validators: []validators.Validator{
				validateStub(0 | types.VFValid), // Valid
			},
		},
		{
			name:          "multi validator, start invalid end with valid",
			shouldBeValid: true,
			validators: []validators.Validator{
				validateStub(0), // Invalid
				validateStub(0 | types.VFSyntax | types.VFValid), // Valid
			},
		},
		{
			name:          "multi validator, start valid end with invalid",
			shouldBeValid: false,
			validators: []validators.Validator{
				validateStub(0 | types.VFSyntax | types.VFValid),    // Valid
				validateStub(0 | types.VFSyntax | types.VFMXLookup), // Invalid
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
