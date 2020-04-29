package erihttp

import (
	"io"
	"io/ioutil"
	"net/http"
)

const (
	MaxBodySize int64 = 1 << 20
)

func GetBodyFromHTTPRequest(r *http.Request) ([]byte, error) {
	var empty []byte

	if r.Body == nil {
		return empty, ErrMissingBody
	}

	if r.ContentLength > MaxBodySize {
		return empty, ErrBodyTooLarge
	}

	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		return empty, ErrUnsupportedContentType
	}

	b, err := ioutil.ReadAll(io.LimitReader(r.Body, MaxBodySize+1))
	if err != nil {
		return empty, ErrInvalidRequest
	}

	if int64(len(b)) > MaxBodySize {
		return empty, ErrBodyTooLarge
	}

	return b, nil
}
