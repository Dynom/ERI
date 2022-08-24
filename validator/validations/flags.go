package validations

import "strings"

// Validation Flags, these flags represent successful validation steps. Depending on how far you want to go, you can
// classify a validation as valid enough, for your use-case.
const (
	FValid         Flag = 1 << iota
	FSyntax        Flag = 1 << iota
	FMXLookup      Flag = 1 << iota
	FMXDomainHasIP Flag = 1 << iota // Flag set when the MX domain is verified to have at least one resolvable IP
	FHostConnect   Flag = 1 << iota
	FValidRCPT     Flag = 1 << iota
	FDisposable    Flag = 1 << iota // Address / Domain is considered a disposable e-mail trap

	// FDomainHasIP is Deprecated: Unclear naming. Prefer FMXDomainHasIP
	FDomainHasIP = FMXDomainHasIP // @deprecated
)

type Flag uint8

func (f Flag) AsStringSlice() []string {
	var flags = []Flag{FValid, FSyntax, FMXLookup, FMXDomainHasIP, FHostConnect, FValidRCPT, FDisposable}
	var r = make([]string, 0, len(flags))

	for _, flag := range flags {
		if f&flag == 0 {
			continue
		}

		// Remove the flag
		f &^= flag
		r = append(r, toString(flag))
	}

	if f > 0 {
		// List of flags is possibly outdated.
		panic("trouble in paradise")
	}

	return r
}

func (f *Flag) String() string {
	return strings.Join(f.AsStringSlice(), ",")
}

func toString(f Flag) string {
	switch f {
	case FValid:
		return "valid"
	case FSyntax:
		return "syntax"
	case FMXLookup:
		return "lookup"
	case FMXDomainHasIP:
		return "mxDomainHasIP"
	case FHostConnect:
		return "hostConnect"
	case FValidRCPT:
		return "validRecipient"
	case FDisposable:
		return "disposable"
	}

	return "nil"
}
