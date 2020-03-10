package services

import (
	"context"
	"time"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewCheckService(hitList *hitlist.HitList, f *finder.Finder, val validator.CheckFn, logger *logrus.Logger) CheckSvc {
	return CheckSvc{
		hitList:   hitList,
		finder:    f,
		validator: val,
		logger:    logger.WithField("svc", "check"),
	}
}

type CheckSvc struct {
	hitList   *hitlist.HitList
	finder    *finder.Finder
	validator validator.CheckFn
	logger    *logrus.Entry
}

type CheckResult struct {
	Valid             bool
	ValidationsPassed []string `json:"validations_passed"`
	Alternative       string
	CacheHitTTL       time.Duration
}

func (c *CheckSvc) HandleCheckRequest(ctx context.Context, email types.EmailParts, includeAlternatives bool) (CheckResult, error) {

	/*

				1. Check the cache, do we have a validation result?
					 Yes:
						a. Do the validation requirements currently match the validation result?
		           Yes:
					     	Goto #3
							 No:
		           	validate();
			     No:
						validate();
				2. Have we seen this domain before?
		       Yes:
		       	goto #3
			     No:
		       	learnValue();
				3. Should we include alternatives?
					 Yes:
						Do we need alternatives?
						Yes
							findAlternatives();
	*/

	log := c.logger.WithField(handlers.RequestID, ctx.Value(handlers.RequestID))

	// @todo remove logging and include more details in CheckResult

	var res CheckResult

	hit, err := c.hitList.GetForEmail(email.Address)
	if err == nil && hit.ValidationResult.Steps.HasBeenValidated() {
		res.Valid = hit.ValidationResult.Validations.IsValid()
		res.CacheHitTTL = hit.TTL()
		//res.ValidationsPassed = hit.Validations.AsString()

	} else {

		if err != nil && err != hitlist.ErrNotPresent {
			return res, err
		}

		result := c.validator(ctx, email)
		res.Valid = result.Validations.IsValid()
		log.WithError(err).WithField("result", result).Info("Validation result")

		knownDomain := c.hitList.HasDomain(email.Domain)

		// @todo depending on the validations above, we should cache with a different TTL and optionally even b0rk completely here
		err = c.hitList.AddEmailAddress(email.Address, result)
		if err != nil {
			return res, err
		}

		// Update finder with positive results
		if !knownDomain && result.Validations.IsValid() {

			// @todo refresh on interval, to release pressure on finder
			c.finder.Refresh(c.hitList.GetValidAndUsageSortedDomains())
		}
	}

	if includeAlternatives {

		// @todo how to propagate the ctx?
		ctx = context.Background()
		alt, score, exact := c.finder.FindCtx(ctx, email.Domain)

		log.WithFields(logrus.Fields{
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
