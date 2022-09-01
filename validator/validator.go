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

func prependOptions(options []ArtifactFn, o ...ArtifactFn) []ArtifactFn {
	return append(o, options...)
}

// CheckWithRCPT performs a thorough check, which includes a RCPT check. It requires a valid PTR and is probably not
// something you want to offer as a user-facing service.
//
// Warning: Using this _can_ degrade your IPs reputation, since it's also a process spammers use.
func (v *EmailValidator) CheckWithRCPT(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {
	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, prependOptions(options, WithDialer(v.dialer))...),
		[]stateFn{
			getSyntaxCheck(emailParts),
			checkIfDomainHasMX,
			checkIfMXHasIP,
			checkMXAcceptsConnect,
			checkRCPT,
		})

	return createResult(artifact)
}

// CheckWithConnect performs a thorough check, which has the low chance of false-positives. It also tests if the MX server
// accepts connections, but won't try any mail commands.
func (v *EmailValidator) CheckWithConnect(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {
	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, prependOptions(options, WithDialer(v.dialer))...),
		[]stateFn{
			getSyntaxCheck(emailParts),
			checkIfDomainHasMX,
			checkIfMXHasIP,
			checkMXAcceptsConnect,
		})

	return createResult(artifact)
}

// CheckWithLookup performs a sanity check using DNS lookups. It won't connect to the actual hosts.
func (v *EmailValidator) CheckWithLookup(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {
	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, prependOptions(options, WithDialer(v.dialer))...),
		[]stateFn{
			getSyntaxCheck(emailParts),
			checkIfDomainHasMX,
			checkIfMXHasIP,
		})

	return createResult(artifact)
}

// CheckWithSyntax performs only a syntax check.
func (v *EmailValidator) CheckWithSyntax(ctx context.Context, emailParts types.EmailParts, options ...ArtifactFn) Result {
	artifact, _ := validateSequence(ctx,
		getNewArtifact(ctx, emailParts, prependOptions(options, WithDialer(v.dialer))...),
		[]stateFn{
			getSyntaxCheck(emailParts),
		})

	return createResult(artifact)
}

// getSyntaxCheck returns a domain only check, when the local part is missing and otherwise uses a full address check
func getSyntaxCheck(parts types.EmailParts) stateFn {
	var syntaxCheck stateFn = checkEmailAddressSyntax
	if parts.Local == "" {
		syntaxCheck = checkDomainSyntax
	}
	return syntaxCheck
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
