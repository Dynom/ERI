package services

import (
	"context"
	"time"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewSuggestService(f *finder.Finder, val validator.CheckFn, logger *logrus.Logger) SuggestSvc {
	return SuggestSvc{
		finder:    f,
		validator: val,
		logger:    logger.WithField("svc", "suggest"),
	}
}

type SuggestSvc struct {
	finder    *finder.Finder
	validator validator.CheckFn
	logger    *logrus.Entry
}

type SuggestResult struct {
	Alternatives []string
}

func (c *SuggestSvc) Suggest(ctx context.Context, email string) (SuggestResult, error) {
	var sr = SuggestResult{
		Alternatives: []string{email},
	}

	log := c.logger.WithFields(logrus.Fields{
		handlers.RequestID.String(): ctx.Value(handlers.RequestID),
		"email":                     email,
	})

	parts, err := types.NewEmailParts(email)
	if err != nil {
		log.WithError(err).Debug("Unable to split input")
		return sr, validator.ErrEmailAddressSyntax
	}

	vr := c.validator(ctx, parts)
	if !vr.HasValidStructure() {
		log.WithFields(logrus.Fields{
			"steps":       vr.Steps.String(),
			"validations": vr.Validations.String(),
		}).Debug("Input doesn't have a valid structure")
		return sr, validator.ErrEmailAddressSyntax
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
		parts := types.NewEmailFromParts(parts.Local, alt)
		return SuggestResult{
			Alternatives: []string{parts.Address},
		}, nil
	}

	return sr, nil
}

func didDeadlineExpire(ctx context.Context) bool {
	if t, set := ctx.Deadline(); set {
		return t.After(time.Now())
	}

	return false
}
