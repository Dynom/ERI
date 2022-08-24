package validator

import (
	"context"
	"errors"
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

	ctxExpiredDeadline, cancel := context.WithTimeout(context.Background(), -1*time.Hour)
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
			name: "A sequence without errors should return with FValid",
			args: args{
				ctx: context.Background(),
				sequence: []stateFn{
					func(a *Artifact) error {
						// Making sure we don't have the FValid flag
						a.Validations.RemoveFlag(validations.FValid)
						return nil
					},
				},
			},
			want: Artifact{
				Validations: validations.Validations(validations.FValid),
				Steps:       0,
			},
		},
		{
			name: "A sequence with errors shouldn't contain FValid",
			args: args{
				ctx: context.Background(),
				sequence: []stateFn{
					func(a *Artifact) error {
						a.Validations.SetFlag(validations.FSyntax)
						return errors.New("b0rk")
					},
				},
			},
			want: Artifact{
				Validations: validations.Validations(validations.FSyntax),
				Steps:       0,
			},
			wantErr: true,
		},
		{
			name: "A sequence with errors should exit early",
			args: args{
				ctx: context.Background(),
				sequence: []stateFn{
					func(a *Artifact) error {
						a.Validations.SetFlag(validations.FSyntax)
						return errors.New("b0rk")
					},
					func(a *Artifact) error {
						// This fn shouldn't run
						a.Validations.SetFlag(validations.FDisposable)
						a.Steps.SetFlag(validations.FMXDomainHasIP)
						return nil
					},
				},
			},
			want: Artifact{
				Validations: validations.Validations(validations.FSyntax),
				Steps:       0,
			},
			wantErr: true,
		},
		{
			name: "Testing with expired deadline",
			args: args{
				ctx: ctxExpiredDeadline,
				sequence: []stateFn{
					func(a *Artifact) error {
						a.Validations.SetFlag(validations.FSyntax)
						a.Steps.SetFlag(validations.FSyntax)
						return nil
					},
					func(a *Artifact) error {
						// This fn shouldn't run
						a.Validations.SetFlag(validations.FDisposable)
						a.Steps.SetFlag(validations.FMXDomainHasIP)
						return nil
					},
				},
			},
			want: Artifact{
				Validations: validations.Validations(validations.FSyntax),
				Steps:       validations.Steps(validations.FSyntax),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSequence(tt.args.ctx, tt.args.artifact, tt.args.sequence)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSequence() error = %v, wantErr %v", err, tt.wantErr)
				t.FailNow()
			}

			if !reflect.DeepEqual(got.Validations, tt.want.Validations) {
				t.Errorf("Validations got = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(got.Steps, tt.want.Steps) {
				t.Errorf("Steps got = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(got.email, tt.want.email) {
				t.Errorf("Email got = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(got.ctx, tt.want.ctx) {
				t.Errorf("Context got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_prependOptions(t *testing.T) {
	t.Run("testing order", func(t *testing.T) {
		// prepend options should add the last argument before the first
		fns := prependOptions([]ArtifactFn{func(artifact *Artifact) {
			artifact.Steps.SetFlag(validations.FValid)
		}}, []ArtifactFn{func(artifact *Artifact) {
			artifact.Validations.SetFlag(validations.FValid)
		}}...)

		a := Artifact{}
		fns[0](&a)
		if a.Validations == 0 {
			t.Errorf("Expected the first fn to set the validations")
		}
		if a.Steps != 0 {
			t.Errorf("Expected the first fn to NOT set the steps")
		}

		fns[1](&a)
		if !a.Validations.HasFlag(validations.FValid) || !a.Steps.HasFlag(validations.FValid) {
			t.Errorf("Expected flag to be set after both runs, instead I got: %+v", a)
		}
	})
	t.Run("nil parent", func(t *testing.T) {
		fns := prependOptions(nil, []ArtifactFn{func(artifact *Artifact) {
			artifact.Validations.SetFlag(validations.FValid)
		}}...)

		a := Artifact{}
		fns[0](&a)
		if !a.Validations.HasFlag(validations.FValid) {
			t.Errorf("Expected the validations flag to have been set, instead I got: %+v", a)
		}
	})
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
