package handlers

import (
	"net/http"
)

type Middleware func(handler http.Handler) http.Handler

type CustomResponseWriter struct {
	http.ResponseWriter
	Status       int
	BytesWritten int
}

func NewCustomResponseWriter(w http.ResponseWriter) *CustomResponseWriter {
	return &CustomResponseWriter{
		ResponseWriter: w,
	}
}

func (w *CustomResponseWriter) WriteHeader(statusCode int) {
	w.Status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *CustomResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.BytesWritten += n
	return n, err
}
