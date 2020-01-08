package services

import (
	"context"
	"sync"

	"github.com/Dynom/ERI/types"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/TySug/finder"
)

const (
	LearnValueEmail  = "email address"
	LearnValueDomain = "domain"
)

type LearnValueType string

func NewLearnService(hitList *hitlist.HitList, f *finder.Finder, v validator.CheckFn, logger *logrus.Logger) LearnSvc {
	return LearnSvc{
		hitList:   hitList,
		finder:    f,
		validator: v,
		logger:    logger,
	}
}

type LearnSvc struct {
	hitList   *hitlist.HitList
	finder    *finder.Finder
	validator validator.CheckFn
	logger    *logrus.Logger
}

type LearnResult struct {
	NumDomains         uint64
	NumEmailAddresses  uint64
	DomainErrors       uint64
	EmailAddressErrors uint64
}

type LearnStatus struct {
	Type        LearnValueType
	Value       string
	Validations validations.Validations
	Error       error
}

// HandleLearnRequest learns of the existence of a domain or e-mail address. It's designed to handle bulk requests
// @todo figure out how a Learn Request should work
func (l *LearnSvc) HandleLearnRequest(ctx context.Context, req erihttp.LearnRequest) LearnResult {
	var result = LearnResult{
		NumDomains:        uint64(len(req.Domains)),
		NumEmailAddresses: uint64(len(req.Emails)),
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		if len(req.Emails) > 0 {
			result.EmailAddressErrors = l.learnAndAddValue(ctx, req.Emails, LearnValueEmail)
		}
		wg.Done()
	}()

	go func() {
		if len(req.Domains) > 0 {
			result.DomainErrors = l.learnAndAddValue(ctx, req.Domains, LearnValueDomain)
		}
		wg.Done()
	}()

	wg.Wait()
	l.finder.Refresh(l.hitList.GetValidAndUsageSortedDomains())

	return result
}

func (l *LearnSvc) learnAndAddValue(ctx context.Context, toLearn []erihttp.ToLearn, valueType LearnValueType) (failures uint64) {
	logger := l.logger.WithFields(logrus.Fields{
		"method": "learnAndAddValue",
		"type":   valueType,
	})

	for _, learn := range toLearn {
		logger := logger.WithFields(logrus.Fields{
			"value":            learn.Value,
			"considered_valid": learn.Valid,
		})

		var v validations.Validations
		var err error

		// Aborting operation if we're canceled
		if ctx.Err() != nil {
			logger.Debug("Context canceled")
			return failures
		}

		var parts types.EmailParts
		if valueType == LearnValueDomain {
			parts = types.EmailParts{
				Address: "",
				Local:   "",
				Domain:  learn.Value,
			}
		} else {
			parts, err = types.NewEmailParts(learn.Value)
			if err != nil {
				failures++
				logger.WithError(err).Debug("unable to parse e-mail address")
				continue
			}
		}

		artifact, err := l.validator(ctx, parts)
		v = artifact.Validations

		logger = logger.WithField("validations", v)

		if err != nil {
			failures++
		}

		var learnFn func(string, validations.Validations) error
		if valueType == LearnValueDomain {
			learnFn = l.hitList.AddDomain
		} else {
			learnFn = l.hitList.AddEmailAddress
		}

		err = learnFn(learn.Value, v)

		if err != nil {
			failures++
			logger.WithError(err).Debug("unable to learn value")
			continue
		}

		logger.Debug("learned value")
	}

	return failures
}
