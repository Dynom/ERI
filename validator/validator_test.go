package validator

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator/validations"
)

var (
	expiredDeadlineContext, _ = context.WithDeadline(context.Background(), time.Now())
)

func TestEmailValidator_CheckWithLookup(t *testing.T) {
	validParts, _ := types.NewEmailParts("john.doe@example.org")

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

				dialer: &net.Dialer{
					Deadline: time.Now().Add(10 * time.Second),
					Timeout:  10 * time.Second,
					Resolver: &net.Resolver{
						PreferGo:     true,
						StrictErrors: true,
					},
				},
			}

			got := v.CheckWithLookup(tt.args.ctx, tt.args.emailParts)
			if got.Validations.IsValid() == tt.wantErr {
				t.Errorf("Expected validations to be invalid on error: %b", got.Validations)
			}
		})
	}
}

func Test_validateSequence(t *testing.T) {
	ctxDeadline, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	type args struct {
		ctx      context.Context
		artifact Artifact
		sequence []stateFn
	}
	tests := []struct {
		name    string
		args    args
		want    Artifact
		wantErr bool
	}{
		{
			name: "Testing if sequence is valid",
			args: args{
				ctx: context.Background(),
				artifact: Artifact{
					email: types.EmailParts{
						Address: "johndoe@example.org",
						Local:   "test",
						Domain:  "example.org",
					},
				},
				sequence: []stateFn{
					checkEmailAddressSyntax,
				},
			},
			want: Artifact{
				Validations: validations.Validations(validations.FValid | validations.FSyntax),
				Steps:       validations.Steps(validations.FSyntax),
				email: types.EmailParts{
					Address: "johndoe@example.org",
					Local:   "test",
					Domain:  "example.org",
				},
			},
		},
		{
			name: "Testing with email-error",
			args: args{
				ctx: context.Background(),
				artifact: Artifact{
					email: types.EmailParts{
						Address: "johndoeexample.org",
					},
				},
				sequence: []stateFn{
					checkEmailAddressSyntax,
				},
			},
			want: Artifact{
				Validations: 0,
				Steps:       validations.Steps(validations.FSyntax),
				email: types.EmailParts{
					Address: "johndoeexample.org",
				},
			},
			wantErr: true,
		},
		{
			name: "Testing with context deadline",
			args: args{
				ctx: ctxDeadline,
				artifact: Artifact{
					email: types.EmailParts{
						Address: "johndoe@example.org",
						Local:   "test",
						Domain:  "example.org",
					},
				},
				sequence: []stateFn{
					checkEmailAddressSyntax,
				},
			},
			want: Artifact{
				Validations: validations.Validations(validations.FSyntax),
				Steps:       validations.Steps(validations.FSyntax),
				email: types.EmailParts{
					Address: "johndoe@example.org",
					Local:   "test",
					Domain:  "example.org",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSequence(tt.args.ctx, tt.args.artifact, tt.args.sequence)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSequence() error = %v, wantErr %v", got, tt.want)
				return
			}
			if !reflect.DeepEqual(got.Validations, tt.want.Validations) {
				t.Errorf("Validations got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got.Steps, tt.want.Steps) {
				t.Errorf("Steps got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got.email, tt.want.email) {
				t.Errorf("Email got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got.ctx, tt.want.ctx) {
				t.Errorf("Context got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_prependOptions(t *testing.T) {
	type args struct {
		options []ArtifactFn
		o       []ArtifactFn
	}
	tests := []struct {
		name string
		args args
		want []ArtifactFn
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prependOptions(tt.args.options, tt.args.o...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prependOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmailAddressValidator(t *testing.T) {
	var v EmailValidator
	v = NewEmailAddressValidator(nil)
	if v.dialer == nil {
		t.Errorf("Expected a default dialer to be present when constructed with nil argument.")
		return
	}

	v = NewEmailAddressValidator(&net.Dialer{Resolver: nil})
	if v.dialer.Resolver == nil {
		t.Errorf("Expected a default resolver to be present, when when none was defined")
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

		// Regex fallback
		"短短.co",
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
