package persist

import (
	"context"
	"sync"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
)

func NewMemory(list *hitlist.HitList) Persister {
	return &Memory{
		m:    &sync.Map{},
		list: list,
	}
}

type Memory struct {
	m    *sync.Map
	list *hitlist.HitList
}

func (s Memory) Store(ctx context.Context, d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.m.Store(string(r)+`@`+string(d), vr)
	return nil
}

func (s Memory) Range(_ context.Context, cb PersistCallbackFn) error {
	s.m.Range(func(key, value interface{}) bool {
		internalParts, err := types.NewEmailParts(key.(string))

		if err != nil {
			return true // Ignoring non-recoverable problem
		}

		vr, ok := value.(validator.Result)

		if !ok {
			return true // Ignoring non-recoverable problem
		}

		domain := hitlist.Domain(internalParts.Domain)
		recipient := hitlist.Recipient(internalParts.Local)

		err = cb(domain, recipient, vr)
		return err == nil
	})

	return nil
}
