package validations

import "fmt"

// Steps holds the validation steps performed, they do not signify validity
type Steps uint8

func (s Steps) String() string {
	return fmt.Sprintf("%08b", s)
}

// MergeWithNext appends to Steps and returns the result.
func (s Steps) MergeWithNext(new Steps) Steps {
	if new < s {
		return new
	}

	return s | new
}

// HasBeenValidated returns true if any validations steps have actually been taken.
func (s Steps) HasBeenValidated() bool {
	return s > 0
}

// SetFlag defines a flag on the type and returns a copy
func (s *Steps) SetFlag(new Flag) Steps {
	*s |= Steps(new)

	return *s
}

func (s *Steps) RemoveFlag(f Flag) Steps {
	*s &^= Steps(f)
	return *s
}

// HasFlag returns true if the type has the flag (or flags) specified
func (s Steps) HasFlag(f Flag) bool {
	return s&Steps(f) != 0
}
