package validations

import "testing"

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
