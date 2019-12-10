package validations

import "fmt"

const (
	// Validation Flags, these flags represent successful validation steps. Depending on how far you want to go, you can
	// classify a validation as valid enough, for your use-case.
	VFValid       Validations = 1 << iota // The e-mail is considered valid (1) or not (0)
	VFSyntax      Validations = 1 << iota // e-mail address follows a (reasonably) valid syntax
	VFMXLookup    Validations = 1 << iota // e-mail domain has MX records
	VFDomainHasIP Validations = 1 << iota // The domain has IP's
	VFHostConnect Validations = 1 << iota // MX accepts connections
	VFValidRCPT   Validations = 1 << iota // MX acknowledges that the RCPT exists
	VFDisposable  Validations = 1 << iota // Address / Domain is considered a disposable e-mail trap
)

// Validations holds the validation steps performed.
type Validations uint64

func (v Validations) String() string {
	return fmt.Sprintf("%08b", v)
}

// IsValid returns true if the Validations are considered successful
func (v Validations) IsValid() bool {
	return v&VFValid == VFValid
}

// MergeWithNext appends to Validations are returns the result. If the new validations aren't considered valid, it will
// mark the new Validations as unsuccessful as well. It's opinionated in that it's part of an incremental validation chain
func (v Validations) MergeWithNext(new Validations) Validations {

	v.MarkAsInvalid()
	return v | new
}

// MarkAsInvalid clears the CFValid bit and marks the Validations as invalid
func (v *Validations) MarkAsInvalid() {
	*v &^= VFValid
}

// MarkAsValid sets the CFValid bit and marks the Validations as valid
func (v *Validations) MarkAsValid() {
	*v |= VFValid
}

// HasFlag returns true if the type has the flag (or flags) specified
func (v Validations) HasFlag(f Validations) bool {
	return v&f != 0
}

// isValidationsForValidDomain checks if a mask of validations really marks a domain as valid.
func (v Validations) IsValidationsForValidDomain() bool {
	return v.HasFlag(VFMXLookup) || v.HasFlag(VFDomainHasIP) || v.HasFlag(VFHostConnect)
}
