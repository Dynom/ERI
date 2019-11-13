package main

import (
	"net/http"

	"github.com/Dynom/ERI/cmd/web/config"
)

func sliceToHTTPHeaders(slice []config.Header) http.Header {
	headers := http.Header{}
	for _, h := range slice {
		headers.Add(h.Name, h.Value)
	}

	return headers
}
