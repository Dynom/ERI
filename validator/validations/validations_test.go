package validations

import (
	"fmt"
	"math"
	"testing"
)

func TestValidations_HasFlag(t *testing.T) {
	tests := []struct {
		name string
		v    Flag
		tf   Flag
		want bool
	}{
		{want: true, name: "has flag", v: FValid, tf: FValid},
		{want: true, name: "has flag (multiple)", v: FValid | FDomainHasIP, tf: FValid},
		{want: true, name: "has flag (multiple)", v: FSyntax | FDomainHasIP, tf: FDomainHasIP},

		{name: "doesn't have flag", v: 0, tf: FValid},
		{name: "doesn't have flag", v: FDomainHasIP, tf: FValid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validations(tt.v)
			if got := v.HasFlag(tt.tf); got != tt.want {
				t.Errorf("HasFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_IsValid(t *testing.T) {
	tests := []struct {
		name string
		v    Flag
		want bool
	}{
		{want: true, name: "mega valid", v: FValid},
		{name: "default value", v: 0},
		{name: "some flags", v: FSyntax | FDomainHasIP},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validations(tt.v)
			if got := v.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_IsValidationsForValidDomain(t *testing.T) {
	tests := []struct {
		name string
		v    Flag
		want bool
	}{
		{want: true, name: "All domain flags", v: FSyntax | FHostConnect | FMXLookup | FDomainHasIP},
		{want: true, name: "Domain has IP", v: FSyntax | FDomainHasIP},
		{want: true, name: "Domain accepted connections", v: FSyntax | FHostConnect},
		{want: true, name: "DNS lookup showed domain has MX records", v: FSyntax | FMXLookup},

		// A valid flag on the domain doesn't have to mean that it's valid, without additional flags
		{name: "valid, doesn't mean valid domain", v: FValid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validations(tt.v)
			if got := v.IsValidationsForValidDomain(); got != tt.want {
				t.Errorf("IsValidationsForValidDomain() = %v, want %v (%08b)", got, tt.want, tt.v)
			}
		})
	}
}

func TestValidations_MarkAsInvalid(t *testing.T) {
	tests := []struct {
		name string
		v    Flag
	}{
		{name: "Starting as valid", v: FValid},
		{name: "Starting as invalid", v: 0},
		{name: "Various flags, invalid", v: FHostConnect | FMXLookup},
		{name: "Various flags, valid", v: FValid | FHostConnect | FMXLookup | FValidRCPT | FMXLookup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validations(tt.v)
			v.MarkAsInvalid()

			if got := v.IsValid(); got != false {
				t.Errorf("Expected v.MarkAsInvalid to always result in invalid validations %08b", v)
			}
		})
	}
}

func TestValidations_MarkAsValid(t *testing.T) {
	tests := []struct {
		name string
		v    Flag
	}{
		{name: "Starting as valid", v: FValid},
		{name: "Starting as invalid", v: 0},
		{name: "Various flags, invalid", v: FHostConnect | FMXLookup},
		{name: "Various flags, valid", v: FValid | FHostConnect | FMXLookup | FValidRCPT | FMXLookup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validations(tt.v)
			v.MarkAsValid()

			if got := v.IsValid(); got != true {
				t.Errorf("Expected v.MarkAsValid to always result in valid validations %08b", v)
			}
		})
	}
}

func TestValidations_MergeWithNext(t *testing.T) {
	tests := []struct {
		name string
		v    Flag
		new  Flag
		want Flag
	}{
		{name: "single flag", v: 0, new: FValid, want: FValid},
		{name: "multiple flags, start from 0", v: 0, new: FMXLookup | FHostConnect, want: FMXLookup | FHostConnect},
		{name: "single flag, start from FMXLookup", v: FMXLookup, new: FHostConnect, want: FMXLookup | FHostConnect},
		{name: "multiple flags, start from FMXLookup", v: FMXLookup, new: FMXLookup | FHostConnect, want: FMXLookup | FHostConnect},

		// MergeWithNext() assumes Full validations as arguments and assumes an incremental chain of validations
		// it unset's the validity of the existing validations.
		{name: "multiple flags, start from FValid", v: FValid, new: FHostConnect, want: FHostConnect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validations(tt.v)
			if got := v.MergeWithNext(Validations(tt.new)); got != Validations(tt.want) {
				t.Errorf("MergeWithNext()\n%08b (%d) got\n%08b (%d) want", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestSizeExpectation(t *testing.T) {
	var v Validations

	v = math.MaxUint8
	if v+1 != 0 {
		t.Errorf("Expected v to be uint8, which should overflow to 0, \nv     = %+v, \nv + 1 = %+v", v, v+1)
	}
}

func TestStartingFromEmptyValidations(t *testing.T) {
	var v Flag

	if v != 0 {
		t.Errorf("Expected the default value of Validations to equal 0, got: %+v", v)
	}

	t.Logf("initial       %08b", v)

	// Setting MX
	v |= FMXLookup
	t.Logf("set mx?       %08b", v)
	t.Logf("is valid?     %08b", v&FValid)

	if !Validations(v).HasFlag(FMXLookup) {
		t.Errorf("Expected v to have the FMXLookup flag set, I got: %08b", v)
	}

	if Validations(v).IsValid() {
		t.Errorf("Expected v not to be valid, I got: %08b", v)
	}

	// Marking as valid
	v |= FValid
	t.Logf("valid masked  %08b", v&FValid)
	t.Logf("is valid?     %t", v&FValid == FValid)

	if !Validations(v).IsValid() {
		t.Errorf("Expected v to be valid, I got: %08b", v)
	}

	v = 0 &^ FValid
	t.Logf("valid cleared %08b", v)
	t.Logf("is valid?     %t", v&FValid == FValid)

	if Validations(v).IsValid() {
		t.Errorf("Expected v no longer to be valid, I got: %08b", v)
	}

}

func TestValidations_Merge(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		new  Validations
		want Validations
	}{
		{
			name: "Marking valid",
			v:    Validations(0),
			new:  Validations(FValid),
			want: Validations(FValid),
		},
		{
			name: "Marking invalid",
			v:    Validations(FValid),
			new:  Validations(0),
			want: Validations(0),
		},
		{
			name: "Marking validations",
			v:    Validations(0),
			new:  Validations(FMXLookup | FSyntax | FHostConnect),
			want: Validations(FMXLookup | FSyntax | FHostConnect),
		},
		{
			name: "Appending flags, excluding valid",
			v:    Validations(FSyntax),
			new:  Validations(FMXLookup | FHostConnect),
			want: Validations(FMXLookup | FSyntax | FHostConnect),
		},
		{
			name: "Appending flags, first valid, new invalid",
			v:    Validations(FSyntax | FValid),
			new:  Validations(FMXLookup | FHostConnect),
			want: Validations(FMXLookup | FSyntax | FHostConnect),
		},
		{
			name: "Appending flags, first invalid, new valid",
			v:    Validations(FSyntax),
			new:  Validations(FMXLookup | FHostConnect | FValid),
			want: Validations(FMXLookup | FSyntax | FHostConnect | FValid),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MergeWithNext(tt.new); got != tt.want {
				t.Errorf("MergeWithNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_String(t *testing.T) {
	var v Validations

	if v.String() != "00000000" {
		t.Errorf("Expected the String method to be 0 padded")
	}
}

func TestValidations_RemoveFlag(t *testing.T) {
	type args struct {
		f Flag
	}
	tests := []struct {
		name string
		v    Validations
		args args
		want Validations
	}{
		{
			name: "test",
			v:    Validations(255),
			args: args{
				f: 255,
			},
			want: Validations(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.RemoveFlag(tt.args.f); got != tt.want {
				t.Errorf("RemoveFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

const amount = 1024 * 1024

var (
	Int8  []int8
	Int16 []int16
	Int32 []int32
	Int64 []int64
)

func BenchmarkTypeMemoryUsageInt8(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int8 = make([]int8, amount)
	}

	Int8[0] += 1
}
func BenchmarkTypeMemoryUsageInt16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int16 = make([]int16, amount)
	}

	Int16[0] += 1
}
func BenchmarkTypeMemoryUsageInt32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int32 = make([]int32, amount)
	}

	Int32[0] += 1
}
func BenchmarkTypeMemoryUsageInt64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int64 = make([]int64, amount)
	}

	Int64[0] += 1
}

func ExampleMaskTest() {
	fmt.Printf("FValid        %08b %d\n", FValid, FValid)
	fmt.Printf("FSyntax       %08b %d\n", FSyntax, FSyntax)
	fmt.Printf("FMXLookup     %08b %d\n", FMXLookup, FMXLookup)
	fmt.Printf("FDomainHasIP  %08b %d\n", FDomainHasIP, FDomainHasIP)
	fmt.Printf("FHostConnect  %08b %d\n", FHostConnect, FHostConnect)
	fmt.Printf("FValidRCPT    %08b %d\n", FValidRCPT, FValidRCPT)
	fmt.Printf("FDisposable   %08b %d\n", FDisposable, FDisposable)

	// Output:
	// FValid        00000001 1
	// FSyntax       00000010 2
	// FMXLookup     00000100 4
	// FDomainHasIP  00001000 8
	// FHostConnect  00010000 16
	// FValidRCPT    00100000 32
	// FDisposable   01000000 64
}
