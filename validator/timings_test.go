package validator

import (
	"testing"
	"time"
)

func TestTimings_Add(t *testing.T) {
	ti := Timings{}
	if len(ti) != 0 {
		t.Errorf("Expected timings to have 0 elements when starting: %+v", ti)
	}

	// Adding with same name
	ti.Add("foo", 1*time.Millisecond)
	ti.Add("foo", 1*time.Minute)

	if len(ti) != 2 {
		t.Errorf("Expecting timings to have 2 elements after adds: %+v", ti)
	}
}
