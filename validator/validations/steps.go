package validations

import "fmt"

// Steps holds the validation steps performed, they do not signify validity
type Steps uint8

func (v Steps) String() string {
	return fmt.Sprintf("%08b", v)
}

// HasBeenValidated returns true if any validations steps have actually been taken.
func (v Steps) HasBeenValidated() bool {
	return v > 0
}

// SetFlag defines a flag on the type and returns a copy
func (v *Steps) SetFlag(new Flag) Steps {
	*v |= Steps(new)

	return *v
}

// HasFlag returns true if the type has the flag (or flags) specified
func (v Steps) HasFlag(f Flag) bool {
	return v&Steps(f) != 0
}
