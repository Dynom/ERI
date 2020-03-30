package hitlist

import (
	"reflect"
	"testing"
	"time"

	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
)

func Test_getValidDomains(t *testing.T) {
	validDuration := time.Now().Add(1 * time.Hour)
	validFlags := validations.FValid | validations.FSyntax | validations.FMXLookup | validations.FDomainHasIP
	validVR := validator.Result{
		Validations: validations.Validations(validFlags),
		Steps:       validations.Steps(validFlags),
	}

	allValidHits := Hits{
		Domain("a"): Hit{
			Recipients: []Recipient{
				[]byte("john.doe"),
				[]byte("jane.doe"),
				[]byte("joan.doe"),
				[]byte("jake.doe"),
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("b"): Hit{
			Recipients: []Recipient{
				[]byte("john.doe"),
				[]byte("jane.doe"),
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("c"): Hit{
			Recipients: []Recipient{
				[]byte("john.doe"),
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("d"): Hit{
			Recipients: []Recipient{
				[]byte("john.doe"),
				[]byte("jane.doe"),
				[]byte("joan.doe"),
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("e"): Hit{
			Recipients: []Recipient{
				[]byte("john.doe"),
				[]byte("jane.doe"),
				[]byte("joan.doe"),
				[]byte("jake.doe"),
				[]byte("winston.doe"),
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
	}

	tests := []struct {
		name string
		hits Hits
		want []string
	}{
		{name: "All valid", hits: allValidHits, want: []string{"e", "a", "d", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getValidDomains(tt.hits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getValidDomains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHitList_AddEmailAddressDeadline(t *testing.T) {

	validVR := validator.Result{
		Validations: validations.Validations(validations.FValid | validations.FSyntax),
	}

	type args struct {
		email    string
		vr       validator.Result
		duration time.Duration
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantDomains int
	}{
		{
			name: "basic add",
			args: args{
				email: "john.doe@example.org",
				vr:    validVR,
			},
			wantErr:     false,
			wantDomains: 1,
		},
		{
			name: "malformed add",
			args: args{
				email: "john.doe#example.org",
				vr:    validVR,
			},
			wantErr:     true,
			wantDomains: 0,
		},
	}

	ttl := time.Hour * 1
	h := mockHasher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := New(h, ttl)
			if err := hl.AddEmailAddressDeadline(tt.args.email, tt.args.vr, ttl); (err != nil) != tt.wantErr {
				t.Errorf("AddEmailAddressDeadline() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(hl.hits) != tt.wantDomains {
				t.Errorf("Expected %d domains in HL, instead I have %d", tt.wantDomains, len(hl.hits))
			}
		})
	}
}

type mockHasher struct {
	v []byte
}

func (s mockHasher) Write(p []byte) (int, error) {
	s.v = p
	return len(p), nil
}

func (s mockHasher) Sum(_ []byte) []byte {
	return s.v
}

func (s mockHasher) Reset() {

}

func (s mockHasher) Size() int {
	return len(s.v)
}

func (s mockHasher) BlockSize() int {
	return 128
}
