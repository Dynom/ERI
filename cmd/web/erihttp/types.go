package erihttp

//{"valid": false, "reason": "bad_domain",         "alternative": "john.doe@gmail.com"}
type CheckResponse struct {
	Valid       bool   `json:"valid"`
	Reason      string `json:"reason,omitempty"`
	Alternative string `json:"alternative,omitempty"`
}

type CheckRequest struct {
	Email        string `json:"email"`
	Alternatives bool   `json:"with_alternatives"`
}

type LearnRequest struct {
	Emails  []ToLearn `json:"emails"`
	Domains []ToLearn `json:"domains"`
}

type ToLearn struct {
	Value string `json:"value"`
	Valid bool   `json:"valid"`
}
