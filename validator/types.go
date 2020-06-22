package validator

import (
	"context"
	"net"
	"time"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator/validations"
)

type Artifact struct {
	Validations validations.Validations
	Steps       validations.Steps
	Timings
	email  types.EmailParts
	mx     []string
	ctx    context.Context
	dialer *net.Dialer
	conn   net.Conn
}

type stateFn func(a *Artifact) error

type Result struct {
	Validations validations.Validations
	Steps       validations.Steps
}

type Details struct {
	Result

	ValidUntil time.Time
}

func (r Result) ValidatorsRan() bool {
	return r.Steps > 0
}

func (r Result) HasValidStructure() bool {
	return r.ValidatorsRan() && r.Validations.HasFlag(validations.FSyntax) && r.Steps.HasFlag(validations.FSyntax)
}

func createResult(a Artifact) Result {
	return Result{
		Validations: a.Validations,
		Steps:       a.Steps,
	}
}
