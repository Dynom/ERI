package commands

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type ReportStats struct {
	Passed   uint64 `json:"passed"`
	Rejected uint64 `json:"rejected"`
	Duration int64  `json:"run_duration_ms"`
}

type CheckResultFull struct {
	Input   string   `json:"input"`
	Valid   bool     `json:"valid"`
	Checks  []string `json:"checks_run"`
	Passed  []string `json:"checks_passed"`
	Version uint     `json:"version"`
}

func (c CheckResultFull) String() string {
	var result = new(strings.Builder)
	var err error

	f := func(format string, arg ...interface{}) {
		if err != nil {
			return
		}

		_, err = fmt.Fprintf(result, format, arg...)
	}

	var valid = "invalid"
	if c.Valid {
		valid = "valid"
	}

	f("%-7s ", valid)
	f("Checks:%-27s ", fmt.Sprintf("%+v", c.Checks))
	f("Passed:%-27s ", fmt.Sprintf("%+v", c.Passed))
	f("Version:%d ", c.Version)

	f("%s", c.Input)

	return result.String()
}

type CheckSettings struct {
	Format  string
	CSV     csvOptions
	Check   checkOptions
	Workers uint64
}

type checkOptions struct {
	Resolver            net.IP
	TTL                 time.Duration
	InputIsEmailAddress bool
}

type csvOptions struct {
	skipRows uint64
	column   uint64
}
