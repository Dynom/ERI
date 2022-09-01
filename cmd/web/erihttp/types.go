package erihttp

import "errors"

var (
	ErrMissingBody            = errors.New("missing body")
	ErrInvalidRequest         = errors.New("request is invalid")
	ErrBodyTooLarge           = errors.New("request body too large")
	ErrUnsupportedContentType = errors.New("unsupported content-type")
)

var empty = make([]string, 0)

type ERIResponse interface {
	// Hacking around Generics, like it's 1999.
	PrepareResponse()
}

type AutoCompleteResponse struct {
	Suggestions []string `json:"suggestions"`
	Error       string   `json:"error,omitempty"`
}

func (r *AutoCompleteResponse) PrepareResponse() {
	if r.Suggestions == nil {
		r.Suggestions = empty
	}
}

type SuggestResponse struct {
	Alternatives    []string `json:"alternatives"`
	MalformedSyntax bool     `json:"malformed_syntax"`
	MisconfiguredMX bool     `json:"misconfigured_mx"`
	Error           string   `json:"error,omitempty"`
}

func (r *SuggestResponse) PrepareResponse() {
	if r.Alternatives == nil {
		r.Alternatives = empty
	}
}

type AutoCompleteRequest struct {
	Domain string `json:"domain"`
}

type SuggestRequest struct {
	Email string `json:"email"`
}
