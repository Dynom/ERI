package services

import (
	"context"
	"fmt"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/cmd/web/types"
	"github.com/Dynom/TySug/finder"
)

func NewLearnService(cache *hitlist.HitList, f *finder.Finder) LearnSvc {
	return LearnSvc{
		cache:  cache,
		finder: f,
	}
}

type LearnSvc struct {
	cache  *hitlist.HitList
	finder *finder.Finder
}

type LearnResult struct {
	NumDomains        uint64
	NumEmailAddresses uint64
}

func (l *LearnSvc) HandleLearnRequest(ctx context.Context, req erihttp.LearnRequest) (LearnResult, error) {
	var result = LearnResult{
		NumDomains:        uint64(len(req.Domains)),
		NumEmailAddresses: uint64(len(req.Emails)),
	}

	var learnErrors = make(map[string]error, len(req.Emails)+len(req.Domains))
	for _, toLearn := range req.Emails {
		var v types.Validations
		if toLearn.Valid {
			v.MarkAsValid()
		}

		err := l.cache.LearnEmailAddress(toLearn.Value, v)
		if err != nil {
			learnErrors[toLearn.Value] = err
		}
	}

	for _, toLearn := range req.Domains {
		var v types.Validations
		if toLearn.Valid {
			v.MarkAsValid()
		}

		err := l.cache.LearnDomain(toLearn.Value, v)
		if err != nil {
			learnErrors[toLearn.Value] = err
		}
	}

	l.finder.Refresh(l.cache.GetValidAndUsageSortedDomains())

	if len(learnErrors) > 0 {
		return result, fmt.Errorf("had %d errors", len(learnErrors))
	}

	return result, nil
}
