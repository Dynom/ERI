package handlers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestWithHeaders(t *testing.T) {
	multiHeaders := http.Header{}
	multiHeaders.Add("X-Test", "a")
	multiHeaders.Add("X-Test", "b")

	standardHeaders := http.Header{}
	standardHeaders.Add("X-Version", "v1.0.1")

	type args struct {
		headers http.Header
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Duplicate header test",
			args: args{
				headers: multiHeaders,
			},
		},
		{
			name: "Standard headers",
			args: args{
				headers: standardHeaders,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			m := WithHeaders(tt.args.headers)

			mockRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

			// A header set by our fictive app
			mockResponse := httptest.NewRecorder()
			mockResponse.Header().Add("X-Some-Header", "foo")

			// Calling our Middleware
			m(mux).ServeHTTP(mockResponse, mockRequest)

			// Testing that our fictive app's header got set
			if got := mockResponse.Header().Get("X-Some-Header"); got != "foo" {
				t.Errorf("Expected that our fictive app header got set to value %q", got)
			}

			// Extracted the known headers from the response.
			var got = http.Header{}
			h := mockResponse.Header()
			for key := range h {
				for _, v := range h.Values(key) {
					if _, ok := tt.args.headers[key]; ok {
						got.Add(key, v)
					}
				}
			}

			// Comparing to see if the headers we expect, are still present
			if !reflect.DeepEqual(got, tt.args.headers) {
				t.Errorf("WithHeaders() = %+v, want %+v", got, tt.args)
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
