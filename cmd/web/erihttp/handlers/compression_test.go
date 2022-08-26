package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithGzipHandler(t *testing.T) {
	tests := []struct {
		name               string
		use                Middleware
		wantVary           bool
		wantCompressedBody bool
		requestBody        string
	}{
		{
			name:               "With",
			wantVary:           true,
			wantCompressedBody: true,
			requestBody:        strings.Repeat("t", mtuSize+1),
			use:                WithGzipHandler(),
		},
		{
			name:               "Without",
			wantVary:           false,
			wantCompressedBody: false,
			requestBody:        strings.Repeat("t", mtuSize+1),
			use: func(handler http.Handler) http.Handler {
				// A noop handler
				return handler
			},
		},
		{
			name:               "With, small response",
			wantVary:           true,
			wantCompressedBody: false,
			requestBody:        strings.Repeat("t", mtuSize-1),
			use:                WithGzipHandler(),
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("io.ReadAll(r.Body) Setting up the test failed %s", err)
			t.FailNow()
		}

		_, err = w.Write(b)
		if err != nil {
			t.Errorf("w.Write(b) Setting up the test failed %s", err)
			t.FailNow()
		}
	})

	// Creating a custom HTTP client, with implicit (de-)compression disabled
	// This means that when explicitly setting the Accept-Encoding header, the body comes back as a compressed string
	c := http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewServer(tt.use(mux))
			defer server.Close()

			request, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(tt.requestBody))
			if err != nil {
				t.Errorf("http.NewRequest() Setting up the test failed %s", err)
				t.FailNow()
			}

			// Setting the header, requesting the GZip compressor to kick in
			request.Header.Set("Accept-Encoding", "gzip")

			// Performing the request to our server
			res, err := c.Do(request)
			if err != nil {
				t.Errorf("c.Do(request) Setting up the test failed %s", err)
				t.FailNow()
			}

			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("io.ReadAll(res.Body) Setting up the test failed %s", err)
				t.FailNow()
			}

			vary := res.Header.Get("Vary")
			if gotVary := vary != ""; gotVary != tt.wantVary {
				t.Errorf("Expected res.Header.Get(\"Vary\") to be %t, instead I got %t (got: %q)", tt.wantVary, gotVary, vary)
			}

			if tt.wantCompressedBody == (tt.requestBody == string(b)) {
				t.Errorf("tt.wantCompressedBody = %t. The response body didn't meet the compress/uncompressed expectation", tt.wantCompressedBody)
			}
		})
	}
}
