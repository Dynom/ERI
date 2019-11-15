package inspector

import "github.com/Dynom/ERI/cmd/web/inspector/validators"

// Option is the type accepted by mail check to set specific options
type Option func(c *Checker)

// WithValidators adds validators to mailcheck. The order is significant, put the cheapest first
func WithValidators(validators ...validators.Validator) Option {
	return func(c *Checker) {
		c.validators = validators
	}
}
