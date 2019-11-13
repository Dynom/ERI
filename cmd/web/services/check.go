package services

import (
	"context"
	"time"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/inspector"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewCheckService(cache *hitlist.HitList, f *finder.Finder, checker inspector.Checker) Check {
	return Check{
		cache:   cache,
		finder:  f,
		checker: checker,
	}
}

type Check struct {
	cache   *hitlist.HitList
	finder  *finder.Finder
	checker inspector.Checker
}

type CheckResult struct {
	Valid       bool
	Alternative string
	CacheHitTTL time.Duration
}

/*
	1. Check the cache, reply with the result if not a miss
	2. Perform the check
	3. If alternative were requested, invoke Finder for the domain
	4. Reply
*/
func (c *Check) HandleCheckRequest(ctx context.Context, email types.EmailParts, includeAlternatives bool) (CheckResult, error) {
	var res CheckResult
	var result inspector.Result
	var now = time.Now()

	l, err := c.cache.GetForEmail(email.Address)
	if err == nil {
		res.Valid = result.Validations.Merge(l.Validations).IsValid()
		res.CacheHitTTL = l.ValidUntil.Sub(now)

	} else {
		if err != hitlist.ErrNotPresent {
			return res, err
		}

		result = c.checker.Check(ctx, email.Address)
		res.Valid = result.Validations.IsValid()

		// @todo not sure if this should result in returning an error
		err := c.cache.LearnEmailAddress(email.Address, result.Validations)
		if err != nil {
			return res, err
		}

		// Update finder with positive results
		if result.Validations.IsValid() {
			c.finder.Refresh(c.cache.GetValidAndUsageSortedDomains())
		}
	}

	if includeAlternatives {
		alt, score, exact := c.finder.FindCtx(ctx, email.Domain)
		if !exact && score > finder.WorstScoreValue {
			parts, err := types.NewEmailFromParts(email.Local, alt)
			if err != nil {
				return res, err
			}

			res.Alternative = parts.Address
		}
	}

	return res, nil
}
