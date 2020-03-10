package services

import (
	"context"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewSuggestService(hitList *hitlist.HitList, f *finder.Finder, val validator.CheckFn, logger *logrus.Logger) SuggestSvc {
	return SuggestSvc{
		hitList:   hitList,
		finder:    f,
		validator: val,
		logger:    logger.WithField("svc", "check"),
	}
}

type SuggestSvc struct {
	hitList   *hitlist.HitList
	finder    *finder.Finder
	validator validator.CheckFn
	logger    *logrus.Entry
}

type SuggestResult struct {
	Alternatives []string
}

func (c *SuggestSvc) HandleRequest(ctx context.Context, email string) (SuggestResult, error) {
	var sr = SuggestResult{
		Alternatives: []string{email},
	}

	log := c.logger.WithField(handlers.RequestID, ctx.Value(handlers.RequestID))

	parts, err := types.NewEmailParts(email)
	if err != nil {
		return sr, nil
	}

	// A direct hit on the domain, passing through
	hit, err := c.hitList.GetForDomain(parts.Domain)
	if err == nil {
		if v := hit.ValidationResult.Validations; v.IsValid() || v.IsValidationsForValidDomain() {
			return sr, nil
		}
	}

	// Validating the argument next
	vr := c.validator(ctx, parts)
	log.WithFields(logrus.Fields{
		"email":       parts.Address,
		"validations": vr.Validations.String(),
		"steps":       vr.Steps.String(),
	}).Debug("Validations ran")

	// Learn of this validation
	if err := c.hitList.AddEmailAddress(email, vr); err == nil && vr.Validations.IsValidationsForValidDomain() {
		c.finder.Refresh(c.hitList.GetValidAndUsageSortedDomains())
	}

	if vr.Validations.IsValid() {
		return sr, nil
	}

	// No result so far, proceeding with finding domains alternatives
	alt, score, exact := c.finder.FindCtx(ctx, parts.Domain)

	log.WithFields(logrus.Fields{
		"alt":         alt,
		"score":       score,
		"exact":       exact,
		"ctx_expired": didDeadlineExpire(ctx),
	}).Debug("Used Finder")

	if score > finder.WorstScoreValue {
		parts, err := types.NewEmailFromParts(parts.Local, alt)
		if err != nil {
			return sr, err
		}

		return SuggestResult{
			Alternatives: []string{parts.Address},
		}, nil
	}

	return sr, nil
}
