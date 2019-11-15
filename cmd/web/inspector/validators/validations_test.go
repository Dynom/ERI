package validators

import "testing"

func TestValidations_Merge(t *testing.T) {
	type args struct {
		new Validations
	}
	tests := []struct {
		name string
		v    Validations
		new  Validations
		want Validations
	}{
		{
			name: "Marking valid",
			v:    Validations(0),
			new:  VFValid,
			want: VFValid,
		},
		{
			name: "Marking invalid",
			v:    VFValid,
			new:  Validations(0),
			want: Validations(0),
		},
		{
			name: "Marking validations",
			v:    Validations(0),
			new:  VFMXLookup | VFSyntax | VFHostConnect,
			want: VFMXLookup | VFSyntax | VFHostConnect,
		},
		{
			name: "Appending flags, excluding valid",
			v:    VFSyntax,
			new:  VFMXLookup | VFHostConnect,
			want: VFMXLookup | VFSyntax | VFHostConnect,
		},
		{
			name: "Appending flags, first valid, new invalid",
			v:    VFSyntax | VFValid,
			new:  VFMXLookup | VFHostConnect,
			want: VFMXLookup | VFSyntax | VFHostConnect,
		},
		{
			name: "Appending flags, first invalid, new valid",
			v:    VFSyntax,
			new:  VFMXLookup | VFHostConnect | VFValid,
			want: VFMXLookup | VFSyntax | VFHostConnect | VFValid,
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
