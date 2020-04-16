package validations

import (
	"testing"
)

func TestSteps_HasBeenValidated(t *testing.T) {
	tests := []struct {
		name string
		s    Steps
		want bool
	}{
		{
			name: "Testing if steps have been taken",
			s:    255,
			want: true,
		},
		{
			name: "Testing with no steps",
			s:    0,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.HasBeenValidated(); got != tt.want {
				t.Errorf("HasBeenValidated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteps_HasFlag(t *testing.T) {
	type args struct {
		f Flag
	}
	tests := []struct {
		name string
		s    Steps
		args args
		want bool
	}{
		{
			name: "Testing if the type has a flag",
			s:    255,
			args: args{
				f: 1,
			},
			want: true,
		},
		{
			name: "Testing with no flag",
			s:    0,
			args: args{
				f: 0,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.HasFlag(tt.args.f); got != tt.want {
				t.Errorf("HasFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteps_MergeWithNext(t *testing.T) {
	type args struct {
		new Steps
	}
	tests := []struct {
		name string
		s    Steps
		args args
		want Steps
	}{
		{
			name: "Testing if new steps are merged",
			s:    255,
			args: args{
				new: Steps(255),
			},
			want: 255,
		},
		{
			name: "Testing with no steps",
			s:    0,
			args: args{
				new: Steps(0),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.MergeWithNext(tt.args.new); got != tt.want {
				t.Errorf("MergeWithNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteps_RemoveFlag(t *testing.T) {
	tests := []struct {
		name string
		s    Steps
		f    Flag
		want Steps
	}{
		{
			name: "Testing with zero-values",
			s:    0,
			f:    0,
			want: 0,
		},
		{name: "Clears Single flag", s: Steps(FSyntax | FMXLookup), f: FSyntax, want: Steps(FMXLookup)},
		{name: "Doesn't clear non existing flag", s: Steps(FSyntax | FMXLookup | FValid), f: FValidRCPT, want: Steps(FSyntax | FMXLookup | FValid)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.RemoveFlag(tt.f); got != tt.want {
				t.Errorf("RemoveFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteps_SetFlag(t *testing.T) {
	type args struct {
		new Flag
	}
	tests := []struct {
		name string
		s    Steps
		args args
		want Steps
	}{
		{
			name: "Testing if new flag is set",
			s:    255,
			args: args{
				new: Flag(255),
			},
			want: 255,
		},
		{
			name: "Testing with no flag",
			s:    0,
			args: args{
				new: Flag(0),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.SetFlag(tt.args.new); got != tt.want {
				t.Errorf("SetFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteps_String(t *testing.T) {
	tests := []struct {
		name string
		s    Steps
		want string
	}{
		{
			name: "Converting steps to string",
			s:    255,
			want: "11111111",
		},
		{
			name: "Testing with zero-value",
			s:    0,
			want: "00000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
