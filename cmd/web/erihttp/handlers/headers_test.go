package handlers

import (
	"net/http"
	"reflect"
	"testing"
)

func TestWithHeaders(t *testing.T) {
	type args struct {
		headers http.Header
	}
	tests := []struct {
		name string
		args args
		want Middleware
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithHeaders(tt.args.headers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addHeaders(t *testing.T) {
	empty := http.Header{}

	d := http.Header{}
	d.Add("Content-Type", "application/json")

	s := http.Header{}
	s.Add("Accept-Encoding", "gzip")

	want := http.Header{}
	want.Add("Content-Type", "application/json")
	want.Add("Accept-Encoding", "gzip")

	wantDup := http.Header{}
	wantDup.Add("Content-Type", "application/json")
	wantDup.Add("Accept-Encoding", "gzip")
	wantDup.Add("Accept-Encoding", "gzip")

	type args struct {
		dst http.Header
		src http.Header
	}
	tests := []struct {
		name     string
		args     args
		expected http.Header
	}{
		{
			name: "Testing if header added",
			args: args{
				dst: d,
				src: s,
			},
			expected: want,
		},
		{
			name: "Testing with duplicate headers",
			args: args{
				dst: d,
				src: s,
			},
			expected: wantDup,
		},
		{
			name: "Testing with empty headers",
			args: args{
				dst: empty,
				src: empty,
			},
			expected: empty,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addHeaders(tt.args.dst, tt.args.src)
			if !reflect.DeepEqual(tt.args.dst, tt.expected) {
				t.Errorf("AddHeaders() dst = %v, src = %v", tt.args.dst, tt.args.src)
			}
		})
	}
}
