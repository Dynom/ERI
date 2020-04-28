package testutil

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
		{name: "simple reverse", args: args{p: []byte("foo")}, want: []byte("oof")},
		{name: "empty", args: args{p: []byte{}}, want: []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := MockHasherReverse{}
			if got := s.Sum(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockHasher_Sum(t *testing.T) {
	type args struct {
		p []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "identical output", args: args{p: []byte("foo")}, want: []byte("foo")},
		{name: "empty", args: args{p: []byte{}}, want: []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := MockHasher{}
			if got := s.Sum(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sum() = %v, want %v", got, tt.want)
			}
		})
	}
}
