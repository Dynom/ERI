package services

import (
	"context"
	"fmt"

	"github.com/Dynom/ERI/types"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/TySug/finder"
)

func NewLearnService(cache *hitlist.HitList, f *finder.Finder, v *validator.EmailValidator, logger *logrus.Logger) LearnSvc {
	return LearnSvc{
		cache:     cache,
		finder:    f,
		validator: v,
		logger:    logger,
	}
}

type LearnSvc struct {
	cache     *hitlist.HitList
	finder    *finder.Finder
	validator *validator.EmailValidator
	logger    *logrus.Logger
}

type LearnResult struct {
	NumDomains         uint64
	NumEmailAddresses  uint64
	DomainErrors       uint64
	EmailAddressErrors uint64
}

// HandleLearnRequest learns of the existence of a domain or e-mail address. It's designed to handle bulk requests and
// respects validity markers if specified. It won't check for existing values so that they can be overwritten.
// @todo figure out how a Learn Request should work
func (l *LearnSvc) HandleLearnRequest(ctx context.Context, req erihttp.LearnRequest) (LearnResult, error) {
	var result = LearnResult{
		NumDomains:        uint64(len(req.Domains)),
		NumEmailAddresses: uint64(len(req.Emails)),
	}

	var emailLearnErrors uint64
	for _, toLearn := range req.Emails {
		var v validations.Validations
		var err error

		// Aborting operation once we're cancelled
		if ctx.Err() != nil {
			break
		}

		parts, err := types.NewEmailParts(toLearn.Value)
		if err != nil {
			emailLearnErrors++
			l.logger.WithError(err).WithField("value", toLearn.Value).Error("unable to split address")
			continue
		}

		// We can assume it's valid, since it was specified as such in the request
		if toLearn.Valid {
			v.MarkAsValid()
		} else {
			artifact, err := l.validator.CheckWithLookup(ctx, parts)

			if err != nil {
				emailLearnErrors++
				l.logger.WithFields(logrus.Fields{
					"value": toLearn.Value,
					"error": err,
				}).Error("address is invalid, marking as such")
			}

			v = artifact.Validations
		}

		l.logger.WithFields(logrus.Fields{
			"value":       toLearn.Value,
			"validations": fmt.Sprintf("%08b", v),
		}).Debug("Adding email address")
		err = l.cache.AddEmailAddress(toLearn.Value, v)
		if err != nil {
			l.logger.WithError(err).WithField("value", toLearn.Value).Error("failed learning address")
			emailLearnErrors++
		}
	}

	var domainLearnErrors uint64
	for _, toLearn := range req.Domains {
		var v validations.Validations
		var err error

		// Aborting operation once we're cancelled
		if ctx.Err() != nil {
			break
		}

		parts := types.EmailParts{
			Address: "",
			Local:   "",
			Domain:  toLearn.Value,
		}

		if toLearn.Valid {
			v.MarkAsValid()
		} else {
			artifact, err := l.validator.CheckDomainWithLookup(ctx, parts)

			if err != nil {
				domainLearnErrors++
				l.logger.WithFields(logrus.Fields{
					"value": toLearn.Value,
					"error": err,
				}).Info("domain is invalid, marking as such")
			}

			v = artifact.Validations
		}

		l.logger.WithFields(logrus.Fields{
			"value":       toLearn.Value,
			"validations": fmt.Sprintf("%08b", v),
		}).Debug("Adding domain")
		err = l.cache.AddDomain(toLearn.Value, v)
		if err != nil {
			l.logger.WithError(err).WithField("value", toLearn.Value).Error("failed learning domain")
			domainLearnErrors++
		}
	}

	result.DomainErrors = domainLearnErrors
	result.EmailAddressErrors = emailLearnErrors

	l.finder.Refresh(l.cache.GetValidAndUsageSortedDomains())

	if emailLearnErrors > 0 || domainLearnErrors > 0 {
		return result, fmt.Errorf("had %d errors", emailLearnErrors+domainLearnErrors)
	}

	return result, nil
}
