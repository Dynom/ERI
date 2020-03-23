package hitlist

import (
	"hash"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/validator/validations"
)

func TestHitList_AddDomain(t *testing.T) {
	type fields struct {
		Set  map[string]domainHit
		ttl  time.Duration
		lock sync.RWMutex
		h    hash.Hash
	}
	type args struct {
		domain string
		vr     validator.Result
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HitList{
				set:  tt.fields.Set,
				ttl:  tt.fields.ttl,
				lock: tt.fields.lock,
				h:    tt.fields.h,
			}
			if err := h.AddDomain(tt.args.domain, tt.args.vr); (err != nil) != tt.wantErr {
				t.Errorf("AddDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHitList_AddEmailAddress(t *testing.T) {
	type fields struct {
		Set  map[string]domainHit
		ttl  time.Duration
		lock sync.RWMutex
		h    hash.Hash
	}
	type args struct {
		email string
		vr    validator.Result
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HitList{
				set:  tt.fields.Set,
				ttl:  tt.fields.ttl,
				lock: tt.fields.lock,
				h:    tt.fields.h,
			}
			if err := h.AddEmailAddress(tt.args.email, tt.args.vr); (err != nil) != tt.wantErr {
				t.Errorf("AddEmailAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHitList_AddEmailAddressDeadline(t *testing.T) {
	type fields struct {
		Set  map[string]domainHit
		ttl  time.Duration
		lock sync.RWMutex
		h    hash.Hash
	}
	type args struct {
		email    string
		vr       validator.Result
		duration time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HitList{
				set:  tt.fields.Set,
				ttl:  tt.fields.ttl,
				lock: tt.fields.lock,
				h:    tt.fields.h,
			}
			if err := h.AddEmailAddressDeadline(tt.args.email, tt.args.vr, tt.args.duration); (err != nil) != tt.wantErr {
				t.Errorf("AddEmailAddressDeadline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHitList_GetValidAndUsageSortedDomains(t *testing.T) {
	type fields struct {
		Set  map[string]domainHit
		ttl  time.Duration
		lock sync.RWMutex
		h    hash.Hash
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HitList{
				set:  tt.fields.Set,
				ttl:  tt.fields.ttl,
				lock: tt.fields.lock,
				h:    tt.fields.h,
			}
			if got := h.GetValidAndUsageSortedDomains(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValidAndUsageSortedDomains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewHitList(t *testing.T) {
	type args struct {
		h   hash.Hash
		ttl time.Duration
	}
	tests := []struct {
		name string
		args args
		want *HitList
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.h, tt.args.ttl); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRCPT_String(t *testing.T) {
	tests := []struct {
		name string
		rcpt Recipient
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rcpt.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculateValidRCPTUsage(t *testing.T) {
	referenceTime := time.Date(2019, 11, 27, 5, 31, 0, 0, time.UTC)

	t.Run("testing oldest", func(t *testing.T) {
		rcpts := make(Recipients, 2)

		validA := referenceTime.Add(10 * time.Hour)
		validOldest := referenceTime.Add(1 * time.Hour)

		rcpts["john@example.org"] = Hit{
			ValidationResult: validator.Result{
				Validations: validations.Validations(validations.FValid),
				Steps:       0,
			},
			ValidUntil: validA,
		}

		rcpts["jane@example.org"] = Hit{
			ValidationResult: validator.Result{
				Validations: validations.Validations(validations.FValid),
				Steps:       0,
			},
			ValidUntil: validOldest,
		}

		gotUsage := calculateValidRCPTUsage(rcpts, referenceTime)
		if wantUsage := uint(len(rcpts)); gotUsage != wantUsage {
			t.Errorf("calculateValidRCPTUsage() gotUsage = %v, want %v", gotUsage, wantUsage)
		}
	})

	t.Run("testing usage", func(t *testing.T) {
		rcpts := make(Recipients, 3)

		want := uint(2)
		validTime := referenceTime.Add(10 * time.Hour)
		expiredTime := referenceTime.Add(-1 * time.Hour)

		rcpts["john@example.org"] = Hit{
			ValidationResult: validator.Result{
				Validations: validations.Validations(validations.FValid),
				Steps:       0,
			},
			ValidUntil: validTime,
		}

		rcpts["jane@example.org"] = Hit{
			ValidationResult: validator.Result{
				Validations: validations.Validations(validations.FValid),
				Steps:       0,
			},
			ValidUntil: validTime,
		}

		// Validity expired
		rcpts["late@example.org"] = Hit{
			ValidationResult: validator.Result{
				Validations: validations.Validations(validations.FValid),
				Steps:       0,
			},
			ValidUntil: expiredTime,
		}

		// Invalid
		rcpts["not-valid@example.org"] = Hit{
			ValidationResult: validator.Result{
				Validations: 0,
				Steps:       0,
			},
			ValidUntil: validTime,
		}

		got := calculateValidRCPTUsage(rcpts, referenceTime)
		if got != want {
			t.Errorf("calculateValidRCPTUsage() got = %v, want %v", got, want)
		}
	})
}

type FakeInt8 struct {
	Validations int8
}
type FakeInt16 struct {
	Validations int16
}
type FakeInt32 struct {
	Validations int32
}
type FakeInt64 struct {
	Validations int64
}

var bigMapInt8 map[string]FakeInt8
var bigMapInt16 map[string]FakeInt16
var bigMapInt32 map[string]FakeInt32
var bigMapInt64 map[string]FakeInt64

func BenchmarkMemoryUsage(b *testing.B) {

	const mapSize = 1000
	const keySize = 5
	const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var keys = make([]string, mapSize)
	for i := 0; i < mapSize; i++ {
		var key = make([]byte, keySize)
		for i := uint(0); i < keySize; i++ {
			key[i] = alnum[rand.Intn(len(alnum))]
		}

		keys[i] = string(key)
	}

	b.Run("int8", func(t *testing.B) {
		for j := 0; j < t.N; j++ {
			bigMapInt8 = make(map[string]FakeInt8, mapSize)
			for _, key := range keys {
				bigMapInt8[key] = FakeInt8{
					Validations: math.MaxInt8,
				}
			}
		}
	})
	b.Run("int16", func(t *testing.B) {
		for j := 0; j < t.N; j++ {
			bigMapInt16 = make(map[string]FakeInt16, mapSize)
			for _, key := range keys {
				bigMapInt16[key] = FakeInt16{
					Validations: math.MaxInt16,
				}
			}
		}
	})
	b.Run("int32", func(t *testing.B) {
		for j := 0; j < t.N; j++ {
			bigMapInt32 = make(map[string]FakeInt32, mapSize)
			for _, key := range keys {
				bigMapInt32[key] = FakeInt32{
					Validations: math.MaxInt32,
				}
			}
		}
	})
	b.Run("int64", func(t *testing.B) {
		for j := 0; j < t.N; j++ {
			bigMapInt64 = make(map[string]FakeInt64, mapSize)
			for _, key := range keys {
				bigMapInt64[key] = FakeInt64{
					Validations: math.MaxInt64,
				}
			}
		}
	})

	_ = bigMapInt8
	_ = bigMapInt16
	_ = bigMapInt32
	_ = bigMapInt64
}
