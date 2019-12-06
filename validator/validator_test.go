package validator

import (
	"context"
	"net"
	"net/mail"
	"testing"
	"time"

	"github.com/Dynom/ERI/types"
)

var (
	expiredDeadlineContext, _ = context.WithDeadline(context.Background(), time.Now())
)

var (
	validParts, _ = types.NewEmailParts("john.doe@example.org")
)

func TestSMValidator_CheckWithLookup(t *testing.T) {
	t.Skipf("Need net.* stubbing for this to properly work")

	type args struct {
		ctx        context.Context
		emailParts types.EmailParts
	}

	p2, _ := types.NewEmailParts("jake@grr.la")
	tests := []struct {
		name    string
		args    args
		want    Artifact
		wantErr bool
	}{
		{name: "Checks halt on context timeout", args: args{ctx: expiredDeadlineContext, emailParts: validParts}, want: Artifact{}, wantErr: true},
		{name: "exists", args: args{ctx: context.Background(), emailParts: p2}, want: Artifact{}},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			v := &EmailValidator{

				// Missing PTR?
				dialer: &net.Dialer{
					Deadline: time.Now().Add(10 * time.Second),
					Timeout:  10 * time.Second,
					Resolver: &net.Resolver{
						PreferGo:     true,
						StrictErrors: true,
					},
				},
			}

			got, err := v.CheckWithLookup(tt.args.ctx, tt.args.emailParts)
			t.Logf("Got: %b (%s)", got.Validations, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckFull() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && got.Validations.IsValid() == tt.wantErr {
				t.Errorf("Expected validations to be invalid on error: %b", got.Validations)
			}
		})
	}
}

func TestSMValidator_getNewArtifact(t *testing.T) {
	d := net.Dialer{}
	v := NewEmailAddressValidator(&d)

	t.Run("Context deadline is propagated", func(t *testing.T) {

		deadline := time.Now().Add(time.Minute * 1)
		ctx, _ := context.WithDeadline(context.Background(), deadline)

		a := v.getNewArtifact(ctx, types.EmailParts{})
		if a.dialer.Deadline.UTC() != deadline.UTC() {
			t.Errorf("Expected the deadline to propagate, it didn't %s\n%+v", deadline, a)
		}
	})
}

func Test_checkSyntax(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		// All good
		{name: "valid but short", email: "john@doe.example.org"},
		{name: "wrong tld, but valid syntax", email: "j@example.mail"},

		// All bad
		{name: "Invalid visible character", email: "j@d.org>", wantErr: true},
		{name: "ending on a dot", email: "j@example.org.", wantErr: true},
		{name: "missing local part and @", email: "example.org", wantErr: true},
		{name: "missing local part", email: "@example.org", wantErr: true},

		// Not picked up by mail.ParseAddress
		{name: "Invalid characters (NBSP)", email: "j@example.org   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Artifact{
				Validations: 0,
				Timings:     make(Timings, 10),
			}

			a.email, _ = types.NewEmailParts(tt.email)
			if err := checkSyntax(a); (err != nil) != tt.wantErr {
				t.Errorf("checkSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}

			if _, err := mail.ParseAddress(a.email.Address); (err != nil) != tt.wantErr {
				t.Logf("Wouldn't have been picked up by mail.ParseAddress: '%s'", a.email.Address)
			}
		})
	}
}

func Test_looksLikeValidLocalPart(t *testing.T) {
	tests := []struct {
		local string
		want  bool
	}{
		// The good
		{want: true, local: "john.doe"},
		{want: true, local: "j0hn.doe"},
		{want: true, local: "John.doe"},

		// The bad
		{local: "."},
		{local: "john doe"},
	}
	for _, tt := range tests {
		t.Run("testing "+tt.local, func(t *testing.T) {
			if got := looksLikeValidLocalPart(tt.local); got != tt.want {
				t.Errorf("looksLikeValidLocalPart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_looksLikeValidDomain(t *testing.T) {
	const (
		char0020 rune = 0x0020 // U+0020 (SP)
		char00A0 rune = 0x00a0 // U+00A0 (NBSP)
		char0009 rune = 0x0009 // control character
		char0010 rune = 0x0010 // control character
		char000a rune = 0x000a
		char003e rune = 0x003e
	)

	tests := []struct {
		local   string
		badChar string
		want    bool
	}{
		// The good
		{want: true, local: "example.org"},
		{want: true, local: "a.b.c.d.e.f.g.h.i.j.example.org"},
		{want: true, local: "d-ash.example.org"},
		{want: true, local: "ex-ample.org"},

		// The bad - Spacing
		{local: "example.org", badChar: "."},
		{local: "example.org", badChar: string(char0020)},
		{local: "example.org", badChar: string(char00A0)},
		{local: "example.org", badChar: string(char0009)},
		{local: "example.org", badChar: string(char00A0)},
		{local: "example.org", badChar: string(char0020)},
		{local: "example.org", badChar: string(char0010)},
		{local: "example.org", badChar: string(char000a)},
		{local: "example.org", badChar: string(char003e)},

		// The bad - Odd, but common, characters
		{local: "example.org", badChar: ">"},
		{local: "example.org", badChar: ","},
		{local: "example.org", badChar: ")"},
	}
	for _, tt := range tests {
		domain := tt.local + tt.badChar
		t.Run("testing "+domain, func(t *testing.T) {
			if got := looksLikeValidDomain(domain); got != tt.want {
				t.Errorf("looksLikeValidLocalPart() = %v, want %v (bad char: 0x%x, %q))", got, tt.want, tt.badChar, tt.badChar)
			}
		})
	}
}

var looksLikeValidDomainResult bool

func Benchmark_looksLikeValidDomain(b *testing.B) {
	var tests = []string{
		// good
		"example.org",
		"a.b.c.d.e.f.g.h.i.j.example.org",
		"d-ash.example.org",

		// bad
		"a.b",
		"example.org.",
		" example.org",
		"example.org ",
		"eXamPLE.org ",
	}

	b.ReportAllocs()
	for _, tt := range tests {
		b.Run("testing '"+tt+"'", func(b *testing.B) {
			tt := tt
			for i := 0; i < b.N; i++ {
				looksLikeValidDomainResult = looksLikeValidDomain(tt)
			}
		})

		_ = !!looksLikeValidDomainResult
	}
}
