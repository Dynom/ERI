package erihttp

import (
	"bytes"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGetBodyFromHTTPRequest(t *testing.T) {
	const MaxBodySize = 1 << 19
	tests := []struct {
		name    string
		req     func(body []byte) *http.Request
		want    []byte
		wantErr error
	}{
		{
			wantErr: nil,
			name:    "All good",
			req: func(body []byte) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			want: []byte("{}"),
		},
		{
			wantErr: ErrMissingBody,
			name:    "Nil body",
			req: func(_ []byte) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Content-Type", "application/json")
				req.Body = nil

				return req
			},
			want: nil,
		},
		{
			wantErr: ErrBodyTooLarge,
			name:    "Too lengthy/Content-Length",
			req: func(_ []byte) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
				req.Header.Set("Content-Type", "application/json")
				req.ContentLength = math.MaxInt64
				return req
			},
			want: nil,
		},
		{
			wantErr: ErrBodyTooLarge,
			name:    "Too lengthy/Body",
			req: func(_ []byte) *http.Request {
				body := strings.Repeat("a", MaxBodySize+1)
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.ContentLength = int64(len(body) - 1)

				return req
			},
			want: nil,
		},
		{
			wantErr: ErrUnsupportedContentType,
			name:    "Content-Type/Missing",
			req: func(_ []byte) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
				req.Header.Del("Content-Type")
				return req
			},
			want: nil,
		},
		{
			wantErr: ErrUnsupportedContentType,
			name:    "Content-Type/Wrong",
			req: func(_ []byte) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
				req.Header.Set("Content-Type", "plain/text")
				return req
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req(tt.want)
			got, err := GetBodyFromHTTPRequest(req, MaxBodySize)

			if err != tt.wantErr {
				t.Errorf("GetBodyFromHTTPRequest() error = %v, wantErr %q", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBodyFromHTTPRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
