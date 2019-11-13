package erihttp

import (
	"io"
	"io/ioutil"
	"net/http"
)

func GetBodyFromHTTPRequest(r *http.Request) ([]byte, error) {
	var empty []byte
	const maxSizePlusOne int64 = 1<<20 + 1

	if r.Body == nil {
		return empty, ErrMissingBody
	}

	b, err := ioutil.ReadAll(io.LimitReader(r.Body, maxSizePlusOne))
	if err != nil {
		if err == io.EOF {
			return empty, ErrMissingBody
		}
		return empty, ErrInvalidRequest
	}

	if int64(len(b)) == maxSizePlusOne {
		return empty, ErrBodyTooLarge
	}

	return b, nil
}
