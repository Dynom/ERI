package erihttp

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Dynom/ERI/cmd/web/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/netutil"
)

func BuildHTTPServer(mux http.Handler, config config.Config, logger logrus.FieldLogger, logWriter io.Writer, handlers ...func(h http.Handler) http.Handler) *Server {
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

	listener, err := net.Listen("tcp", config.Server.ListenOn)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":     err,
			"listen_on": config.Server.ListenOn,
		}).Error("Unable to start listener")
	}

	if config.Server.ConnectionLimit > 0 {
		listener = netutil.LimitListener(listener, int(config.Server.ConnectionLimit))
	}

	server.RegisterOnShutdown(func() {
		err := listener.Close()
		logger.WithError(err).Debug("Closing listener")
	})

	return &Server{
		server:   server,
		listener: listener,
	}
}

type Server struct {
	server   *http.Server
	listener net.Listener
}

func (s *Server) ServeERI() error {
	return s.server.Serve(s.listener)
}
