package handlers

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	RequestID contextValue = "request_id"
)

type contextValue string

func (cv contextValue) String() string {
	return string(cv)
}

func WithRequestLogger(logger logrus.FieldLogger) HandlerWrapper {
	return func(handler http.Handler) http.Handler {

		var reqID uint64
		m := sync.Mutex{}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var now time.Time
			m.Lock()
			reqID++
			m.Unlock()

			rid := strconv.FormatUint(reqID, 10)

			l := logger.WithFields(logrus.Fields{
				"request_id": rid,
				"method":     r.Method,
				"uri":        r.RequestURI,
			})

			r = r.WithContext(context.WithValue(r.Context(), RequestID, rid))

			l.Debug("Request start")

			defer func() {
				l.WithFields(logrus.Fields{
					"time_µs": time.Since(now).Microseconds(),
				}).Debug("Request stop")
			}()

			now = time.Now()
			handler.ServeHTTP(w, r)
		})
	}
}
