package persist

import (
	"context"
	"io"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/validator"
)

type PersistCallbackFn func(d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error

type Persister interface {
	// Store stores the parts and vr. The implementation decides what key to use, although it should use a similar one
	// use to restore data using the Get or Range implementations
	Store(ctx context.Context, d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error

	// Range reads all data back and invokes the callback, until all data is read back, or until the callback returns
	// a non-nil error. The implementation decides on the most optimal strategy.
	Range(ctx context.Context, cb PersistCallbackFn) error

	io.Closer
}
