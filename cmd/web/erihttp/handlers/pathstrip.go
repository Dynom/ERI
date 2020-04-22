package handlers

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// WithPathStrip strips the path from the request URL, paths always start with a /.
func WithPathStrip(logger logrus.FieldLogger, path string) func(h http.Handler) http.Handler {
	logger = logger.WithField("middleware", "path_strip")

	if path == "" {
		logger.Warn("Path strip is used with empty path argument, returning an empty handler")
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			})
		}
	}

	path = normalizeSlashes(logger, path)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, path)

			h.ServeHTTP(w, r)
		})
	}
}

// normalizeSlashes makes sure the path starts with a `/` and doesn't end with a `/`
func normalizeSlashes(logger logrus.FieldLogger, path string) string {
	if !strings.HasPrefix(path, `/`) {
		original := path
		path = `/` + path
		logger.WithFields(logrus.Fields{
			"from": original,
			"to":   path,
		}).Warn("The argument to Path strip doesn't start with a `/`, auto correcting to prevent miss-matches")
	}

	// Make sure paths don't end with a /, since that will limit matching
	if strings.HasSuffix(path, `/`) {
		original := path
		path = path[:len(path)-1]
		logger.WithFields(logrus.Fields{
			"from": original,
			"to":   path,
		}).Warn("The argument to Path strip ends with a `/`, auto correcting to prevent miss-matches")
	}

	return path
}
