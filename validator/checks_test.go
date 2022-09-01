package validator

import (
	"context"
	"errors"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator/validations"
)

func Test_checkEmailAddressSyntax(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		// All good
		{name: "valid but short local", email: "e@wx.yz"},
		{name: "valid but short", email: "me@wx.yz"},
		{name: "with subdomain", email: "john@doe.example.org"},
		{name: "wrong tld, but valid syntax", email: "js@example.mail"},

		{name: "Unicode", email: "ทีเ@อชนิค.ไทย"},

		// All bad
		{name: "Invalid visible character", email: "js@d.org>", wantErr: true},
		{name: "ending on a dot", email: "js@example.org.", wantErr: true},
		{name: "White space character in local part", email: "joh n@hot1mail.com", wantErr: true},
		{name: "missing local", email: "@hot1mail.com", wantErr: true},
		{name: "missing domain", email: "john.doe@", wantErr: true},

		// Not picked up by mail.ParseAddress
		{name: "Invalid characters (NBSP)", email: "j\u00a0s@example.org", wantErr: true},
		{name: "Invalid characters (NBSP)", email: "js@example.org\u00a0", wantErr: true},
		{name: "Invalid characters (NL)", email: "john.doe@example.org\njane@foo", wantErr: true},
		{name: "Invalid characters (NL) with valid e-mail suffix", email: "john.doe@example.org\njane@example.org", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			a := &Artifact{
				Validations: 0,
				Timings:     make(Timings, 10),
			}

			a.email, err = types.NewEmailParts(tt.email)
			if err != nil && !tt.wantErr {
				t.Errorf("types.NewEmailParts(%q) error = %v", tt.email, err)
			}

			if err := checkEmailAddressSyntax(a); (err != nil) != tt.wantErr {
				t.Errorf("checkEmailAddressSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}

			if _, err := mail.ParseAddress(a.email.Address); (err != nil) != tt.wantErr {
				t.Logf("Wouldn't have been picked up by mail.ParseAddress(): '%s'", a.email.Address)
			}
		})
	}

	for _, tt := range []struct {
		parts   types.EmailParts
		wantErr bool
	}{
		{parts: types.NewEmailFromParts("", "example.org"), wantErr: true},
		{parts: types.NewEmailFromParts("john.doe", ""), wantErr: true},
	} {
		t.Run("only structure check/"+tt.parts.Address, func(t *testing.T) {

			a := &Artifact{
				Validations: 0,
				Timings:     make(Timings, 10),
				email:       tt.parts,
			}

			if err := checkEmailAddressSyntax(a); (err != nil) != tt.wantErr {
				t.Errorf("checkEmailAddressSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checkDomainSyntax(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		// All good
		{name: "valid but short", domain: "wx.yz"},
		{name: "with sub-domain", domain: "doe.example.org"},
		{name: "wrong tld, but valid syntax", domain: "example.mail"},

		{name: "Unicode", domain: "อชนิค.ไทย"},

		// All bad
		{name: "Invalid visible character", domain: "d.org>", wantErr: true},
		{name: "ending on a dot", domain: "example.org.", wantErr: true},
		{name: "too long", domain: strings.Repeat("a", 250) + ".org", wantErr: true},

		// Not picked up by mail.ParseAddress
		{name: "Invalid characters (NBSP)", domain: "example.org\u00a0", wantErr: true},
		{name: "Invalid characters (NL)", domain: "example.org\nexample.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Artifact{
				Validations: 0,
				Timings:     make(Timings, 10),
				email: types.EmailParts{
					Address: "",
					Local:   "",
					Domain:  tt.domain,
				},
			}

			if err := checkDomainSyntax(a); (err != nil) != tt.wantErr {
				t.Errorf("checkEmailAddressSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_looksLikeValidLocalPartSpecifics(t *testing.T) {

	// Should match up with the classes we test in our regexes
	localSpecifics := "!#$%&'*+-/=?^_\x60{|}~"

	for _, r := range localSpecifics {
		local := `john` + string(r) + `doe`
		if !looksLikeValidLocalPart(local) {
			t.Errorf("looksLikeValidLocalPart(%q) = false, want true", local)
		}
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
		{want: true, local: "john`doe"},      // \x60
		{want: true, local: "john\u00C0doe"}, // First letter in the Latin-1 supplement block

		// The good, Unicode
		{want: true, local: "用户"},       // Chinese
		{want: true, local: "अजय"},      // Hindi
		{want: true, local: "квіточка"}, // Ukrainian
		{want: true, local: "θσερ"},     // Greek
		{want: true, local: "Dörte"},    // German
		{want: true, local: "коля"},     // Russian

		// The bad
		{local: ""},
		{local: "."},
		{local: "john doe"},
		{local: "john\ndoe"},
		{local: "john.doe\n"},
		{local: "john.doe\u00A0"}, // nbsp

		// Common copy&paste mistakes
		{local: "\u2018john"}, // A common quotation mark
		{local: "\u00A0john"}, // A common space character, found in the lower end of the Latin-1 supplement block
		{local: "\u2000john"}, // Start of of the Unicode General Punctuation block
		{local: "\u206Fjohn"}, // End of the Unicode General Punctuation block
	}

	for _, tt := range tests {
		tt := tt
		t.Run("testing "+tt.local, func(t *testing.T) {
			if got := looksLikeValidLocalPart(tt.local); got != tt.want {
				t.Errorf("looksLikeValidLocalPart(%q) = %v, want %v", tt.local, got, tt.want)
			}
		})
	}
}

func Test_looksLikeValidDomain(t *testing.T) {
	const (

		// Explicitly testing real-world occurring characters
		char0020 rune = 0x0020 // U+0020 (SP)
		char00A0 rune = 0x00a0 // U+00A0 (NBSP)
		char0009 rune = 0x0009 // control character
		char0010 rune = 0x0010 // control character
		char000a rune = 0x000a
		char003e rune = 0x003e
	)

	tests := []struct {
		domain  string
		badChar string
		want    bool
	}{
		// The good
		{want: true, domain: "example.org"},
		{want: true, domain: "a.b.c.d.e.f.g.h.i.j.example.org"},
		{want: true, domain: "d-ash.example.org"},
		{want: true, domain: "ex-ample.org"},
		{want: true, domain: "eXample.org"},
		{want: true, domain: "ex4mple.org"},

		// Unicode
		{want: true, domain: "إختبار.إختبار"},       // Arabic
		{want: true, domain: "آزمایشی.آزمایشی"},     // Persian	Arabic
		{want: true, domain: "测试.测试"},               // Chinese	Han (Simplified variant)
		{want: true, domain: "測試.測試"},               // Chinese	Han (Traditional variant)
		{want: true, domain: "испытание.испытание"}, // Russian	Cyrillic
		{want: true, domain: "परीक्षा.परीक्षा"},     // Hindi	Devanagari (Nagari)
		{want: true, domain: "δοκιμή.δοκιμή"},       // Greek, Modern (1453-)	Greek
		{want: true, domain: "테스트.테스트"},             // Korean	Hangul (Hangŭl, Hangeul)
		{want: true, domain: "טעסט.טעסט"},           // Yiddish	Hebrew
		{want: true, domain: "テスト.テスト"},             // Japanese	Katakana
		{want: true, domain: "பரிட்சை.பரிட்சை"},     // Tamil	Tamil

		// Punycode domains
		{want: true, domain: "xn--kgbechtv.xn--kgbechtv"},
		{want: true, domain: "xn--hgbk6aj7f53bba.xn--hgbk6aj7f53bba"},

		// The bad - length
		{domain: ""},
		{domain: "a.a"},

		// The bad - Spacing
		{domain: "example.org", badChar: "."},
		{domain: "example.org", badChar: string(char0020)},
		{domain: "example.org", badChar: string(char00A0)},
		{domain: "example.org", badChar: string(char0009)},
		{domain: "example.org", badChar: string(char00A0)},
		{domain: "example.org", badChar: string(char0020)},
		{domain: "example.org", badChar: string(char0010)},
		{domain: "example.org", badChar: string(char000a)},
		{domain: "example.org", badChar: string(char003e)},
		{domain: "example.org", badChar: "\n"},

		// The bad - Odd, but common, characters
		{domain: "example.org", badChar: ">"},
		{domain: "example.org", badChar: ","},
		{domain: "example.org", badChar: ")"},
	}
	for _, tt := range tests {
		domain := tt.domain + tt.badChar
		t.Run("testing "+domain, func(t *testing.T) {
			if got := looksLikeValidDomain(domain); got != tt.want {
				t.Errorf("looksLikeValidDomain(%q) = %v, want %v (bad char: 0x%x, %q))", tt.domain, got, tt.want, tt.badChar, tt.badChar)
			}
		})
	}
}

func Test_checkIfDomainHasMX(t *testing.T) {
	tests := []struct {
		name     string
		resolver LookupMX
		steps    validations.Steps
		wantErr  bool
	}{
		{
			name:     "all good",
			resolver: buildLookupMX([]string{"example.org"}, nil),
			wantErr:  false,
		},
		{
			name:     "lookup error",
			resolver: buildLookupMX([]string{"example.org"}, errors.New("rip")),
			wantErr:  true,
		},
		{
			name:     "no mx hosts",
			resolver: buildLookupMX([]string{}, nil),
			wantErr:  true,
		},
		{
			name:     "bad MX hosts",
			resolver: buildLookupMX([]string{".", ".."}, nil),
			wantErr:  true,
		},
		{
			name:     "Step already defined, but not valid",
			resolver: buildLookupMX([]string{".", ".."}, nil),
			steps:    validations.Steps(validations.FMXLookup),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a := &Artifact{
				resolver:    tt.resolver,
				Steps:       tt.steps,
				Validations: 0,
			}

			if err := checkIfDomainHasMX(a); (err != nil) != tt.wantErr {
				t.Errorf("checkIfDomainHasMX() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Since we run an MX check, the step should always be defined
			if !a.Steps.HasFlag(validations.FMXLookup) {
				t.Errorf("Expected the flag to be specified as Step %+v", a.Steps)
			}

			// Depending on the validations, we either want or don't want the flag defined
			if !tt.wantErr && !a.Validations.HasFlag(validations.FMXLookup) {
				t.Errorf("Expected the flag to be specified as Validation %+v", a.Validations)
			} else if tt.wantErr && a.Validations.HasFlag(validations.FMXLookup) {
				t.Errorf("Expected the flag to NOT be specified as Validation %+v", a.Validations)
			}
		})
	}
}

func Test_checkIfMXHasIP(t *testing.T) {

	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now())
	cancel()

	tests := []struct {
		name     string
		resolver LookupMX
		steps    validations.Steps
		mx       []string
		ctx      context.Context
		wantErr  bool
	}{
		{
			name:     "all good",
			resolver: buildLookupMX([]string{"example.org"}, nil),
			wantErr:  false,
		},
		{
			name:     "lookup fail",
			resolver: buildLookupMX([]string{"127.0.0.1"}, errors.New("lookup fail")),
			mx:       []string{"mx.example.org"},
			wantErr:  true,
		},
		{
			name:     "deadline expired",
			resolver: &net.Resolver{},
			mx:       []string{"mx.example.org"},
			ctx:      expiredCtx,
			wantErr:  true,
		},
		{
			name:    "Step already defined, but not valid",
			steps:   validations.Steps(validations.FMXDomainHasIP),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.ctx == nil {
				tt.ctx = context.Background()
			}

			a := &Artifact{
				resolver:    tt.resolver,
				Steps:       tt.steps,
				mx:          tt.mx,
				ctx:         tt.ctx,
				Validations: 0,
			}

			err := checkIfMXHasIP(a)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkIfMXHasIP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
