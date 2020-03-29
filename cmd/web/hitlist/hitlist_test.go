package hitlist

import (
	"math"
	"math/rand"
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
