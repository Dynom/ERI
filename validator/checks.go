package validator

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/Dynom/ERI/validator/validations"
)

// checkEmailAddressSyntax checks for "common sense" e-mail address syntax. It doesn't try to be fully compliant.
func checkEmailAddressSyntax(a *Artifact) error {
	var err error

	start := time.Now()
	defer a.Timings.Add("checkEmailAddressSyntax", time.Since(start))

	_, err = mail.ParseAddress(a.email.Address)
	if err != nil {
		return err
	}

	// Perform additional checks to weed out commonly occurring errors (see tests for details)
	if !looksLikeValidLocalPart(a.email.Local) {
		return fmt.Errorf("local part '%s' has invalid syntax", a.email.Local)
	}

	if !looksLikeValidDomain(a.email.Domain) {
		return fmt.Errorf("domain part '%s' has invalid syntax", a.email.Domain)
	}

	a.Validations = a.Validations.MergeWithNext(validations.VFSyntax)
	return nil
}

// checkDomainSyntax checks if the Domain part has a sensible syntax. It ignores the Local part, so that can be omitted
func checkDomainSyntax(a *Artifact) error {
	start := time.Now()
	defer a.Timings.Add("checkDomainSyntax", time.Since(start))

	if !looksLikeValidDomain(a.email.Domain) {
		return fmt.Errorf("domain part '%s' has invalid syntax", a.email.Domain)
	}

	a.Validations = a.Validations.MergeWithNext(validations.VFSyntax)
	return nil
}

// getEarliestDeadlineCTX returns a context with the deadline set to whatever is earliest
func getEarliestDeadlineCTX(parentCTX context.Context, ttl time.Duration) (context.Context, context.CancelFunc) {

	parentDeadline, ok := parentCTX.Deadline()
	if ok {
		ourDeadline := time.Now().Add(ttl)
		if ourDeadline.Before(parentDeadline) {
			return context.WithDeadline(parentCTX, ourDeadline)
		}
	}

	return context.WithTimeout(parentCTX, ttl)
}

// checkIfDomainHasMX performs a DNS lookup and fetches MX records.
func checkIfDomainHasMX(a *Artifact) error {

	const ttl = 100 * time.Millisecond
	ctx, cancel := getEarliestDeadlineCTX(a.ctx, ttl)
	defer cancel()

	start := time.Now()
	mxs, err := fetchMXHosts(ctx, a.dialer.Resolver, a.email.Domain)
	a.Timings.Add("checkIfDomainHasMX", time.Since(start))

	if err != nil {
		if e, ok := err.(net.Error); ok && e.Temporary() {
			_ = e
			// @todo what do we do when it's a temporary error?
		}
		return err
	}

	a.mx = mxs
	a.Validations = a.Validations.MergeWithNext(validations.VFMXLookup)
	return nil
}

// checkIfMXHasIP performs a NS lookup and fetches the IP addresses of the MX hosts
func checkIfMXHasIP(a *Artifact) error {
	var err error

	for i, domain := range a.mx {
		start := time.Now()
		ips, innerErr := a.dialer.Resolver.LookupIPAddr(a.ctx, domain)
		a.Timings.Add("checkIfMXHasIP "+domain, time.Since(start))

		if innerErr != nil || len(ips) == 0 {
			a.mx[i] = ""

			if e, ok := err.(net.Error); ok && e.Temporary() {
				// @todo what do we do when it's a temporary error?
				_ = e
			}

			if innerErr != nil {
				err = wrapError(err, innerErr)
			}
		}
	}

	if err != nil {
		return err
	}

	a.Validations = a.Validations.MergeWithNext(validations.VFDomainHasIP)
	return nil
}

// checkMXAcceptsConnect checks if an MX host accepts connections. Expensive and requires a valid PTR setup for most
// real world applications
func checkMXAcceptsConnect(a *Artifact) error {

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

	if e, ok := err.(net.Error); ok && e.Temporary() {
		// @todo what do we do when it's a temporary error?
		_ = e
	}

	if err != nil {
		return err
	}

	a.conn = conn
	a.Validations = a.Validations.MergeWithNext(validations.VFHostConnect)
	return nil
}

// checkRCPT issues mail commands to the mail server, asking politely if a recipient inbox exists. High chance of false
// positives on real world applications, due to security reasons.
func checkRCPT(a *Artifact) error {
	if a.HasFlag(validations.VFValidRCPT) {
		return nil
	}

	var client *smtp.Client
	var start time.Time
	var err error

	client, err = smtp.NewClient(a.conn, a.email.Domain)

	if err != nil {
		return err
	}

	defer func() {
		_ = client.Quit()
	}()

	start = time.Now()
	err = client.Verify(a.email.Address)
	a.Timings.Add("checkRCPT", time.Since(start))

	if err == nil {
		a.Validations = a.Validations.MergeWithNext(validations.VFValidRCPT)
	}

	return err
}
