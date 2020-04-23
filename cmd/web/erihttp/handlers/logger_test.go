package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	testLog "github.com/sirupsen/logrus/hooks/test"
)

func TestWithRequestLogger(t *testing.T) {
	logger, hook := testLog.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("request ID increments", func(t *testing.T) {
		hook.Reset()

		h := WithRequestLogger(logger)
		handler := h(http.NewServeMux())

		// Request 1
		{
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			handler.ServeHTTP(rec, req)
		}

		// Request 2
		{
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			handler.ServeHTTP(rec, req)
		}

		rid := hook.Entries[len(hook.Entries)-1].Data["request_id"]
		if want := "2"; rid != want {
			t.Errorf("Expected the request_id to increment with every request. got %s, want %s", rid, want)
		}
	})
}
