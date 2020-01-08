package validations

import (
	"fmt"
	"math"
	"testing"
)

func TestValidations_HasFlag(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		tf   Validations
		want bool
	}{
		{want: true, name: "has flag", v: VFValid, tf: VFValid},
		{want: true, name: "has flag (multiple)", v: VFValid | VFDomainHasIP, tf: VFValid},
		{want: true, name: "has flag (multiple)", v: VFSyntax | VFDomainHasIP, tf: VFDomainHasIP},

		{name: "doesn't have flag", v: 0, tf: VFValid},
		{name: "doesn't have flag", v: VFDomainHasIP, tf: VFValid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.HasFlag(tt.tf); got != tt.want {
				t.Errorf("HasFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_IsValid(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		want bool
	}{
		{want: true, name: "mega valid", v: VFValid},
		{name: "default value", v: 0},
		{name: "some flags", v: VFSyntax | VFDomainHasIP},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_IsValidationsForValidDomain(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		want bool
	}{
		{want: true, name: "All domain flags", v: VFSyntax | VFHostConnect | VFMXLookup | VFDomainHasIP},
		{want: true, name: "Domain has IP", v: VFSyntax | VFDomainHasIP},
		{want: true, name: "Domain accepted connections", v: VFSyntax | VFHostConnect},
		{want: true, name: "DNS lookup showed domain has MX records", v: VFSyntax | VFMXLookup},

		// A valid flag on the domain doesn't have to mean that it's valid, without additional flags
		{name: "valid, doesn't mean valid domain", v: VFValid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.IsValidationsForValidDomain(); got != tt.want {
				t.Errorf("IsValidationsForValidDomain() = %v, want %v (%08b)", got, tt.want, tt.v)
			}
		})
	}
}

func TestValidations_MarkAsInvalid(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
	}{
		{name: "Starting as valid", v: VFValid},
		{name: "Starting as invalid", v: 0},
		{name: "Various flags, invalid", v: VFHostConnect | VFMXLookup},
		{name: "Various flags, valid", v: VFValid | VFHostConnect | VFMXLookup | VFValidRCPT | VFMXLookup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.v
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
		v    Validations
	}{
		{name: "Starting as valid", v: VFValid},
		{name: "Starting as invalid", v: 0},
		{name: "Various flags, invalid", v: VFHostConnect | VFMXLookup},
		{name: "Various flags, valid", v: VFValid | VFHostConnect | VFMXLookup | VFValidRCPT | VFMXLookup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.v
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
		v    Validations
		new  Validations
		want Validations
	}{
		{name: "single flag", v: 0, new: VFValid, want: VFValid},
		{name: "multiple flags, start from 0", v: 0, new: VFMXLookup | VFHostConnect, want: VFMXLookup | VFHostConnect},
		{name: "single flag, start from VFMXLookup", v: VFMXLookup, new: VFHostConnect, want: VFMXLookup | VFHostConnect},
		{name: "multiple flags, start from VFMXLookup", v: VFMXLookup, new: VFMXLookup | VFHostConnect, want: VFMXLookup | VFHostConnect},

		// MergeWithNext() assumes Full validations as arguments and assumes an incremental chain of validations
		// it unset's the validity of the existing validations.
		{name: "multiple flags, start from VFValid", v: VFValid, new: VFHostConnect, want: VFHostConnect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MergeWithNext(tt.new); got != tt.want {
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
	var v Validations

	if v != 0 {
		t.Errorf("Expected the default value of Validations to equal 0, got: %+v", v)
	}

	t.Logf("initial       %08b", v)

	// Setting MX
	v |= VFMXLookup
	t.Logf("set mx?       %08b", v)
	t.Logf("is valid?     %08b", v&VFValid)

	if !v.HasFlag(VFMXLookup) {
		t.Errorf("Expected v to have the VFMXLookup flag set, I got: %08b", v)
	}

	if v.IsValid() {
		t.Errorf("Expected v not to be valid, I got: %08b", v)
	}

	// Marking as valid
	v |= VFValid
	t.Logf("valid masked  %08b", v&VFValid)
	t.Logf("is valid?     %t", v&VFValid == VFValid)

	if !v.IsValid() {
		t.Errorf("Expected v to be valid, I got: %08b", v)
	}

	v = 0 &^ VFValid
	t.Logf("valid cleared %08b", v)
	t.Logf("is valid?     %t", v&VFValid == VFValid)

	if v.IsValid() {
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
			new:  VFValid,
			want: VFValid,
		},
		{
			name: "Marking invalid",
			v:    VFValid,
			new:  Validations(0),
			want: Validations(0),
		},
		{
			name: "Marking validations",
			v:    Validations(0),
			new:  VFMXLookup | VFSyntax | VFHostConnect,
			want: VFMXLookup | VFSyntax | VFHostConnect,
		},
		{
			name: "Appending flags, excluding valid",
			v:    VFSyntax,
			new:  VFMXLookup | VFHostConnect,
			want: VFMXLookup | VFSyntax | VFHostConnect,
		},
		{
			name: "Appending flags, first valid, new invalid",
			v:    VFSyntax | VFValid,
			new:  VFMXLookup | VFHostConnect,
			want: VFMXLookup | VFSyntax | VFHostConnect,
		},
		{
			name: "Appending flags, first invalid, new valid",
			v:    VFSyntax,
			new:  VFMXLookup | VFHostConnect | VFValid,
			want: VFMXLookup | VFSyntax | VFHostConnect | VFValid,
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

func TestString(t *testing.T) {
	var v Validations

	if v.String() != "00000000" {
		t.Errorf("Expected the String method to be 0 padded")
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
	fmt.Printf("VFValid        %08b %d\n", VFValid, VFValid)
	fmt.Printf("VFSyntax       %08b %d\n", VFSyntax, VFSyntax)
	fmt.Printf("VFMXLookup     %08b %d\n", VFMXLookup, VFMXLookup)
	fmt.Printf("VFDomainHasIP  %08b %d\n", VFDomainHasIP, VFDomainHasIP)
	fmt.Printf("VFHostConnect  %08b %d\n", VFHostConnect, VFHostConnect)
	fmt.Printf("VFValidRCPT    %08b %d\n", VFValidRCPT, VFValidRCPT)
	fmt.Printf("VFDisposable   %08b %d\n", VFDisposable, VFDisposable)

	// Output:
	// VFValid        00000001 1
	// VFSyntax       00000010 2
	// VFMXLookup     00000100 4
	// VFDomainHasIP  00001000 8
	// VFHostConnect  00010000 16
	// VFValidRCPT    00100000 32
	// VFDisposable   01000000 64
}
