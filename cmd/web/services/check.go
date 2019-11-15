package services

import (
	"context"
	"time"

	"github.com/Dynom/ERI/cmd/web/inspector/validators"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/inspector"
	"github.com/Dynom/ERI/cmd/web/types"
	"github.com/Dynom/TySug/finder"
)

func NewCheckService(cache *hitlist.HitList, f *finder.Finder, checker inspector.Checker, logger *logrus.Logger) CheckSvc {
	return CheckSvc{
		cache:   cache,
		finder:  f,
		checker: checker,
		logger:  logger.WithField("svc", "check"),
	}
}

type CheckSvc struct {
	cache   *hitlist.HitList
	finder  *finder.Finder
	checker inspector.Checker
	logger  *logrus.Entry
}

type CheckResult struct {
	Valid       bool
	Alternative string
	CacheHitTTL time.Duration
}

func (c *CheckSvc) HandleCheckRequest(ctx context.Context, email types.EmailParts, includeAlternatives bool) (CheckResult, error) {
	var res CheckResult
	var result validators.Result
	var now = time.Now()

	l, err := c.cache.GetForEmail(email.Address)
	if err == nil {
		res.Valid = result.Validations.MergeWithNext(l.Validations).IsValid()
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

		c.logger.WithContext(ctx).WithFields(logrus.Fields{
			"alt":   alt,
			"score": score,
			"exact": exact,
		}).Debug("Used Finder")

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
