package services

import (
	"context"
	"strings"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewSuggestService(f *finder.Finder, val validator.CheckFn, logger logrus.FieldLogger) *SuggestSvc {
	return &SuggestSvc{
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
	// @todo make this configurable and Algorithm dependent
	const finderThreshold = 0.8

	var emailStrLower = strings.ToLower(email)
	var sr = SuggestResult{
		Alternatives: []string{email},
	}

	log := c.logger.WithFields(logrus.Fields{
		handlers.RequestID.String(): ctx.Value(handlers.RequestID),
		"email":                     emailStrLower,
	})

	parts, partsErr := types.NewEmailParts(emailStrLower)
	if partsErr != nil {
		log.WithError(partsErr).Debug("Unable to split input")
		return sr, validator.ErrEmailAddressSyntax
	}

	if ctx.Err() != nil {
		return sr, ctx.Err()
	}

	var err error
	vr := c.validator(ctx, parts)
	if !vr.HasValidStructure() {
		log.WithFields(logrus.Fields{
			"steps":       vr.Steps.String(),
			"validations": vr.Validations.String(),
		}).Debug("Input doesn't have a valid structure")

		err = validator.ErrEmailAddressSyntax
	}

	if vr.Validations.IsValid() {
		return sr, err
	}

	// No result so far, proceeding with finding domains alternatives
	alt, score, exact := c.finder.FindCtx(ctx, parts.Domain)

	log.WithFields(logrus.Fields{
		"alt":         alt,
		"score":       score,
		"exact":       exact,
		"ctx_expired": didDeadlineExpire(ctx),
	}).Debug("Used Finder")

	if score > finderThreshold {
		parts := types.NewEmailFromParts(parts.Local, alt)
		return SuggestResult{
			Alternatives: []string{parts.Address},
		}, err
	}

	return sr, err
}

func didDeadlineExpire(ctx context.Context) bool {
	return ctx.Err() != nil
}
