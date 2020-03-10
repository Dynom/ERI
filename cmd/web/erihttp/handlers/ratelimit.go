package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/juju/ratelimit"
	"github.com/sirupsen/logrus"
)

func NewRateLimitHandler(logger logrus.FieldLogger, b *ratelimit.Bucket, maxDelay time.Duration) HandlerWrapper {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logger.WithFields(logrus.Fields{
				"remote_addr": r.RemoteAddr,
				RequestID:     r.Context().Value(RequestID),
				"max_delay":   maxDelay,
			})

			d, ok := b.TakeMaxDuration(1, maxDelay)
			if !ok {
				logger.Warn("rate limit: aborting request, above max allowed delay")

				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = fmt.Fprint(w, "Server busy, request aborted")
				return
			}

			if d > 0 {
				logger.WithField("delay", d).Warn("rate limit: throttling request, will continue after delay")
				time.Sleep(d)
			}

			h.ServeHTTP(w, r)
		})
	}
}
