package validations

import "testing"

func TestSteps_HasBeenValidated(t *testing.T) {
	tests := []struct {
		name string
		s    Steps
		want bool
	}{
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
