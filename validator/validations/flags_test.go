package validations

import (
	"math"
	"testing"
)

func TestFlag_String(t *testing.T) {
	tests := []struct {
		name string
		f    Flag
		want string
	}{
		{name: "just one", f: FValid, want: "valid"},
		{name: "some", f: FValid | FSyntax, want: "valid,syntax"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toString(t *testing.T) {
	var max Flag = 0
	for i := 0; max < math.MaxUint8; i++ {
		sub := uint8(math.Pow(2, float64(i)))
		if got := toString(Flag(sub)); got == "" {
			t.Errorf("Got empty value from toString(%08b) = this is unexpected", sub)
		}

		max += Flag(sub)
	}
}
