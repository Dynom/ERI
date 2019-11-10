package inspector

import (
	"testing"

	"github.com/Dynom/ERI/types"
)

func Test_MaskTest(t *testing.T) {

	t.Run("Flag values", func(t *testing.T) {
		t.Logf("VFValid       %d", types.VFValid)
		t.Logf("VFSyntax      %d", types.VFSyntax)
		t.Logf("VFMXLookup    %d", types.VFMXLookup)
		t.Logf("VFHostConnect %d", types.VFHostConnect)
		t.Logf("VFValidRCPT   %d", types.VFValidRCPT)
	})

	t.Run("Setting", func(t *testing.T) {
		var v types.Validations

		t.Logf("initial      %08b", v)
		// Setting MX?
		//v = 1 | VFMXLookup
		v |= types.VFMXLookup
		t.Logf("set mx?      %08b", v)
		//t.Logf("is valid? %08b", v&VFValid)

		v |= types.VFValid
		t.Logf("valid masked %08b", v&types.VFValid)
		t.Logf("is valid?    %08b", v)

		v = 0 &^ types.VFValid
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
