package types

import "testing"

func Test_splitLocalAndDomain(t *testing.T) {
	type expect struct {
		local  string
		domain string
	}
	tests := []struct {
		input  string
		expect expect
	}{
		{input: "john@example.org", expect: expect{local: "john", domain: "example.org"}},
		{input: "john.doe@example.org", expect: expect{local: "john.doe", domain: "example.org"}},
	}

	for _, test := range tests {
		d, err := splitLocalAndDomain(test.input)
		if err != nil || d.Local != test.expect.local || d.Domain != test.expect.domain {
			t.Errorf("Expected %+v instead it was %+v (%v)", test.expect, d, err)
		}
	}
}
