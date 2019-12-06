package validator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/Dynom/ERI/types"
)

var (
	ErrInvalidHost = errors.New("invalid host")
)

type Artifact struct {
	validations.Validations
	Timings
	email  types.EmailParts
	mx     []string
	ctx    context.Context
	dialer *net.Dialer
	conn   net.Conn
}

type stateFn func(a *Artifact) error

func NewEmailAddressValidator(dialer *net.Dialer) EmailValidator {

	// @todo fix when Go's stdlib offers a nicer API for this
	if dialer.Resolver == nil {
		dialer.Resolver = net.DefaultResolver
	}

	return EmailValidator{
		dialer: dialer,
	}
}

type EmailValidator struct {
	dialer *net.Dialer
}

func (v *EmailValidator) getNewArtifact(ctx context.Context, ep types.EmailParts) Artifact {
	a := Artifact{
		Validations: 0,
		Timings:     make(Timings, 10),
		email:       ep,
		mx:          []string{""},
		ctx:         ctx,
		dialer:      v.dialer,
		conn:        nil,
	}

	if deadline, set := ctx.Deadline(); set {
		a.dialer.Deadline = deadline
	}

	return a
}

// CheckWithConnect performs a thorough check, which has the least chance of false-positives. It requires a valid PTR
// and is probably not something you want to offer as a user-facing service.
func (v *EmailValidator) CheckWithConnect(ctx context.Context, emailParts types.EmailParts) (Artifact, error) {
	return validateSequence(ctx,
		v.getNewArtifact(ctx, emailParts),
		[]stateFn{
			checkSyntax,
			checkIfDomainHasMX,
			checkIfMXHasIP,
			checkMXAcceptsConnect,
			checkRCPT,
		})
}

// CheckWithLookup performs a sanity check using DNS lookups. It won't connect to the actual hosts.
func (v *EmailValidator) CheckWithLookup(ctx context.Context, emailParts types.EmailParts) (Artifact, error) {
	return validateSequence(ctx,
		v.getNewArtifact(ctx, emailParts),
		[]stateFn{
			checkSyntax,
			checkIfDomainHasMX,
			checkIfMXHasIP,
		})
}

// CheckWithSyntax performs only a syntax check.
func (v *EmailValidator) CheckWithSyntax(ctx context.Context, emailParts types.EmailParts) (Artifact, error) {
	return validateSequence(ctx,
		v.getNewArtifact(ctx, emailParts),
		[]stateFn{
			checkSyntax,
		})
}

func validateSequence(ctx context.Context, artifact Artifact, sequence []stateFn) (Artifact, error) {
	for _, v := range sequence {
		if err := v(&artifact); err != nil {
			return artifact, err
		}

		if t, deadlineSet := ctx.Deadline(); deadlineSet && !t.After(time.Now()) {
			return artifact, fmt.Errorf("deadline expired")
		}
	}

	artifact.Validations.MarkAsValid()
	return artifact, nil
}

// checkSyntax checks for "common sense" e-mail address syntax. It doesn't try to be fully compliant.
func checkSyntax(a *Artifact) error {
	var err error

	if a.Validations.HasFlag(validations.VFSyntax) {
		return nil
	}

	start := time.Now()
	defer a.Timings.Add("checkSyntax", time.Since(start))

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

// checkIfDomainHasMX performs a DNS lookup and fetches MX records.
func checkIfDomainHasMX(a *Artifact) error {
	start := time.Now()
	mxs, err := fetchMXHosts(a.ctx, a.dialer.Resolver, a.email.Domain)
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
				err = fmt.Errorf("%s %w", err, innerErr)
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

//nolint:gocyclo
func looksLikeValidLocalPart(local string) bool {

	var length = len(local)
	if 1 > length || length > 253 {
		return false
	}

	for i, c := range local {
		switch {
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 46 && 0 < i && i < length-1 /* . not first or last */ :
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :

		case c == 33 /* ! */ :
		case c == 35 /* # */ :
		case c == 36 /* $ */ :
		case c == 37 /* % */ :
		case c == 38 /* & */ :
		case c == 39 /* ' */ :
		case c == 42 /* * */ :
		case c == 43 /* + */ :
		case c == 45 /* - */ :
		case c == 47 /* / */ :
		case c == 61 /* = */ :
		case c == 63 /* ? */ :
		case c == 94 /* ^ */ :
		case c == 95 /* _ */ :
		case c == 96 /* ` */ :
		case c == 123 /* { */ :
		case c == 124 /* | */ :
		case c == 125 /* } */ :
		case c == 126 /* ~ */ :
		default:
			return false
		}
	}

	return true
}

//nolint:gocyclo
func looksLikeValidDomain(domain string) bool {
	var length = len(domain)
	const dot uint8 = 46

	// Normally we can assume that host names have a tld and/or consists at least out of 4 characters
	if 4 >= length || length >= 253 {
		return false
	}

	// No dots on the outside of the domain name
	if domain[0] == dot || domain[length-1] == dot {
		return false
	}

	for i, c := range domain {
		switch {
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 46 /* dot . */ :

		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case c == 45 && 0 < i /* dash - */ :

		default:
			return false
		}
	}

	return true
}
