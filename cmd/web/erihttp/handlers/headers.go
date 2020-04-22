package handlers

import "net/http"

func WithHeaders(headers http.Header) Middleware {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addHeaders(w.Header(), headers)

			handler.ServeHTTP(w, r)
		})
	}
}

func addHeaders(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}
