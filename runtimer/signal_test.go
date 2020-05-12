package runtimer

import (
	"os"
	"testing"
)

func TestSignalHandler_RegisterCallback(t *testing.T) {

	sh := New()

	if got, expect := len(sh.fns), 0; got != expect {
		t.Errorf("RegisterCallback() pre length (%d) doesn't have expected value of %d", got, expect)
	}

	sh.RegisterCallback(func(s os.Signal) {})
	sh.RegisterCallback(func(s os.Signal) {})

	if got, expect := len(sh.fns), 2; got != expect {
		t.Errorf("RegisterCallback() post length (%d) doesn't have expected value of %d", got, expect)
	}
}

func TestSignalHandler_handle(t *testing.T) {

	sh := New(os.Interrupt)

	const expect = 42
	var got uint
	sh.RegisterCallback(func(s os.Signal) {
		got = expect
	})

	// Faking an interrupt
	sh.c <- os.Interrupt

	sh.Wait()

	if got != expect {
		t.Errorf("handle() is expected to invoke all registered callbacks")
	}
}
