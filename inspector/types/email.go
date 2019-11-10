package types

import (
	"strings"

	"github.com/Dynom/ERI/inspector"
)

func NewEmailParts(emailAddress string) (EmailParts, error) {
	p, err := splitLocalAndDomain(emailAddress)
	if err != nil {
		return EmailParts{}, err
	}

	return p, nil
}

type EmailParts struct {
	Address string
	Local   string
	Domain  string
}

func splitLocalAndDomain(input string) (EmailParts, error) {
	i := strings.LastIndex(input, "@")
	if 0 >= i || i >= len(input) {
		return EmailParts{}, inspector.ErrInvalidEmailAddress
	}

	return EmailParts{
		Address: input,
		Local:   input[:i],
		Domain:  strings.ToLower(input[i+1:]),
	}, nil
}
