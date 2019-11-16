package validators

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"

	"github.com/Dynom/ERI/cmd/web/types"
)

type Artifact struct {
	Validations
	types.Timings
	Error    error
	email    types.EmailParts
	mx       []string
	ctx      context.Context
	resolver net.Resolver
	dialer   net.Dialer
	conn     net.Conn
}

type stateFn func(a *Artifact) stateFn

func NewSMValidator(resolver net.Resolver, dialer net.Dialer) SMValidator {
	return SMValidator{
		resolver: resolver,
		dialer:   dialer,
	}
}

type SMValidator struct {
	resolver net.Resolver
	dialer   net.Dialer
}

func (v *SMValidator) getNewArtifact(ctx context.Context, ep types.EmailParts) Artifact {
	a := Artifact{
		Validations: 0,
		Timings:     make(types.Timings, 10),
		Error:       nil,
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

func (v *SMValidator) CheckEmailAddress(ctx context.Context, addr string) (error, Artifact) {
	p, err := types.NewEmailParts(addr)
	if err != nil {
		return err, Artifact{}
	}

	a := v.getNewArtifact(ctx, p)

	for validator := checkSyntax; validator != nil; {
		validator = validator(&a)
	}

	if a.Error == nil {
		a.Validations |= VFValid
	}

	return nil, a
}

func checkSyntax(a *Artifact) stateFn {
	_, err := mail.ParseAddress(a.email.Address)
	if err != nil {
		a.Error = err
		return nil
	}

	a.Validations |= VFSyntax
	return checkIfDomainHasMX
}

func checkIfDomainHasMX(a *Artifact) stateFn {
	mxHost, err := fetchMXHost(a.ctx, &a.resolver, a.email.Domain)
	if err != nil {
		a.Error = err
		return nil
	}

	a.mx = []string{mxHost}
	a.Validations |= VFMXLookup
	return checkIfMXHasIP
}

func checkIfMXHasIP(a *Artifact) stateFn {
	var err error
	for i, domain := range a.mx {
		ips, innerErr := a.resolver.LookupIPAddr(a.ctx, domain)
		if innerErr != nil || len(ips) == 0 {
			a.mx[i] = ""

			if innerErr != nil {
				err = fmt.Errorf("%s %w", err, innerErr)
			}
		}
	}

	if err != nil {
		a.Error = err
		return nil
	}

	a.Validations |= VFDomainHasIP
	return checkMXAcceptsConnect
}

func checkMXAcceptsConnect(a *Artifact) stateFn {
	conn, err := getConnection(a.ctx, a.dialer, a.mx[0])
	if err != nil {
		a.Error = err
		return nil
	}

	a.conn = conn
	a.Validations |= VFHostConnect
	return checkRCPT
}

func checkRCPT(a *Artifact) stateFn {
	const recipient = "eri@tysug.net"

	client, err := smtp.NewClient(a.conn, a.email.Domain)
	if err != nil {
		a.Error = err
		return nil
	}

	defer func() {
		_ = client.Quit()
	}()

	err = client.Mail(recipient)
	if err != nil {
		a.Error = err
		return nil
	}

	a.Error = client.Rcpt(a.email.Address)

	if a.Error == nil {
		a.Validations |= VFValidRCPT
	}

	return nil
}
