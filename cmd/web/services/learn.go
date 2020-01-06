package services

import (
	"context"
	"fmt"
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

type ResultStreamLearnStatus chan LearnStatus

func (rs ResultStreamLearnStatus) Announce(status LearnStatus) {
	rs <- status
}

func NewLearnService(hitList *hitlist.HitList, f *finder.Finder, v validator.CheckFn, logger *logrus.Logger) LearnSvc {
	return LearnSvc{
		hitList:      hitList,
		finder:       f,
		validator:    v,
		logger:       logger,
		ResultStream: make(ResultStreamLearnStatus),
	}
}

type LearnSvc struct {
	hitList      *hitlist.HitList
	finder       *finder.Finder
	validator    validator.CheckFn
	logger       *logrus.Logger
	ResultStream ResultStreamLearnStatus
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
		result.EmailAddressErrors = learnAndAddValue(ctx, l.validator, l.hitList, req.Emails, l.ResultStream, LearnValueEmail)
		wg.Done()
	}()

	go func() {
		result.DomainErrors = learnAndAddValue(ctx, l.validator, l.hitList, req.Domains, l.ResultStream, LearnValueDomain)
		wg.Done()
	}()

	wg.Wait()
	l.finder.Refresh(l.hitList.GetValidAndUsageSortedDomains())

	return result
}

func learnAndAddValue(ctx context.Context, validator validator.CheckFn, hitList *hitlist.HitList, toLearn []erihttp.ToLearn, resultStream ResultStreamLearnStatus, valueType LearnValueType) (failures uint64) {
	for _, learn := range toLearn {
		var v validations.Validations
		var err error
		var ls = LearnStatus{
			Type:        valueType,
			Value:       learn.Value,
			Validations: 0,
			Error:       nil,
		}

		// Aborting operation if we're canceled
		if ctx.Err() != nil {
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
				ls := ls
				ls.Error = fmt.Errorf("unable to split address %w", err)
				resultStream.Announce(ls)
				continue
			}
		}

		// We can assume it's valid, since it was specified as such in the request
		artifact, err := validator(ctx, parts)
		v = artifact.Validations

		if err != nil {
			failures++
			ls := ls
			ls.Error = fmt.Errorf("address is invalid, marking as such %w", err)
			resultStream.Announce(ls)
		}

		var learnFn func(string, validations.Validations) error
		if valueType == LearnValueDomain {
			learnFn = hitList.AddDomain
		} else {
			learnFn = hitList.AddEmailAddress
		}

		err = learnFn(learn.Value, v)
		ls.Validations = v

		if err != nil {
			failures++
			ls.Error = fmt.Errorf("failed learning address %w", err)
		}

		resultStream.Announce(ls)
	}

	return failures
}
