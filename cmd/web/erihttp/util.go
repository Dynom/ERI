package erihttp

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// GetBodyFromHTTPRequest performs basic request validation and returns the body if all conditions are met
func GetBodyFromHTTPRequest(r *http.Request, maxBodySize int64) ([]byte, error) {
	var empty []byte

	if r.Method != http.MethodPost {
		if len(r.Method) > 16 {
			// If the method value exceeds this size, let's not bother logging it since it might be abuse. Number is arbitrary
			return empty, ErrInvalidRequest
		}

		return empty, fmt.Errorf("%w HTTP Method %q is unsupported", ErrInvalidRequest, r.Method)
	}

	if r.Body == nil {
		return empty, ErrMissingBody
	}

	if r.ContentLength > maxBodySize {
		return empty, ErrBodyTooLarge
	}

	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		if len(ct) > 128 {
			// If the header value exceeds this size, let's not bother logging it since it might be abuse. Number is arbitrary
			return empty, ErrUnsupportedContentType
		}

		return empty, fmt.Errorf("%w %q", ErrUnsupportedContentType, ct)
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
