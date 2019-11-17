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
	email    types.EmailParts
	mx       []string
	ctx      context.Context
	resolver *net.Resolver
	dialer   *net.Dialer
	conn     net.Conn
	next     stateFn
}

type stateFn func(a *Artifact) error

func NewSMValidator(resolver *net.Resolver, dialer *net.Dialer) SMValidator {
	return SMValidator{
		resolver: resolver,
		dialer:   dialer,
	}
}

type SMValidator struct {
	resolver *net.Resolver
	dialer   *net.Dialer
}

func (v *SMValidator) getNewArtifact(ctx context.Context, ep types.EmailParts) Artifact {
	a := Artifact{
		Validations: 0,
		Timings:     make(types.Timings, 10),
		email:       ep,
		mx:          []string{""},
		ctx:         ctx,
		resolver:    v.resolver,
		dialer:      v.dialer,
		conn:        nil,
	}

	if deadline, set := ctx.Deadline(); set {
		a.dialer.Deadline = deadline
	}

	return a
}

func (v *SMValidator) CheckEmailAddress(ctx context.Context, emailParts types.EmailParts) (Artifact, error) {
	a := v.getNewArtifact(ctx, emailParts)
	for validator := checkSyntax; validator != nil; {
		if err := validator(&a); err != nil {
			return a, err
		}

		validator = a.next
	}

	a.Validations |= VFValid
	return a, nil
}

func checkSyntax(a *Artifact) error {
	start := time.Now()
	_, err := mail.ParseAddress(a.email.Address)
	a.Timings.Add("checkSyntax", time.Since(start))

	if err != nil {
		return err
	}

	a.Validations |= VFSyntax
	a.next = checkIfDomainHasMX
	return nil
}

func checkIfDomainHasMX(a *Artifact) error {
	start := time.Now()
	mxs, err := fetchMXHosts(a.ctx, a.resolver, a.email.Domain)
	a.Timings.Add("checkIfDomainHasMX", time.Since(start))

	if err != nil {
		return err
	}

	a.mx = mxs
	a.Validations |= VFMXLookup
	a.next = checkIfMXHasIP
	return nil
}

func checkIfMXHasIP(a *Artifact) error {
	var err error

	for i, domain := range a.mx {
		start := time.Now()
		ips, innerErr := a.resolver.LookupIPAddr(a.ctx, domain)
		a.Timings.Add("checkIfMXHasIP "+domain, time.Since(start))

		if innerErr != nil || len(ips) == 0 {
			a.mx[i] = ""

			if innerErr != nil {
				err = fmt.Errorf("%s %w", err, innerErr)
			}
		}
	}

	if err != nil {
		return err
	}

	a.Validations |= VFDomainHasIP
	a.next = checkMXAcceptsConnect
	return nil
}

func checkMXAcceptsConnect(a *Artifact) error {
	start := time.Now()
	conn, err := getConnection(a.ctx, a.dialer, a.mx[0])
	a.Timings.Add("checkMXAcceptsConnect", time.Since(start))

	if err != nil {
		return err
	}

	a.conn = conn
	a.Validations |= VFHostConnect
	a.next = checkRCPT
	return nil
}

func checkRCPT(a *Artifact) error {
	const fakeSender = "eri@tysug.net"
	var start time.Time

	client, err := smtp.NewClient(a.conn, a.email.Domain)

	if err != nil {
		return err
	}

	defer func() {
		_ = client.Quit()
	}()

	start = time.Now()
	err = client.Mail(fakeSender)
	a.Timings.Add("checkRCPT Mail", time.Since(start))
	if err != nil {
		return err
	}

	start = time.Now()
	err = client.Rcpt(a.email.Address)
	a.Timings.Add("checkRCPT RCPT", time.Since(start))

	if err == nil {
		a.Validations |= VFValidRCPT
	}

	return err
}
