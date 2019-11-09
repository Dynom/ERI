package inspector

import (
	"context"
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*3000)

	v := New(WithValidators(
		ValidateSyntax(),
		ValidateMXAndRCPT(DefaultRecipient),
	))

	result := v.Check(ctx, "mark@grr.la")

	if result.Error != nil {
		t.Errorf("Error: %+v", result.Error)
	}

	t.Logf("Result is: %+v", result)
}

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
		if d.partLocal != test.expect.local || d.partDomain != test.expect.domain {
			t.Errorf("Expected %+v instead it was %+v (%v)", test.expect, d, err)
		}
	}
}
