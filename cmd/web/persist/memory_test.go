package persist

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/testutil"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
)

func TestStorage_Range(t *testing.T) {
	list := hitlist.New(&testutil.MockHasher{}, time.Second*1)

	type testDataS struct {
		parts types.EmailParts
		vr    validator.Result
	}

	testData := []testDataS{
		{
			parts: types.NewEmailFromParts("john", "example.org"),
			vr: validator.Result{
				Validations: validations.Validations(validations.FSyntax),
				Steps:       validations.Steps(validations.FSyntax | validations.FMXLookup),
			},
		}, {
			parts: types.NewEmailFromParts("jane", "example.org"),
			vr: validator.Result{
				Validations: validations.Validations(validations.FSyntax | validations.FMXLookup),
				Steps:       validations.Steps(validations.FSyntax | validations.FMXLookup),
			},
		},
	}

	t.Run("Range/All", func(t *testing.T) {
		ctx := context.Background()
		s := NewMemory()
		defer s.Close()

		for _, td := range testData {

			domain, recipient, err := list.CreateInternalTypes(td.parts)
			if err != nil {
				t.Errorf("list.CreateInternalTypes() Unexpected error while setting up test %s", err)
				t.FailNow()
			}

			err = s.Store(ctx, domain, recipient, td.vr)
			if err != nil {
				t.Errorf("s.Store() Unexpected error while setting up test %s", err)
				t.FailNow()
			}

			_ = s.Range(ctx, func(d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error {
				var match bool
				for _, td := range testData {
					domain, recipient, err := list.CreateInternalTypes(td.parts)
					if err == nil && domain == d && bytes.Equal(recipient, r) {
						match = true
						break
					}
				}

				if !match {
					t.Errorf("s.Range() didn't match the state. Got %q, %q expected %q, %q", d, r, domain, recipient)
				}

				return nil
			})
		}
	})

	t.Run("Range/Abort", func(t *testing.T) {
		ctx := context.Background()
		s := NewMemory()
		defer s.Close()

		for _, td := range testData {

			domain, recipient, err := list.CreateInternalTypes(td.parts)
			if err != nil {
				t.Errorf("list.CreateInternalTypes() Unexpected error while setting up test %s", err)
				t.FailNow()
			}

			err = s.Store(ctx, domain, recipient, td.vr)
			if err != nil {
				t.Errorf("s.Store() Unexpected error while setting up test %s", err)
				t.FailNow()
			}

			var collected uint
			const want = 1
			_ = s.Range(ctx, func(domain hitlist.Domain, recipient hitlist.Recipient, vr validator.Result) error {
				collected++
				return errors.New("") // Range should cancel, when a CB returns a non-nil error
			})

			if collected != want {
				t.Errorf("Expected Range to stop after %d callbacks, instead %d were invoked", want, collected)
			}
		}
	})

	t.Run("Range/Bad Key", func(t *testing.T) {
		ctx := context.Background()
		s := NewMemory().(*Memory)
		defer s.Close()

		// Preparing data with a bad key (and value)
		s.m.Store("foo", "bar")

		var collected uint
		const want = 0
		_ = s.Range(ctx, func(domain hitlist.Domain, recipient hitlist.Recipient, vr validator.Result) error {
			collected++
			return nil
		})

		if collected != want {
			t.Errorf("Expected no CB call with bad key. Expected %d, got %d", want, collected)
		}
	})

	t.Run("Range/Bad Value", func(t *testing.T) {
		ctx := context.Background()

		s := NewMemory().(*Memory)
		defer s.Close()

		// Preparing data with a good key, and with a bad vr type.
		s.m.Store("john@example.org", "bar")

		var collected uint
		const want = 0
		_ = s.Range(ctx, func(domain hitlist.Domain, recipient hitlist.Recipient, vr validator.Result) error {
			collected++
			return nil
		})

		if collected != want {
			t.Errorf("Expected no CB call with bad key. Expected %d, got %d", want, collected)
		}
	})
}

func TestStorage_Store(t *testing.T) {
	list := hitlist.New(&testutil.MockHasher{}, time.Second*1)

	type args struct {
		ctx   context.Context
		parts types.EmailParts
	}

	ctxNormal := context.Background()
	ctxCanceled, ctxCancel := context.WithCancel(ctxNormal)
	ctxCancel()

	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantStored bool
	}{
		{
			name: "Basic store",
			args: args{
				ctx:   ctxNormal,
				parts: types.NewEmailFromParts("john", "example.org"),
			},
			wantErr:    false,
			wantStored: true,
		},
		{
			name: "Canceled context",
			args: args{
				ctx:   ctxCanceled,
				parts: types.NewEmailFromParts("john", "example.org"),
			},
			wantErr:    true,
			wantStored: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemory()
			defer s.Close()

			vr := validator.Result{
				Validations: validations.Validations(validations.FSyntax | validations.FMXLookup),
				Steps:       validations.Steps(validations.FSyntax | validations.FMXLookup),
			}

			domain, recipient, err := list.CreateInternalTypes(tt.args.parts)
			if err != nil {
				t.Errorf("list.CreateInternalTypes() Unexpected error while setting up test %s", err)
				t.FailNow()
			}

			if err := s.Store(tt.args.ctx, domain, recipient, vr); (err != nil) != tt.wantErr {
				t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
			}

			var collected uint
			_ = s.Range(context.Background(), func(domain hitlist.Domain, recipient hitlist.Recipient, vr validator.Result) error {
				collected++
				return nil
			})

			if got := collected > 0; tt.wantStored != got {
				t.Errorf("Expected the items to have been stored, want %t, got %t ", tt.wantStored, got)
			}
		})
	}
}

func TestStoreAndRetrieve(t *testing.T) {
	ctx := context.Background()

	var domain hitlist.Domain = "example.org"
	var recipient = hitlist.Recipient("jane")

	s := NewMemory()
	err := s.Store(ctx, domain, recipient, validator.Result{})
	if err != nil {
		t.Errorf("Test setup failed %s", err)
		t.FailNow()
	}

	_ = s.Range(ctx, func(d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error {
		if d != domain {
			t.Errorf("s.Range() Expected %s, got %s", domain, d)
		}

		if !bytes.Equal(r, recipient) {
			t.Errorf("s.Range() Expected %s, got %s", recipient, r)
		}

		return nil
	})
}
