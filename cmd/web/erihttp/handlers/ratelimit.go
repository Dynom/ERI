package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	logRateLimiterDisabled    = "Rate Limiter disabled, no bucket defined."
	logRateLimitThrottled     = "Rate limit: throttling request, will continue after delay"
	logRateLimitAboveMaxDelay = "Rate limit: aborting request, above max allowed delay"
)

type TakeMaxDuration interface {
	TakeMaxDuration(count int64, maxWait time.Duration) (time.Duration, bool)
}

func WithRateLimiter(logger logrus.FieldLogger, b TakeMaxDuration, maxDelay time.Duration) Middleware {
	logger = logger.WithField("middleware", "rate_limiter")

	if b == nil {
		logger.Info(logRateLimiterDisabled)
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			})
		}
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			d, ok := b.TakeMaxDuration(1, maxDelay)

			logger := logger.WithFields(logrus.Fields{
				"remote_addr":      r.RemoteAddr,
				RequestID.String(): r.Context().Value(RequestID),
				"max_delay":        maxDelay.String(),
				"delay":            d.String(),
			})

			if !ok {
				logger.Warn(logRateLimitAboveMaxDelay)

				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = fmt.Fprint(w, "Server busy, request aborted")
				return
			}

			if d > 0 {
				logger.Warn(logRateLimitThrottled)
				time.Sleep(d)
			}

			h.ServeHTTP(w, r)
		})
	}
}
