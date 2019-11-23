package validators

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/Dynom/ERI/cmd/web/types"
)

type Artifact struct {
	Validations
	types.Timings
	email  types.EmailParts
	mx     []string
	ctx    context.Context
	dialer *net.Dialer
	conn   net.Conn
}

type stateFn func(a *Artifact) error

func NewSMValidator(dialer *net.Dialer) SMValidator {

	// @todo fix when Go's stdlib offers a nicer API for this
	if dialer.Resolver == nil {
		dialer.Resolver = net.DefaultResolver
	}

	return SMValidator{
		dialer: dialer,
	}
}

type SMValidator struct {
	dialer *net.Dialer
}

func (v *SMValidator) getNewArtifact(ctx context.Context, ep types.EmailParts) Artifact {
	a := Artifact{
		Validations: 0,
		Timings:     make(types.Timings, 10),
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

func (v *SMValidator) CheckFull(ctx context.Context, emailParts types.EmailParts) (Artifact, error) {
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

func (v *SMValidator) CheckBasic(ctx context.Context, emailParts types.EmailParts) (Artifact, error) {
	return validateSequence(ctx,
		v.getNewArtifact(ctx, emailParts),
		[]stateFn{
			checkSyntax,
			checkIfDomainHasMX,
			checkIfMXHasIP,
			//checkMXAcceptsConnect,	// Connections to MX's requires a valid PTR setup, which isn't applicable in certain situations
			//checkRCPT,
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

func checkSyntax(a *Artifact) error {
	var err error

	if a.Validations.HasFlag(VFSyntax) {
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

	a.Validations = a.Validations.MergeWithNext(VFSyntax)
	return nil
}

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
	a.Validations = a.Validations.MergeWithNext(VFMXLookup)
	return nil
}

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

	a.Validations = a.Validations.MergeWithNext(VFDomainHasIP)
	return nil
}

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
	a.Validations = a.Validations.MergeWithNext(VFHostConnect)
	return nil
}

func checkRCPT(a *Artifact) error {
	if a.HasFlag(VFValidRCPT) {
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
	a.Timings.Add("checkRCPT RCPT", time.Since(start))

	if err == nil {
		a.Validations = a.Validations.MergeWithNext(VFValidRCPT)
	}

	return err
}

//nolint:gocyclo
func looksLikeValidLocalPart(local string) bool {

	var length = len(local)
	if length < 1 {
		return false
	}

	for i, c := range local {
		switch {
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 46 && 0 < i && i < length-1 /* . not first or last */ :

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

	// Normally we can assume that host names have a tld or consists at least out of 4 characters
	if 4 >= length || length >= 253 {
		return false
	}

	if domain[0] == 46 || domain[length-1] == 46 {
		return false
	}

	for i, c := range domain {
		switch {
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 45 && 0 < i /* dash - */ :
		case c == 46 /* dot . */ :
		default:
			return false
		}
	}

	return true
}
