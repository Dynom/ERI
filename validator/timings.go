package validator

import "time"

type Timings []Timing

func (t *Timings) Add(l string, d time.Duration) {
	*t = append(*t, Timing{Label: l, Duration: d})
}

type Timing struct {
	Label    string
	Duration time.Duration
}
