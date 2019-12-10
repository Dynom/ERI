package inspector

// Option is the type accepted by mail check to set specific options
type Option func(c *Checker)

// WithValidators adds validators to mailcheck. The order is significant, put the cheapest first
func WithValidators(validators ...Validator) Option {
	return func(c *Checker) {
		c.validators = validators
	}
}
