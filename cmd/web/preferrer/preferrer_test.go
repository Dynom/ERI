package preferrer

import (
	"reflect"
	"testing"

	"github.com/Dynom/ERI/types"
)

func TestNew(t *testing.T) {
	type args struct {
		mapping Mapping
	}

	tests := []struct {
		name string
		args args
		want *Preferrer
	}{
		{name: "nil map", args: args{mapping: nil}, want: &Preferrer{}},
		{name: "populated 1", args: args{mapping: Mapping{"a": "b"}}, want: &Preferrer{m: Mapping{"a": "b"}}},
		{name: "populated N", args: args{mapping: Mapping{"a": "b", "b": "c"}}, want: &Preferrer{m: Mapping{"a": "b", "b": "c"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.mapping); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreferrer_HasPreferred(t *testing.T) {
	tests := []struct {
		name  string
		m     Mapping
		parts types.EmailParts
		want  string
		has   bool
	}{
		{name: "nil map", m: nil, parts: types.NewEmailFromParts("john.doe", "example.org"), want: "example.org", has: false},
		{name: "match", m: Mapping{"example.com": "example.org"}, parts: types.NewEmailFromParts("john.doe", "example.com"), want: "example.org", has: true},
		{name: "no match", m: Mapping{"a": "b"}, parts: types.NewEmailFromParts("john.doe", "example.org"), want: "example.org", has: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preferrer{
				m: tt.m,
			}

			got, has := p.HasPreferred(tt.parts)
			if got != tt.want || has != tt.has {
				t.Errorf("HasPreferred() got = %v, %t; want %v, %t", got, has, tt.want, tt.has)
			}
		})
	}
}
