package validator

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/Dynom/ERI/validator/validations"
)

type ValidationError struct {
	Validator string
	Internal  error
	error
}

// checkEmailAddressSyntax checks for "common sense" e-mail address syntax. It doesn't try to be fully compliant.
func checkEmailAddressSyntax(a *Artifact) error {
	a.Steps.SetFlag(validations.FSyntax)

	var err error

	start := time.Now()
	defer func() {
		a.Timings.Add("checkEmailAddressSyntax", time.Since(start))
	}()

	_, err = mail.ParseAddress(a.email.Address)
	if err != nil {
		return ValidationError{
			Validator: "checkEmailAddressSyntax",
			Internal:  err,
			error:     ErrEmailAddressSyntax,
		}
	}

	// Perform additional checks to weed out commonly occurring errors (see tests for details)
	if !looksLikeValidLocalPart(a.email.Local) {
		return ValidationError{
			Validator: "checkEmailAddressSyntax",
			Internal:  fmt.Errorf("local part '%s' has invalid syntax", a.email.Local),
			error:     ErrEmailAddressSyntax,
		}
	}

	if !looksLikeValidDomain(a.email.Domain) {
		return ValidationError{
			Validator: "checkEmailAddressSyntax",
			Internal:  fmt.Errorf("domain part '%s' has invalid syntax", a.email.Domain),
			error:     ErrEmailAddressSyntax,
		}
	}

	a.Validations = a.Validations.SetFlag(validations.FSyntax)
	return nil
}

// checkDomainSyntax checks if the Domain part has a sensible syntax. It ignores the Local part, so that can be omitted
func checkDomainSyntax(a *Artifact) error {
	a.Steps.SetFlag(validations.FSyntax)

	start := time.Now()
	defer func() {
		a.Timings.Add("checkDomainSyntax", time.Since(start))
	}()

	if !looksLikeValidDomain(a.email.Domain) {
		return ValidationError{
			Validator: "checkDomainSyntax",
			Internal:  fmt.Errorf("domain part '%s' has invalid syntax", a.email.Domain),
			error:     ErrEmailAddressSyntax,
		}
	}

	a.Validations.SetFlag(validations.FSyntax)
	return nil
}

// checkIfDomainHasMX performs a DNS lookup and fetches MX records.
func checkIfDomainHasMX(a *Artifact) error {
	if a.Steps.HasFlag(validations.FMXLookup) {
		if !a.Validations.HasFlag(validations.FMXLookup) {
			return ValidationError{
				Validator: "checkIfDomainHasMX",
				error:     ErrEmailAddressSyntax,
			}
		}

		return nil
	}

	a.Steps.SetFlag(validations.FMXLookup)

	start := time.Now()
	mxs, err := fetchMXHosts(a.ctx, a.resolver, a.email.Domain)
	a.Timings.Add("checkIfDomainHasMX", time.Since(start))

	if err != nil {
		return ValidationError{
			Validator: "checkIfDomainHasMX",
			Internal:  err,
			error:     ErrEmailAddressSyntax,
		}
	}

	a.mx = mxs
	a.Validations.SetFlag(validations.FMXLookup)
	return nil
}

// checkIfMXHasIP performs a NS lookup and fetches the IP addresses of the MX hosts.
// expects to run after checkIfDomainHasMX()
func checkIfMXHasIP(a *Artifact) error {
	if a.Steps.HasFlag(validations.FMXDomainHasIP) {
		if !a.Validations.HasFlag(validations.FMXDomainHasIP) {
			return ValidationError{
				Validator: "checkIfMXHasIP",
				error:     ErrEmailAddressSyntax,
			}
		}

		return nil
	}

	a.Steps.SetFlag(validations.FMXDomainHasIP)

	var err error
	for i, domain := range a.mx {
		start := time.Now()
		ips, innerErr := a.resolver.LookupMX(a.ctx, domain)
		a.Timings.Add("checkIfMXHasIP "+domain, time.Since(start))

		if innerErr != nil || len(ips) == 0 {
			a.mx[i] = ""

			if innerErr != nil {
				err = wrapError(err, innerErr)
			}
		}
	}

	if err != nil {
		return ValidationError{
			Validator: "checkIfMXHasIP",
			Internal:  err,
			error:     ErrEmailAddressSyntax,
		}
	}

	a.Validations.SetFlag(validations.FMXDomainHasIP)
	return nil
}

// checkMXAcceptsConnect checks if an MX host accepts connections. Expensive and requires a valid PTR setup for most
// real world applications
func checkMXAcceptsConnect(a *Artifact) error {
	if a.Steps.HasFlag(validations.FHostConnect) {
		if !a.Validations.HasFlag(validations.FHostConnect) {
			return ValidationError{
				Validator: "checkMXAcceptsConnect",
				error:     ErrEmailAddressSyntax,
			}
		}

		return nil
	}

	a.Steps.SetFlag(validations.FHostConnect)

	start := time.Now()
	var mxToCheck string
	for _, domain := range a.mx {
		if domain != "" {
			mxToCheck = domain
			break
		}
	}

	conn, err := getConnection(a.ctx, a.dialer, mxToCheck)
	a.Timings.Add("checkMXAcceptsConnect", time.Since(start))

	if err != nil {
		return ValidationError{
			Validator: "checkMXAcceptsConnect",
			Internal:  err,
			error:     ErrEmailAddressSyntax,
		}
	}

	a.conn = conn
	a.Validations.SetFlag(validations.FHostConnect)
	return nil
}

// checkRCPT issues mail commands to the mail server, asking politely if a recipient inbox exists. High chance of false
// positives on real world applications, due to security reasons.
func checkRCPT(a *Artifact) error {
	if a.Steps.HasFlag(validations.FValidRCPT) {
		if !a.Validations.HasFlag(validations.FValidRCPT) {
			return ValidationError{
				Validator: "checkRCPT",
				error:     ErrEmailAddressSyntax,
			}
		}

		return nil
	}

	a.Steps.SetFlag(validations.FValidRCPT)

	if a.Validations.HasFlag(validations.FValidRCPT) {
		return nil
	}

	var client *smtp.Client
	var start time.Time
	var err error

	client, err = smtp.NewClient(a.conn, a.email.Domain)

	if err != nil {
		return ValidationError{
			Validator: "checkRCPT",
			Internal:  err,
			error:     ErrEmailAddressSyntax,
		}
	}

	defer func() {
		_ = client.Quit()
	}()

	start = time.Now()
	err = client.Verify(a.email.Address)
	a.Timings.Add("checkRCPT", time.Since(start))

	if err == nil {
		a.Validations.SetFlag(validations.FValidRCPT)
	}

	return ValidationError{
		Validator: "checkRCPT",
		Internal:  err,
		error:     ErrEmailAddressSyntax,
	}
}
