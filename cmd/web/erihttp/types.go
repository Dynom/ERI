package erihttp

type AutoCompleteResponse struct {
	Suggestions []string `json:"suggestions"`
}

type CheckResponse struct {
	Valid       bool   `json:"valid"`
	Reason      string `json:"reason,omitempty"`
	Alternative string `json:"alternative,omitempty"`
}

type AutoCompleteRequest struct {
	Domain string `json:"domain"`
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
