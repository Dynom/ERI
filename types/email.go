package types

import (
	"errors"
	"strings"
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
		return EmailParts{}, ErrInvalidEmailAddress
	}

	return EmailParts{
		Address: input,
		Local:   input[:i],
		Domain:  strings.ToLower(input[i+1:]),
	}, nil
}

var (
	ErrInvalidEmailAddress = errors.New("invalid e-mail address, address is missing @")
)
