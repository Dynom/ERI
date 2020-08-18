package services

import (
	"context"
	"strings"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"
	"github.com/Dynom/ERI/cmd/web/preferrer"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/TySug/finder"
)

func NewSuggestService(f *finder.Finder, val validator.CheckFn, prefer preferrer.HasPreferred, logger logrus.FieldLogger) *SuggestSvc {
	if prefer == nil {
		prefer = preferrer.New(nil)
	}

	return &SuggestSvc{
		finder:    f,
		validator: val,
		logger:    logger.WithField("svc", "suggest"),
		prefer:    prefer,
	}
}

type SuggestSvc struct {
	finder    *finder.Finder
	validator validator.CheckFn
	logger    *logrus.Entry
	prefer    preferrer.HasPreferred
}

type SuggestResult struct {
	Alternatives []string
}

// @todo make this configurable and Algorithm dependent
const finderThreshold = 0.8

func (c *SuggestSvc) Suggest(ctx context.Context, email string) (SuggestResult, error) {

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

	if !vr.Validations.IsValid() {

		// No result so far, proceeding with finding domain alternatives
		alts := c.getAlternatives(ctx, parts)
		if len(alts) > 0 {
			sr.Alternatives = alts
		}
	}

	var alts = make([]string, 0, len(sr.Alternatives))
	for _, alt := range sr.Alternatives {
		parts, err := types.NewEmailParts(alt)
		if err != nil {
			log.WithError(err).Error("Input doesn't have valid structure")
			continue
		}

		if preferred, exists := c.prefer.HasPreferred(parts); exists {
			parts := types.NewEmailFromParts(parts.Local, preferred)
			alts = append(alts, parts.Address, alt)
		} else {
			alts = append(alts, alt)
		}
	}

	sr.Alternatives = alts

	return sr, err
}

func (c *SuggestSvc) getAlternatives(ctx context.Context, parts types.EmailParts) []string {

	alt, score, exact := c.finder.FindCtx(ctx, parts.Domain)

	c.logger.WithFields(logrus.Fields{
		handlers.RequestID.String(): ctx.Value(handlers.RequestID),
		"alt":                       alt,
		"score":                     score,
		"threshold_met":             score > finderThreshold,
		"exact":                     exact,
		"ctx_expired":               didDeadlineExpire(ctx),
	}).Debug("Used Finder")

	if score > finderThreshold {
		parts = types.NewEmailFromParts(parts.Local, alt)
	}

	return []string{parts.Address}
}

func didDeadlineExpire(ctx context.Context) bool {
	return ctx.Err() != nil
}
