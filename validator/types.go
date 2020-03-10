package validator

import (
	"context"
	"net"

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
	Timings
}

func (r Result) ValidatorsRan() bool {
	return r.Steps > 0 || r.Validations > 0
}

func createResult(a Artifact) Result {
	return Result{
		Validations: a.Validations,
		Steps:       a.Steps,
		Timings:     a.Timings,
	}
}
