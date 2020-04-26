package commands

import (
	"net"
	"time"
)

type ReportStats struct {
	Passed   uint64 `json:"passed"`
	Rejected uint64 `json:"rejected"`
	Duration int64  `json:"run_duration_ms"`
}

type CheckResultFull struct {
	Email   string   `json:"email"`
	Valid   bool     `json:"valid"`
	Checks  []string `json:"checks_run"`
	Passed  []string `json:"checks_passed"`
	Version int      `json:"version"`
}

type CheckSettings struct {
	Format string
	CSV    csvOptions
	Check  checkOptions
}

type checkOptions struct {
	Resolver net.IP
	TTL      time.Duration
}

type csvOptions struct {
	skipRows uint64
	column   uint64
}
