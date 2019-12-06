package validators

import (
	"testing"

	"github.com/Dynom/ERI/validator/validations"
)

func TestValidations_Merge(t *testing.T) {
	tests := []struct {
		name string
		v    validations.Validations
		new  validations.Validations
		want validations.Validations
	}{
		{
			name: "Marking valid",
			v:    validations.Validations(0),
			new:  validations.VFValid,
			want: validations.VFValid,
		},
		{
			name: "Marking invalid",
			v:    validations.VFValid,
			new:  validations.Validations(0),
			want: validations.Validations(0),
		},
		{
			name: "Marking validations",
			v:    validations.Validations(0),
			new:  validations.VFMXLookup | validations.VFSyntax | validations.VFHostConnect,
			want: validations.VFMXLookup | validations.VFSyntax | validations.VFHostConnect,
		},
		{
			name: "Appending flags, excluding valid",
			v:    validations.VFSyntax,
			new:  validations.VFMXLookup | validations.VFHostConnect,
			want: validations.VFMXLookup | validations.VFSyntax | validations.VFHostConnect,
		},
		{
			name: "Appending flags, first valid, new invalid",
			v:    validations.VFSyntax | validations.VFValid,
			new:  validations.VFMXLookup | validations.VFHostConnect,
			want: validations.VFMXLookup | validations.VFSyntax | validations.VFHostConnect,
		},
		{
			name: "Appending flags, first invalid, new valid",
			v:    validations.VFSyntax,
			new:  validations.VFMXLookup | validations.VFHostConnect | validations.VFValid,
			want: validations.VFMXLookup | validations.VFSyntax | validations.VFHostConnect | validations.VFValid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MergeWithNext(tt.new); got != tt.want {
				t.Errorf("MergeWithNext() = %v, want %v", got, tt.want)
			}
		})
	}
}
