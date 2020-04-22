package test

import (
	"reflect"
	"testing"
)

func TestMockHasherReverse_Sum(t *testing.T) {
	type args struct {
		p []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := MockHasherReverse{
				MockHasher: tt.fields.MockHasher,
			}
			if got := s.Sum(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sum() = %v, want %v", got, tt.want)
			}
		})
	}
}
