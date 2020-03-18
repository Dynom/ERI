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
	Result []string
}

func (c *SuggestSvc) HandleRequest(ctx context.Context, email string) (SuggestResult, error) {
	var sr = SuggestResult{
		Result: []string{email},
	}

	log := c.logger.WithField(handlers.RequestID, ctx.Value(handlers.RequestID))

	parts, err := types.NewEmailParts(email)
	if err != nil {
		return sr, nil
	}

	vr := c.validator(ctx, parts)
	if !vr.HasValidStructure() {
		// No need to run finder, since it can't be a valid address
		log.WithFields(logrus.Fields{
			"email":       email,
			"steps":       vr.Steps.String(),
			"validations": vr.Validations.String(),
		}).Debug("Input doesn't have a valid structure")
		return sr, nil
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
			Result: []string{parts.Address},
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
