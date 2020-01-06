package hitlist

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/Dynom/ERI/validator/validations"
)

func Test_calculateValidRCPTUsage(t *testing.T) {
	referenceTime := time.Date(2019, 11, 27, 5, 31, 0, 0, time.UTC)

	t.Run("testing oldest", func(t *testing.T) {
		rcpts := make(Recipients, 2)

		validA := referenceTime.Add(10 * time.Hour)
		validOldest := referenceTime.Add(1 * time.Hour)

		rcpts["john@example.org"] = Hit{
			Validations: validations.VFValid,
			ValidUntil:  validA,
		}

		rcpts["jane@example.org"] = Hit{
			Validations: validations.VFValid,
			ValidUntil:  validOldest,
		}

		gotUsage, gotOldest := calculateValidRCPTUsage(rcpts, referenceTime)
		if wantUsage := uint(len(rcpts)); gotUsage != wantUsage {
			t.Errorf("calculateValidRCPTUsage() gotUsage = %v, want %v", gotUsage, wantUsage)
		}

		if wantOldest := validOldest; !validOldest.Equal(wantOldest) {
			t.Errorf("calculateValidRCPTUsage() oldest %v isn't the oldest %v", gotOldest, wantOldest)
		}
	})

	t.Run("testing usage", func(t *testing.T) {
		rcpts := make(Recipients, 3)

		want := uint(2)
		validTime := referenceTime.Add(10 * time.Hour)
		expiredTime := referenceTime.Add(-1 * time.Hour)

		rcpts["john@example.org"] = Hit{
			Validations: validations.VFValid,
			ValidUntil:  validTime,
		}

		rcpts["jane@example.org"] = Hit{
			Validations: validations.VFValid,
			ValidUntil:  validTime,
		}

		// Validity expired
		rcpts["late@example.org"] = Hit{
			Validations: validations.VFValid,
			ValidUntil:  expiredTime,
		}

		// Invalid
		rcpts["not-valid@example.org"] = Hit{
			Validations: 0,
			ValidUntil:  validTime,
		}

		got, oldest := calculateValidRCPTUsage(rcpts, referenceTime)
		if got != want {
			t.Errorf("calculateValidRCPTUsage() got = %v, want %v", got, want)
		}

		if oldest != validTime {
			t.Errorf("calculateValidRCPTUsage() got = %v, want %v", oldest, validTime)
			for _, rcpt := range rcpts {
				t.Logf("%+v", rcpt)
			}
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
