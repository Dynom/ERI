package erihttp

import (
	"io"
	"io/ioutil"
	"net/http"
)

// GetBodyFromHTTPRequest performs basic request validation and returns the body if all conditions are met
func GetBodyFromHTTPRequest(r *http.Request, maxBodySize int64) ([]byte, error) {
	var empty []byte

	if r.Body == nil {
		return empty, ErrMissingBody
	}

	if r.ContentLength > maxBodySize {
		return empty, ErrBodyTooLarge
	}

	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		return empty, ErrUnsupportedContentType
	}

	b, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	if err != nil {
		return empty, ErrInvalidRequest
	}

	if int64(len(b)) > maxBodySize {
		return empty, ErrBodyTooLarge
	}

	return b, nil
}
