package services

import (
	"context"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/inspector"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewCheckService(cache *hitlist.HitList, f *finder.Finder, val validator.CheckFn, logger *logrus.Logger) CheckSvc {
	return CheckSvc{
		cache:     cache,
		finder:    f,
		validator: val,
		logger:    logger.WithField("svc", "check"),
	}
}

type CheckSvc struct {
	cache     *hitlist.HitList
	finder    *finder.Finder
	validator validator.CheckFn
	logger    *logrus.Entry
}

type CheckResult struct {
	Valid       bool
	Alternative string
	CacheHitTTL time.Duration
}

func (c *CheckSvc) HandleCheckRequest(ctx context.Context, email types.EmailParts, includeAlternatives bool) (CheckResult, error) {

	// @todo remove logging and include more details in CheckResult

	var res CheckResult
	var result inspector.Result

	hit, err := c.cache.GetForEmail(email.Address)
	if err == nil {
		res.Valid = result.Validations.MergeWithNext(hit.Validations).IsValid()
		res.CacheHitTTL = hit.TTL()

	} else {
		if err != hitlist.ErrNotPresent {
			return res, err
		}

		result, err := c.validator(ctx, email)
		res.Valid = result.Validations.IsValid()
		c.logger.WithContext(ctx).WithError(err).WithField("result", result).Info("Validation result")

		// @todo depending on the validations above, we should cache with a different TTL and optionally even b0rk completely here
		err = c.cache.AddEmailAddress(email.Address, result.Validations)
		if err != nil {
			return res, err
		}

		// Update finder with positive results
		if result.Validations.IsValid() {
			c.finder.Refresh(c.cache.GetValidAndUsageSortedDomains())
		}
	}

	if includeAlternatives {
		ctx = context.Background()
		alt, score, exact := c.finder.FindCtx(ctx, email.Domain)

		c.logger.WithContext(ctx).WithFields(logrus.Fields{
			"alt":              alt,
			"score":            score,
			"exact":            exact,
			"deadline_expired": didDeadlineExpire(ctx),
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

func didDeadlineExpire(ctx context.Context) bool {
	if t, set := ctx.Deadline(); set {
		return t.After(time.Now())
	}

	return false
}
