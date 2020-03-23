package config

import (
	"testing"
)

func TestValidatorTypes_AsStringSlice(t *testing.T) {
	t.Run("alloc size test", func(t *testing.T) {
		v := ValidatorTypes{"a", "b"}
		if got := v.AsStringSlice(); cap(got) != len(got) {
			t.Errorf("Expected the capacity %d to be equal to the length %d, it wasn't.", cap(got), len(got))
		}

		if got := v.AsStringSlice(); len(got) != len(v) {
			t.Errorf("Got %d, expected a length of %d", len(got), len(v))
		}
	})
}

func TestValidatorType_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// The good
		{name: "Valid value", value: string(VTLookup)},

		// The bad
		{wantErr: true, name: "Invalid value", value: "Hakuna matata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := ValidatorType(tt.value)

			if err := vt.UnmarshalText([]byte(tt.value)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if _ = vt.UnmarshalText([]byte(tt.value)); string(vt) != tt.value {
				t.Errorf("UnmarshalText() value not on value receiver. Setting value %s doesn't reflect variable %v", tt.value, vt)
			}

		})
	}
}
