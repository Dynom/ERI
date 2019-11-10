package inspector

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynom/ERI/types"
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
func validateStub(v types.Validations) Validator {
	var err error
	if v&types.VFValid == 0 {
		err = errors.New("stuff failed")
	}

	return func(ctx context.Context, e types.EmailParts) Result {
		return Result{
			Error:       err,
			Timings:     make(types.Timings, 0),
			Validations: v,
		}
	}
}
