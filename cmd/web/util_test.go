package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	testLog "github.com/sirupsen/logrus/hooks/test"
)

func Test_writeErrorJSONResponse(t *testing.T) {
	t.Run("unable to write", func(t *testing.T) {
		logger, hook := testLog.NewNullLogger()
		writer := &brokenResponseWriter{ResponseWriter: httptest.NewRecorder()}
		writer.writeErr = fmt.Errorf("boop")
		writeErrorJSONResponse(logger, writer, &erihttp.AutoCompleteResponse{})

		if hook.LastEntry().Message != "Failed to write response" {
			t.Errorf("Expected an error log")
		}
	})
}

type brokenResponseWriter struct {
	http.ResponseWriter
	writeErr error
}

func (b *brokenResponseWriter) Write(bytes []byte) (int, error) {
	if b.writeErr == nil {
		return b.ResponseWriter.Write(bytes)
	}

	return len(bytes), b.writeErr
}
