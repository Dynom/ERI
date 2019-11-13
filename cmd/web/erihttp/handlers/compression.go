package handlers

import (
	"compress/gzip"
	"net/http"

	"github.com/NYTimes/gziphandler"
)

const (
	mtuSize = 1500
)

func WithGzipHandler() HandlerWrapper {
	return func(handler http.Handler) http.Handler {
		wrapper, _ := gziphandler.GzipHandlerWithOpts(
			gziphandler.CompressionLevel(gzip.BestCompression),
			gziphandler.MinSize(mtuSize),
		)

		return wrapper(handler)
	}
}
