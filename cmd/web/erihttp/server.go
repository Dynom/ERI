package erihttp

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Dynom/ERI/cmd/web/config"
)

var (
	ErrMissingBody    = errors.New("missing body")
	ErrInvalidRequest = errors.New("request is invalid")
	ErrBodyTooLarge   = errors.New("request body too large")
)

func BuildHTTPServer(mux http.Handler, config config.Config, logWriter io.Writer, handlers ...func(h http.Handler) http.Handler) *http.Server {
	for _, h := range handlers {
		mux = h(mux)
	}

	wTTL := 10 * time.Second
	if config.Server.Profiler.Enable {
		wTTL = 31 * time.Second
	}

	server := &http.Server{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       wTTL,
		WriteTimeout:      wTTL, // Is overridden, when the profiler is enabled.
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 19, // 512 kb
		Handler:           mux,
		Addr:              config.Server.ListenOn,
		ErrorLog:          log.New(logWriter, "", 0),
	}

	return server
}
