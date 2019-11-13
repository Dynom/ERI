package handlers

import "net/http"

func WithHeaders(headers http.Header) HandlerWrapper {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			copyHeaders(w.Header(), headers)

			handler.ServeHTTP(w, r)
		})
	}
}

func copyHeaders(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}
