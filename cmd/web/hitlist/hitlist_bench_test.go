package hitlist

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/minio/highwayhash"
)

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

var (
	bigMapInt8  map[string]FakeInt8
	bigMapInt16 map[string]FakeInt16
	bigMapInt32 map[string]FakeInt32
	bigMapInt64 map[string]FakeInt64
)

func BenchmarkMemoryUsage(b *testing.B) {
	const mapSize = 1000
	const keySize = 5
	const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	keys := make([]string, mapSize)
	for i := 0; i < mapSize; i++ {
		key := make([]byte, keySize)
		for i := uint(0); i < keySize; i++ {
			key[i] = alnum[rand.Intn(len(alnum))]
		}

		keys[i] = string(key)
	}

	b.ResetTimer()
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

func BenchmarkHitlistHas(b *testing.B) {
	h, err := highwayhash.New128([]byte("00000000000000000000000000000000"))
	if err != nil {
		b.Errorf("Unable to create our hash.Hash %s", err)
		return
	}

	hl := New(mockHasher{}, time.Second*1)

	domains := []string{
		"kwlwyboeei", "rasuesvqky", "lvtdvnorpe", "jyzbmzhhgt", "azuhmpiwzv", "vlefllcgkn", "cxwxgxnczu", "cnqjdfdfpf",
		"odlxcokvva", "sbtnbohdqh", "kiynuiqtyu", "mzqjubnwvc", "ydutxgqbms", "psshkrpbtp", "wlcybzfnkj", "rsqsuebaes",
		"ulxxqospvh", "wusaaihguc", "xoorcshnee", "toqaxnfrlc",
	}

	local := []string{
		"dlqvbsdzgk", "qseknaftgb", "qwyjxmvjnn", "anspxtshqh", "spemdcjsqd", "xfsefutgrs", "iezvefinnw", "rjctjlurny",
		"ebihzruhnz", "hxvmfrxjgz", "gctysnzeoh", "suywpgfuqf", "bgrrkrfliy", "grvdyaxrzu", "ltflpxhwnv", "bqfvpupkvt",
		"bfpwstdrkd", "wntdeedlxx", "nbfiqwqans", "vpdyiolhfk",
	}

	for _, d := range domains {
		for _, l := range local {
			_ = hl.Add(types.NewEmailFromParts(l, d), validator.Result{})
		}
	}

	ExpectNonExisting := types.NewEmailFromParts("john", "example.org")
	ExpectExisting := types.NewEmailFromParts(local[0], domains[0])

	var t1l, t1d, t2l, t2d bool

	b.ResetTimer()
	b.SetParallelism(1)
	b.Run("Bench nonexisting, mock hasher", func(b *testing.B) {
		b.ReportAllocs()
		hl.h = mockHasher{}
		for i := 0; i < b.N; i++ {
			t1l, t1d = hl.Has(ExpectNonExisting)
		}
	})

	b.Run("Bench existing, mock hasher", func(b *testing.B) {
		b.ReportAllocs()
		hl.h = mockHasher{}
		for i := 0; i < b.N; i++ {
			t2l, t2d = hl.Has(ExpectExisting)
		}
	})

	b.Run("Bench nonexisting", func(b *testing.B) {
		b.ReportAllocs()
		hl.h = h
		for i := 0; i < b.N; i++ {
			t1l, t1d = hl.Has(ExpectNonExisting)
		}
	})

	b.Run("Bench existing", func(b *testing.B) {
		b.ReportAllocs()
		hl.h = h
		for i := 0; i < b.N; i++ {
			t2l, t2d = hl.Has(ExpectExisting)
		}
	})

	_ = t1l && t1d && t2l && t2d
}

func BenchmarkLenOrEqual(b *testing.B) {
	input := []byte("raboof")
	var refs [][]byte
	for _, v := range []string{
		"foo", "foob", "fooba", "foobar", "foobarf", "foobarfo",
	} {
		refs = append(refs, []byte(v))
	}

	var t1, t2, t3 uint64
	b.Run("just equal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, v := range refs {
				if bytes.Equal(v, input) {
					t1++
				}
			}
		}
	})

	b.Run("len before equal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, v := range refs {
				if len(v) == len(input) && bytes.Equal(v, input) {
					t2++
				}
			}
		}
	})

	b.Run("len, pos and equal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c := input[len(input)-1]
			for _, v := range refs {
				if len(v) == len(input) && v[len(v)-1] == c && bytes.Equal(v, input) {
					t3++
				}
			}
		}
	})

	_ = t1
	_ = t2
	_ = t3
}
