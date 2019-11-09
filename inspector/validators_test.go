package inspector

import (
	"testing"
)

func Test_MaskTest(t *testing.T) {

	t.Run("Flag values", func(t *testing.T) {
		t.Logf("VFValid       %d", VFValid)
		t.Logf("VFSyntax      %d", VFSyntax)
		t.Logf("VFMXLookup    %d", VFMXLookup)
		t.Logf("VFHostConnect %d", VFHostConnect)
		t.Logf("VFValidRCPT   %d", VFValidRCPT)
	})

	t.Run("Setting", func(t *testing.T) {
		var v Validations

		t.Logf("initial      %08b", v)
		// Setting MX?
		//v = 1 | VFMXLookup
		v |= VFMXLookup
		t.Logf("set mx?      %08b", v)
		//t.Logf("is valid? %08b", v&VFValid)

		v |= VFValid
		t.Logf("valid masked %08b", v&VFValid)
		t.Logf("is valid?    %08b", v)

		v = 0 &^ VFValid
		t.Logf("valid clear? %08b", v)

	})

}

func Test_looksLikeAHost(t *testing.T) {
	type args struct {
		h string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{name: "localhost", args: args{h: "localhost"}, want: true},
		{name: "IP", args: args{h: "127.0.0.1"}, want: true},

		{name: "Domain", args: args{h: "example.org"}, want: true},
		{name: "Domain", args: args{h: "example.co.uk"}, want: true},

		{name: "dot", args: args{h: "."}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mightBeAHostOrIP(tt.args.h); got != tt.want {
				t.Errorf("mightBeAHostOrIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
