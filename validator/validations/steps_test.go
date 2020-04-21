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
			s:    Steps(FSyntax),
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
			name: "Testing if the type has the flag",
			s:    Steps(FValid),
			args: args{
				f: FValid,
			},
			want: true,
		},
		{
			name: "Testing if the type has the flags",
			s:    Steps(FSyntax | FMXLookup | FValid),
			args: args{
				f: FMXLookup,
			},
			want: true,
		},
		{
			name: "Testing if the type doesn't have the flags",
			s:    Steps(FSyntax | FValid),
			args: args{
				f: FMXLookup,
			},
			want: false,
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
			// @todo test passes, but is the behaviour correct?
			name: "Testing if only next flags are added",
			s:    Steps(FSyntax),
			args: args{
				new: Steps(FValid),
			},
			want: Steps(FValid),
		},
		{
			name: "Testing if all new steps are merged",
			s:    Steps(FSyntax),
			args: args{
				new: Steps(FSyntax | FMXLookup | FValid),
			},
			want: Steps(FSyntax | FMXLookup | FValid),
		},
		{
			name: "Testing without steps",
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
			s:    Steps(FSyntax),
			args: args{
				new: FValid,
			},
			want: Steps(FSyntax | FValid),
		},
		{
			name: "Testing if flag is still set",
			s:    Steps(FSyntax | FValid | FMXLookup),
			args: args{
				new: FMXLookup,
			},
			want: Steps(FSyntax | FValid | FMXLookup),
		},
		{
			name: "Testing with no flag",
			s:    0,
			args: args{
				new: 0,
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
