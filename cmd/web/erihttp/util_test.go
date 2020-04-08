package erihttp

import (
	"net/http"
	"reflect"
	"testing"
)

func TestGetBodyFromHTTPRequest(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBodyFromHTTPRequest(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBodyFromHTTPRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBodyFromHTTPRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
