package validators

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynom/ERI/types"
)

func Test_ValidateMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		max       uint64
		input     string
		wantValid bool
		wantError bool
	}{
		// No error? We're golden
		{name: "basic valid", max: 5, input: "12345", wantError: false, wantValid: true},
		{name: "no max", max: 0, input: "123456789", wantError: false, wantValid: true},

		// Erroneous situations shouldn't result in a valid result
		{name: "basic invalid", max: 5, input: "123456", wantError: true, wantValid: false},
	}

	ctx := context.Background()
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			validator := ValidateMaxLength(tt.max)
			ep := types.EmailParts{}
			ep.Address = tt.input

			got := validator(ctx, ep)
			if (got.Error != nil) != tt.wantError || got.Validations.IsValid() != tt.wantValid {
				t.Errorf("ValidateMaxLength() Got err? %+v, want %t. Validations: %t, want: %t", got.Error, tt.wantError, got.Validations.IsValid(), tt.wantValid)
			}

			if tt.wantError && !errors.Is(got.Error, ErrValueTooGreat) {
				t.Errorf("Expected an error of type ErrValueTooGreat, instead I got %+v", got.Error)
			}
		})
	}
}
