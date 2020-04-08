package validations

const (
	// Validation Flags, these flags represent successful validation steps. Depending on how far you want to go, you can
	// classify a validation as valid enough, for your use-case.
	FValid       Flag = 1 << iota
	FSyntax      Flag = 1 << iota
	FMXLookup    Flag = 1 << iota
	FDomainHasIP Flag = 1 << iota
	FHostConnect Flag = 1 << iota
	FValidRCPT   Flag = 1 << iota
	FDisposable  Flag = 1 << iota // Address / Domain is considered a disposable e-mail trap
)

type Flag uint8
