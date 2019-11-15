package inspector

import (
	"context"
	"testing"
	"time"

	"github.com/Dynom/ERI/cmd/web/types"
)

func TestTimer(t *testing.T) {
	tt := time.NewTicker(time.Second * 1)
	defer tt.Stop()

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)

	done := make(chan bool, 0)
	go func() {
		for {
			select {
			case <-tt.C:

				// context deadline check
				if err := ctx.Err(); err != nil {
					t.Log("Deadline exceeded")
					return
				}

			case <-done:
				t.Log("Done, stopping ticket")
				tt.Stop()
				return
			}
		}
	}()
	time.Sleep(time.Second * 2)
	done <- true
	close(done)
}

func Test_MaskTest(t *testing.T) {

	t.Run("Flag values", func(t *testing.T) {
		t.Logf("VFValid        %08b %d", types.VFValid, types.VFValid)
		t.Logf("VFSyntax       %08b %d", types.VFSyntax, types.VFSyntax)
		t.Logf("VFMXLookup     %08b %d", types.VFMXLookup, types.VFMXLookup)
		t.Logf("VFHostConnect  %08b %d", types.VFHostConnect, types.VFHostConnect)
		t.Logf("VFValidRCPT    %08b %d", types.VFValidRCPT, types.VFValidRCPT)
		t.Logf("VFDisposable   %08b %d", types.VFDisposable, types.VFDisposable)
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

func Test_mightBeAHostOrIP(t *testing.T) {
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
