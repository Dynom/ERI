package hitlist

import (
	"hash"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Dynom/ERI/types"
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
			Recipients: map[rcpt]struct{}{
				rcpt("john.doe"): {},
				rcpt("jane.doe"): {},
				rcpt("joan.doe"): {},
				rcpt("jake.doe"): {},
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("b"): Hit{
			Recipients: map[rcpt]struct{}{
				rcpt("john.doe"): {},
				rcpt("jane.doe"): {},
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("c"): Hit{
			Recipients: map[rcpt]struct{}{
				rcpt("john.doe"): {},
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("d"): Hit{
			Recipients: map[rcpt]struct{}{
				rcpt("john.doe"): {},
				rcpt("jane.doe"): {},
				rcpt("joan.doe"): {},
			},
			ValidUntil:       validDuration,
			ValidationResult: validVR,
		},
		Domain("e"): Hit{
			Recipients: map[rcpt]struct{}{
				rcpt("john.doe"):    {},
				rcpt("jane.doe"):    {},
				rcpt("joan.doe"):    {},
				rcpt("jake.doe"):    {},
				rcpt("winston.doe"): {},
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

// TestHitList_FunctionalAddAndReturn is a functional test, testing adding and retrieving of
func TestHitList_FunctionalAddAndReturn(t *testing.T) {

	validVR := validator.Result{

		// Validations need to be valid for a domain for this test
		Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
	}

	type args struct {
		email    string
		vr       validator.Result
		duration time.Duration
	}

	tests := []struct {
		name             string
		args             args
		wantErr          bool
		wantTotalDomains int
		wantValidDomains int
	}{
		{
			name: "basic add",
			args: args{
				email: "john.doe@example.org",
				vr:    validVR,
			},
			wantErr:          false,
			wantTotalDomains: 1,
			wantValidDomains: 1,
		},
		{
			name: "malformed add",
			args: args{
				email: "john.doe#example.org",
				vr:    validVR,
			},
			wantErr:          true,
			wantTotalDomains: 0,
			wantValidDomains: 0,
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

			if len(hl.hits) != tt.wantTotalDomains {
				t.Errorf("Expected %d domains known to HL, instead I have %d", tt.wantTotalDomains, len(hl.hits))
			}

			if vds := hl.GetValidAndUsageSortedDomains(); len(vds) != tt.wantValidDomains {
				t.Errorf("Expected %d valid domains in HL, instead I have %d", tt.wantValidDomains, len(vds))
			}
		})
	}
}

func TestHitList_AddEmailAddressDeadlineDuplicates(t *testing.T) {
	validVR := validator.Result{

		// Validations need to be valid for a domain for this test
		Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
	}

	populatedHitList := New(mockHasher{}, time.Hour*1)
	_ = populatedHitList.AddEmailAddress("john.doe@example.org", validVR) // example caseR
	_ = populatedHitList.AddEmailAddress("jane.doe@example.org", validVR)

	expect := 2
	if got := len(populatedHitList.hits[Domain("example.org")].Recipients); got != expect {
		t.Errorf("Expecting multiple recipients to be added for the same domain. Expected %d, got %d", expect, got)
	}
}

func TestHitList_AddEmailAddressDeadline(t *testing.T) {
	validVR := validator.Result{

		// Validations need to be valid for a domain for this test
		Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
	}

	populatedHitList := New(mockHasher{}, time.Hour*1)
	_ = populatedHitList.AddEmailAddress("john.doe@example.org", validVR) // example caseR
	_ = populatedHitList.AddEmailAddress("jane.doe@example.org", validVR)
	_ = populatedHitList.AddEmailAddress("alexander@example.com", validVR)
	_ = populatedHitList.AddEmailAddress("edward@example.com", validVR)

	type fields struct {
		hits Hits
		ttl  time.Duration
		lock sync.RWMutex
		h    hash.Hash
	}

	type args struct {
		emailLocal  string
		emailDomain string
		vr          validator.Result
		duration    time.Duration
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Add with future duration",
			fields: fields{
				hits: make(Hits),
				ttl:  time.Hour * 2, // Not used in this case
				lock: sync.RWMutex{},
				h:    mockHasher{},
			},
			args: args{
				emailLocal:  "john.doe",
				emailDomain: "example.org",
				vr:          validVR,
				duration:    time.Hour * 1,
			},
			wantErr: false,
		},
		{
			name: "Add with expired duration",
			fields: fields{
				hits: make(Hits),
				ttl:  time.Hour * 2, // Not used in this case
				lock: sync.RWMutex{},
				h:    mockHasher{},
			},
			args: args{
				emailLocal:  "john.doe",
				emailDomain: "example.org",
				vr:          validVR,
				duration:    time.Hour * -1,
			},
			wantErr: false,
		},
		{
			name: "Add duplicate",
			fields: fields{
				hits: populatedHitList.hits,
				ttl:  time.Hour * 2,
				lock: sync.RWMutex{},
				h:    mockHasher{},
			},
			args: args{
				emailLocal:  "john.doe",
				emailDomain: "example.org",
				vr:          validVR,
				duration:    time.Hour * -1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := &HitList{
				hits: tt.fields.hits,
				ttl:  tt.fields.ttl,
				lock: tt.fields.lock,
				h:    tt.fields.h,
			}

			email := tt.args.emailLocal + `@` + tt.args.emailDomain

			if err := hl.AddEmailAddressDeadline(email, tt.args.vr, tt.args.duration); (err != nil) != tt.wantErr {
				t.Errorf("AddEmailAddressDeadline() error = %v, wantErr %v", err, tt.wantErr)
			}

			hit, ok := hl.hits[Domain(tt.args.emailDomain)]
			if !ok {
				t.Errorf("Expected %q to be present, it's not", tt.args.emailDomain)
				return
			}

			// Adding time to now, to compensate for test execution time
			now := time.Now().Add(time.Second * 10)
			if expect := now.Add(tt.args.duration); hit.ValidUntil.After(expect) {
				t.Errorf("Expected the validity to not expire before %q, instead it was %q", expect, hit.ValidUntil)
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

func (s mockHasher) Sum(p []byte) []byte {

	// Make sure we did something, reverse the input
	var r = make([]byte, len(p))
	for i, v := range p {
		j := len(r) - 1 - i
		r[j] = v
	}

	return r
}

func (s mockHasher) Reset() {

}

func (s mockHasher) Size() int {
	return len(s.v)
}

func (s mockHasher) BlockSize() int {
	return 128
}

func TestHitList_GetValidAndUsageSortedDomains(t *testing.T) {
	validVR := validator.Result{

		// Validations need to be valid for a domain for this test
		Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
	}

	invalidVR := validator.Result{
		Validations: validations.Validations(0),
	}

	populatedFullyValidHitList := New(mockHasher{}, time.Hour*1)
	_ = populatedFullyValidHitList.AddEmailAddress("john.doe@example.org", validVR) // example case
	_ = populatedFullyValidHitList.AddEmailAddress("jane.doe@example.org", validVR)
	_ = populatedFullyValidHitList.AddEmailAddress("alexander@example.com", validVR)

	populatedHitListFaultyDomains := New(mockHasher{}, time.Hour*1)
	_ = populatedHitListFaultyDomains.AddEmailAddress("john.doe@example.or", invalidVR)
	_ = populatedHitListFaultyDomains.AddEmailAddress("alexan der@example.com", invalidVR)

	_ = populatedHitListFaultyDomains.AddEmailAddress("jane.doe@eXamplE.org", validVR)

	populatedHitListExpiredDomains := New(mockHasher{}, time.Hour*1 /* Not used for this test set */)
	_ = populatedHitListExpiredDomains.AddEmailAddressDeadline("john.doe@example.org", validVR, 0)
	_ = populatedHitListExpiredDomains.AddEmailAddressDeadline("alexander@example.com", validVR, 0)

	_ = populatedHitListExpiredDomains.AddEmailAddressDeadline("jane.doe@example.org", validVR, 0)

	type fields struct {
		hits Hits
		ttl  time.Duration
		lock sync.RWMutex
		h    hash.Hash
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "All valid domains",
			fields: fields{
				hits: populatedFullyValidHitList.hits,
				ttl:  populatedFullyValidHitList.ttl,
				lock: sync.RWMutex{},
				h:    populatedFullyValidHitList.h,
			},
			want: []string{
				"example.org",
				"example.com",
			},
		},
		{
			name: "With faulty domains",
			fields: fields{
				hits: populatedHitListFaultyDomains.hits,
				ttl:  populatedHitListFaultyDomains.ttl,
				lock: sync.RWMutex{},
				h:    populatedHitListFaultyDomains.h,
			},
			want: []string{
				"example.org",
			},
		},
		{
			name: "With expired domains",
			fields: fields{
				hits: populatedHitListExpiredDomains.hits,
				ttl:  populatedHitListExpiredDomains.ttl,
				lock: sync.RWMutex{},
				h:    populatedHitListExpiredDomains.h,
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := &HitList{
				hits: tt.fields.hits,
				ttl:  tt.fields.ttl,
				lock: tt.fields.lock,
				h:    tt.fields.h,
			}

			if got := hl.GetValidAndUsageSortedDomains(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValidAndUsageSortedDomains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHitList_AddEmailAddress(t *testing.T) {
	validVR := validator.Result{

		// Validations need to be valid for a domain for this test
		Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
	}

	hl := New(mockHasher{}, time.Hour*1)

	now := time.Now()
	_ = hl.AddEmailAddress("john.doe@example.org", validVR) // example caseR

	if vu, expected := hl.hits[Domain("example.org")].ValidUntil.Round(time.Second*1), now.Add(hl.ttl).Round(time.Second*1); !expected.Equal(vu) {
		t.Errorf("Expected the TTL to have been set with the short-hand AddEmailAddress. \nExpected %v, \ngot      %v", expected, vu)
	}
}

func TestHitList_AddDomain(t *testing.T) {
	validVR := validator.Result{

		// Validations need to be valid for a domain for this test
		Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
	}

	invalidVR := validator.Result{
		Validations: validations.Validations(0),
	}

	populatedFullyValidHitList := New(mockHasher{}, time.Hour*1)
	_ = populatedFullyValidHitList.AddDomain("example.org", validVR)

	type fields struct {
		hits Hits
		ttl  time.Duration
		h    hash.Hash
	}

	type args struct {
		d  string
		vr validator.Result
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantVR  validator.Result
	}{
		{
			name: "Unknown",
			fields: fields{
				hits: populatedFullyValidHitList.hits,
				ttl:  populatedFullyValidHitList.ttl,
				h:    populatedFullyValidHitList.h,
			},
			args: args{
				d:  "example1.com",
				vr: validVR,
			},
			wantErr: false,
			wantVR:  validVR,
		},
		{
			name: "Unknown, invalid VR",
			fields: fields{
				hits: populatedFullyValidHitList.hits,
				ttl:  populatedFullyValidHitList.ttl,
				h:    populatedFullyValidHitList.h,
			},
			args: args{
				d:  "example2.com",
				vr: invalidVR,
			},
			wantErr: false,
			wantVR:  invalidVR,
		},
		{
			name: "Known, with same VR",
			fields: fields{
				hits: populatedFullyValidHitList.hits,
				ttl:  populatedFullyValidHitList.ttl,
				h:    populatedFullyValidHitList.h,
			},
			args: args{
				d:  "example.org",
				vr: validVR,
			},
			wantErr: false,
			wantVR:  validVR,
		},
		{
			name: "Known, with different VR",
			fields: fields{
				hits: populatedFullyValidHitList.hits,
				ttl:  populatedFullyValidHitList.ttl,
				h:    populatedFullyValidHitList.h,
			},
			args: args{
				d: "example.org",
				vr: validator.Result{
					Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup | validations.FDomainHasIP),
				},
			},
			wantErr: false,
			wantVR: validator.Result{
				Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup | validations.FDomainHasIP),
			},
		},
		{
			name: "Empty input",
			fields: fields{
				hits: populatedFullyValidHitList.hits,
				ttl:  populatedFullyValidHitList.ttl,
				h:    populatedFullyValidHitList.h,
			},
			args: args{
				d:  "",
				vr: validVR,
			},
			wantErr: true,
			wantVR:  validator.Result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := &HitList{
				hits: tt.fields.hits,
				ttl:  tt.fields.ttl,
				lock: sync.RWMutex{},
				h:    tt.fields.h,
			}

			if err := hl.AddDomain(tt.args.d, tt.args.vr); (err != nil) != tt.wantErr {
				t.Errorf("AddDomain() error = %v, wantErr %v", err, tt.wantErr)
			}

			if vr := hl.hits[Domain(tt.args.d)].ValidationResult; !reflect.DeepEqual(vr, tt.wantVR) {
				t.Errorf("Expected the Validation Result to be \n%+v, instead I got \n%+v", tt.wantVR, vr)
			}
		})
	}
}

func TestHitList_Add(t *testing.T) {

	type args struct {
		parts types.EmailParts
		vr    validator.Result
	}

	tests := []struct {
		name           string
		args           args
		wantErr        bool
		domainAdded    bool
		recipientCount int
	}{
		{
			name: "Add empty local should add only domain", args: args{
				parts: types.NewEmailFromParts("", "gmail.com"),
				vr:    validator.Result{},
			},
			wantErr:        false,
			domainAdded:    true,
			recipientCount: 0,
		},
		{
			name: "Add full", args: args{
				parts: types.NewEmailFromParts("john", "gmail.com"),
				vr:    validator.Result{},
			},
			wantErr:        false,
			domainAdded:    true,
			recipientCount: 1,
		},
		{
			name: "Add local only", args: args{
				parts: types.NewEmailFromParts("john", ""),
				vr:    validator.Result{},
			},
			wantErr:        true,
			domainAdded:    false,
			recipientCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := New(mockHasher{}, time.Hour*1)
			if err := hl.Add(tt.args.parts, tt.args.vr); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			domain := Domain(tt.args.parts.Domain)
			_, exists := hl.GetDomainValidationResult(domain)
			if exists != tt.domainAdded {
				t.Errorf("Domain wasn't added, while it should've been")
			}

			if rcnt := len(hl.hits[domain].Recipients); rcnt != tt.recipientCount {
				t.Errorf("Expected %d recipients to have been added, instead I have %d", tt.recipientCount, rcnt)
			}
		})
	}
}

func TestHitList_Has(t *testing.T) {
	type args struct {
		parts types.EmailParts
	}

	tests := []struct {
		name       string
		toAdd      []types.EmailParts
		args       args
		wantDomain bool
		wantLocal  bool
	}{
		{
			name: "exact match",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.org"),
			},
			args: args{
				parts: types.NewEmailFromParts("john", "example.org"),
			},
			wantDomain: true,
			wantLocal:  true,
		},
		{
			name: "exact match, after normalisation",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("JOHN", "EXAMPLE.ORG"),
			},
			args: args{
				parts: types.NewEmailFromParts("joHn", "eXample.org"),
			},
			wantDomain: true,
			wantLocal:  true,
		},
		{
			name: "domain but not local match",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.org"),
			},
			args: args{
				parts: types.NewEmailFromParts("jane", "example.org"),
			},
			wantDomain: true,
			wantLocal:  false,
		},
		{
			name: "no match",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.com"),
			},
			args: args{
				parts: types.NewEmailFromParts("jane", "example.org"),
			},
			wantDomain: false,
			wantLocal:  false,
		},
		{
			name:  "empty domain",
			toAdd: []types.EmailParts{},
			args: args{
				parts: types.NewEmailFromParts("jane", ""),
			},
			wantDomain: false,
			wantLocal:  false,
		},
		{
			name: "empty local",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.com"),
			},
			args: args{
				parts: types.NewEmailFromParts("", "example.com"),
			},
			wantDomain: true,
			wantLocal:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := New(mockHasher{}, time.Hour*1)
			for _, p := range tt.toAdd {
				err := hl.Add(p, validator.Result{})
				if err != nil {
					t.Errorf("Failed adding parts, test setup failed.")
					return
				}
			}

			gotDomain, gotLocal := hl.Has(tt.args.parts)
			if gotDomain != tt.wantDomain {
				t.Errorf("Has() gotDomain = %v, want %v", gotDomain, tt.wantDomain)
			}

			if gotLocal != tt.wantLocal {
				t.Errorf("Has() gotLocal = %v, want %v", gotLocal, tt.wantLocal)
			}
		})
	}
}

func TestHitList_GetInternalTypes(t *testing.T) {
	tests := []struct {
		name          string
		p             types.EmailParts
		wantRecipient Recipient
		wantDomain    Domain
		wantErr       bool
	}{
		{
			name:          "All good",
			p:             types.NewEmailFromParts("john", "example.org"),
			wantRecipient: []byte("nhoj"),
			wantDomain:    "example.org",
			wantErr:       false,
		},
		{
			name:          "No domain",
			p:             types.NewEmailFromParts("john", ""),
			wantRecipient: Recipient(""),
			wantDomain:    "",
			wantErr:       true,
		},
		{
			name:          "No recipient",
			p:             types.NewEmailFromParts("", "example.org"),
			wantRecipient: Recipient(""),
			wantDomain:    "",
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := New(mockHasher{}, time.Second*1)

			gotDomain, gotRecipient, err := hl.CreateInternalTypes(tt.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateInternalTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRecipient, tt.wantRecipient) {
				t.Errorf("CreateInternalTypes() gotRecipient = %#v, want %#v", gotRecipient, tt.wantRecipient)
			}
			if gotDomain != tt.wantDomain {
				t.Errorf("CreateInternalTypes() gotDomain = %v, want %v", gotDomain, tt.wantDomain)
			}
		})
	}
}

func TestHitList_GetRecipientCount(t *testing.T) {

	tests := []struct {
		name       string
		toAdd      []types.EmailParts
		domain     Domain
		wantAmount uint64
	}{
		{
			name: "basics",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.org"),
			},
			domain:     "example.org",
			wantAmount: 1,
		},
		{
			name: "multiple",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.org"),
				types.NewEmailFromParts("jane", "example.org"),
			},
			domain:     "example.org",
			wantAmount: 2,
		},
		{
			name: "no domain match",
			toAdd: []types.EmailParts{
				types.NewEmailFromParts("john", "example.org"),
			},
			domain:     "a",
			wantAmount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hl := New(mockHasher{}, time.Second*1)
			for _, a := range tt.toAdd {
				err := hl.Add(a, validator.Result{})
				if err != nil {
					t.Errorf("Preparing test failed %s", err)
					t.FailNow()
				}
			}

			if gotAmount := hl.GetRecipientCount(tt.domain); gotAmount != tt.wantAmount {
				t.Errorf("GetRecipientCount() = %v, want %v", gotAmount, tt.wantAmount)
			}
		})
	}
}
