package validator

import (
	"context"
	"net"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator/validations"
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
