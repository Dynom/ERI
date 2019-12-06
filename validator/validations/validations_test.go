package validations

import (
	"testing"
)

func TestValidations_HasFlag(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		tf   Validations
		want bool
	}{
		{want: true, name: "has flag", v: VFValid, tf: VFValid},
		{want: true, name: "has flag (multiple)", v: VFValid | VFDomainHasIP, tf: VFValid},
		{want: true, name: "has flag (multiple)", v: VFSyntax | VFDomainHasIP, tf: VFDomainHasIP},

		{name: "doesn't have flag", v: 0, tf: VFValid},
		{name: "doesn't have flag", v: VFDomainHasIP, tf: VFValid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.HasFlag(tt.tf); got != tt.want {
				t.Errorf("HasFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_IsValid(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		want bool
	}{
		{want: true, name: "mega valid", v: VFValid},
		{name: "default value", v: 0},
		{name: "some flags", v: VFSyntax | VFDomainHasIP},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidations_IsValidationsForValidDomain(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		want bool
	}{
		{want: true, name: "All domain flags", v: VFHostConnect | VFMXLookup | VFDomainHasIP},
		{want: true, name: "Domain has IP", v: VFDomainHasIP},
		{want: true, name: "Domain accepted connections", v: VFHostConnect},
		{want: true, name: "DNS lookup showed domain has MX records", v: VFMXLookup},

		// Valid doesn't mean the domain is actually valid (e.g. we might've only performed a syntax check)
		{name: "valid, doesn't mean valid domain", v: VFValid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.IsValidationsForValidDomain(); got != tt.want {
				t.Errorf("IsValidationsForValidDomain() = %v, want %v (%08b)", got, tt.want, tt.v)
			}
		})
	}
}

func TestValidations_MarkAsInvalid(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
	}{
		{name: "Starting as valid", v: VFValid},
		{name: "Starting as invalid", v: 0},
		{name: "Various flags, invalid", v: VFHostConnect | VFMXLookup},
		{name: "Various flags, valid", v: VFValid | VFHostConnect | VFMXLookup | VFValidRCPT | VFMXLookup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.v
			v.MarkAsInvalid()

			if got := v.IsValid(); got != false {
				t.Errorf("Expected v.MarkAsInvalid to always result in invalid validations %08b", v)
			}
		})
	}
}

func TestValidations_MarkAsValid(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
	}{
		{name: "Starting as valid", v: VFValid},
		{name: "Starting as invalid", v: 0},
		{name: "Various flags, invalid", v: VFHostConnect | VFMXLookup},
		{name: "Various flags, valid", v: VFValid | VFHostConnect | VFMXLookup | VFValidRCPT | VFMXLookup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.v
			v.MarkAsValid()

			if got := v.IsValid(); got != true {
				t.Errorf("Expected v.MarkAsValid to always result in valid validations %08b", v)
			}
		})
	}
}

func TestValidations_MergeWithNext(t *testing.T) {
	tests := []struct {
		name string
		v    Validations
		new  Validations
		want Validations
	}{
		{name: "single flag", v: 0, new: VFValid, want: VFValid},
		{name: "multiple flags, start from 0", v: 0, new: VFMXLookup | VFHostConnect, want: VFMXLookup | VFHostConnect},
		{name: "single flag, start from VFMXLookup", v: VFMXLookup, new: VFHostConnect, want: VFMXLookup | VFHostConnect},
		{name: "multiple flags, start from VFMXLookup", v: VFMXLookup, new: VFMXLookup | VFHostConnect, want: VFMXLookup | VFHostConnect},

		// MergeWithNext() assumes Full validations as arguments and assumes an incremental chain of validations
		// it unset's the validity of the existing validations.
		{name: "multiple flags, start from VFValid", v: VFValid, new: VFHostConnect, want: VFHostConnect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MergeWithNext(tt.new); got != tt.want {
				t.Errorf("MergeWithNext()\n%08b (%d) got\n%08b (%d) want", got, got, tt.want, tt.want)
			}
		})
	}
}

func Test_MaskTest(t *testing.T) {

	t.Run("Flag values", func(t *testing.T) {
		t.Logf("VFValid        %08b %d", VFValid, VFValid)
		t.Logf("VFSyntax       %08b %d", VFSyntax, VFSyntax)
		t.Logf("VFMXLookup     %08b %d", VFMXLookup, VFMXLookup)
		t.Logf("VFHostConnect  %08b %d", VFHostConnect, VFHostConnect)
		t.Logf("VFValidRCPT    %08b %d", VFValidRCPT, VFValidRCPT)
		t.Logf("VFDisposable   %08b %d", VFDisposable, VFDisposable)
	})

	t.Run("Setting", func(t *testing.T) {
		var v Validations

		t.Logf("initial      %08b", v)
		// Setting MX?
		//v = 1 | VFMXLookup
		v |= VFMXLookup
		t.Logf("set mx?      %08b", v)
		//t.Logf("is valid? %08b", v&VFValid)

		v |= VFValid
		t.Logf("valid masked %08b", v&VFValid)
		t.Logf("is valid?    %t", v&VFValid == VFValid)

		v = 0 &^ VFValid
		t.Logf("valid cleared %08b", v)
		t.Logf("is valid?    %t", v&VFValid == VFValid)

	})
}
