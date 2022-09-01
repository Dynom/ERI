package services

import (
	"context"
	"errors"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
)

var (
	ErrEmptyInput   = errors.New("input is empty")
	ErrInputTooLong = errors.New("input is too long")
)

func NewAutocompleteService(f *finder.Finder, hitList *hitlist.HitList, recipientThreshold uint64, logger logrus.FieldLogger) *AutocompleteSvc {
	return &AutocompleteSvc{
		finder:             f,
		logger:             logger,
		hitList:            hitList,
		recipientThreshold: recipientThreshold,
	}
}

type AutocompleteSvc struct {
	finder             *finder.Finder
	logger             logrus.FieldLogger
	hitList            *hitlist.HitList
	recipientThreshold uint64
}

type AutocompleteResult struct {
	Suggestions []string
}

func (a *AutocompleteSvc) Autocomplete(ctx context.Context, domain string, limit uint64) (AutocompleteResult, error) {
	if domain == "" {
		return AutocompleteResult{}, ErrEmptyInput
	}

	if len(domain) > 253 {
		return AutocompleteResult{}, ErrInputTooLong
	}

	// Fetching a bit more, to have a greater chance that we're left with enough when we're done with filtering
	list, err := a.finder.GetMatchingPrefix(ctx, domain, uint(limit*2))
	if err != nil {
		return AutocompleteResult{}, err
	}

	filteredList, err := a.filter(ctx, list, limit)
	if err != nil {
		return AutocompleteResult{}, err
	}

	return AutocompleteResult{
		Suggestions: filteredList,
	}, nil
}

func (a *AutocompleteSvc) filter(ctx context.Context, list []string, limit uint64) (filteredList []string, err error) {
	filteredList = make([]string, 0, limit)
	for _, domain := range list {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}

		if cnt := a.hitList.GetRecipientCount(hitlist.Domain(domain)); cnt >= a.recipientThreshold {
			filteredList = append(filteredList, domain)
			if len(filteredList) >= int(limit) {
				break
			}
		}
	}

	return
}
