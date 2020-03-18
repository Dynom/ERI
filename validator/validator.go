package validator

import (
	"context"
	"net"
	"time"

	"github.com/Dynom/ERI/types"
)

type CheckFn func(ctx context.Context, parts types.EmailParts, options ...ArtifactFn) Result
type ArtifactFn func(artifact *Artifact)

func NewEmailAddressValidator(dialer *net.Dialer) EmailValidator {

	// @todo fix when Go's stdlib offers a nicer API for this
	if dialer == nil {
		dialer = &net.Dialer{}
	}

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

// CheckWithConnect performs a thorough check, which has the least chance of false-positives. It requires a valid PTR
// and is probably not something you want to offer as a user-facing service.
func (v *EmailValidator) CheckWithConnect(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {

	var syntaxCheck stateFn = checkEmailAddressSyntax
	if emailParts.Local == "" {
		syntaxCheck = checkDomainSyntax
	}

	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, v.dialer, options...),
		[]stateFn{
			syntaxCheck,
			checkIfDomainHasMX,
			checkIfMXHasIP,
			checkMXAcceptsConnect,
			checkRCPT,
		})

	return createResult(artifact)
}

// CheckWithLookup performs a sanity check using DNS lookups. It won't connect to the actual hosts.
func (v *EmailValidator) CheckWithLookup(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {

	var syntaxCheck stateFn = checkEmailAddressSyntax
	if emailParts.Local == "" {
		syntaxCheck = checkDomainSyntax
	}

	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, v.dialer, options...),
		[]stateFn{
			syntaxCheck,
			checkIfDomainHasMX,
			checkIfMXHasIP,
		})

	return createResult(artifact)
}

// CheckWithSyntax performs only a syntax check.
func (v *EmailValidator) CheckWithSyntax(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {

	var syntaxCheck stateFn = checkEmailAddressSyntax
	if emailParts.Local == "" {
		syntaxCheck = checkDomainSyntax
	}

	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, v.dialer, options...),
		[]stateFn{
			syntaxCheck,
		})

	return createResult(artifact)
}

func validateSequence(ctx context.Context, artifact Artifact, sequence []stateFn) (Artifact, error) {
	for _, v := range sequence {
		if err := v(&artifact); err != nil {
			return artifact, err
		}

		if t, deadlineSet := ctx.Deadline(); deadlineSet && !t.After(time.Now()) {
			return artifact, context.DeadlineExceeded
		}
	}

	artifact.Validations.MarkAsValid()
	return artifact, nil
}
