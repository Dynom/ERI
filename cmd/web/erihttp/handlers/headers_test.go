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
		want HandlerWrapper
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
	type args struct {
		dst http.Header
		src http.Header
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
