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
	RequestID = "request_id"
)

func WithRequestLogger(logger *logrus.Logger) HandlerWrapper {
	return func(handler http.Handler) http.Handler {

		var reqID uint64
		m := sync.Mutex{}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			now := time.Now()
			l.Debug("Request start")

			defer func() {
				l.WithField("time_µs", time.Since(now).Microseconds()).Debug("Request stop")
			}()

			handler.ServeHTTP(w, r)
		})
	}
}
