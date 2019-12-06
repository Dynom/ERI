package services

import (
	"context"
	"fmt"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/TySug/finder"
)

func NewLearnService(cache *hitlist.HitList, f *finder.Finder, logger *logrus.Logger) LearnSvc {
	return LearnSvc{
		cache:  cache,
		finder: f,
		logger: logger,
	}
}

type LearnSvc struct {
	cache  *hitlist.HitList
	finder *finder.Finder
	logger *logrus.Logger
}

type LearnResult struct {
	NumDomains         uint64
	NumEmailAddresses  uint64
	DomainErrors       uint64
	EmailAddressErrors uint64
}

func (l *LearnSvc) HandleLearnRequest(ctx context.Context, req erihttp.LearnRequest) (LearnResult, error) {
	var result = LearnResult{
		NumDomains:        uint64(len(req.Domains)),
		NumEmailAddresses: uint64(len(req.Emails)),
	}

	var emailLearnErrors uint64
	for _, toLearn := range req.Emails {
		var v validations.Validations
		if toLearn.Valid {
			v.MarkAsValid()
		}

		err := l.cache.LearnEmailAddress(toLearn.Value, v)
		if err != nil {
			l.logger.WithError(err).WithField("value", toLearn.Value).Error("failed learning address")
			emailLearnErrors++
		}
	}

	var domainLearnErrors uint64
	for _, toLearn := range req.Domains {
		var v validations.Validations
		if toLearn.Valid {
			v.MarkAsValid()
		}

		err := l.cache.LearnDomain(toLearn.Value, v)
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
