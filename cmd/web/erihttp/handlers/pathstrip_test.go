package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	testLog "github.com/sirupsen/logrus/hooks/test"
)

func Test_normalizeSlashes(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "All OK", path: "/eri", want: "/eri"},
		{name: "Fixing Suffix", path: "/eri/", want: "/eri"},
		{name: "Fixing Prefix", path: "eri/", want: "/eri"},
		{name: "Fixing Both", path: "eri", want: "/eri"},
	}

	t.Run("Logs", func(t *testing.T) {
		t.Run("Prefix", func(t *testing.T) {
			logger, hook := testLog.NewNullLogger()
			_ = normalizeSlashes(logger, "foo")

			if len(hook.Entries) != 1 {
				t.Errorf("Expected a log message, instead I got %d %+v", len(hook.Entries), hook.Entries)
				return
			}

			if hook.Entries[0].Level != logrus.WarnLevel {
				t.Errorf("Expected warning level messages")
			}
		})

		t.Run("Suffix", func(t *testing.T) {
			logger, hook := testLog.NewNullLogger()
			_ = normalizeSlashes(logger, "/foo/")

			if len(hook.Entries) != 1 {
				t.Errorf("Expected a log message, instead I got %d %+v", len(hook.Entries), hook.Entries)
				return
			}

			if hook.Entries[0].Level != logrus.WarnLevel {
				t.Errorf("Expected warning level messages")
			}
		})
	})

	logger, _ := testLog.NewNullLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSlashes(logger, tt.path); got != tt.want {
				t.Errorf("normalizeSlashes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithPathStrip(t *testing.T) {
	tests := []struct {
		stripPath   string
		requestPath string
		wantPath    string
	}{
		{stripPath: "/", requestPath: "/foo", wantPath: "/foo"},
		{stripPath: "/eri", requestPath: "/eri/foo", wantPath: "/foo"},
		{stripPath: "/eri/", requestPath: "/eri/foo", wantPath: "/foo"},
		{stripPath: "/eri/", requestPath: "/eri/foo/", wantPath: "/foo/"},

		// Non matching request path
		{stripPath: "/eri", requestPath: "/foo", wantPath: "/foo"},

		// strip path should be prepended with a '/'
		{stripPath: "eri/", requestPath: "/eri/foo/", wantPath: "/foo/"},

		// Only left-hand matching is what we want
		{stripPath: "/foo", requestPath: "/eri/foo", wantPath: "/eri/foo"},
	}

	for _, tt := range tests {
		t.Run("Testing path "+tt.stripPath, func(t *testing.T) {
			mux := http.NewServeMux()

			logger, _ := testLog.NewNullLogger()
			h := WithPathStrip(logger, tt.stripPath)

			// Creating a mock response and request object
			mockResponse := httptest.NewRecorder()
			mockRequest := httptest.NewRequest(http.MethodPost, tt.requestPath, strings.NewReader(""))

			h(mux).ServeHTTP(mockResponse, mockRequest)

			if got := mockRequest.URL.Path; got != tt.wantPath {
				t.Errorf("WithPathStrip() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}
