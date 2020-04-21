package erihttp

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Dynom/ERI/cmd/web/config"
	"github.com/Dynom/ERI/runtimer"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/netutil"
)

func BuildHTTPServer(mux http.Handler, config config.Config, logger logrus.FieldLogger, logWriter io.Writer, rt *runtimer.SignalHandler, handlers ...func(h http.Handler) http.Handler) *Server {
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

	eriServer := &Server{
		server:   server,
		listener: listener,
	}

	if rt != nil {
		rt.RegisterCallback(func(_ os.Signal) {
			logger.Info("Shutting down web server")
			err := eriServer.Close()
			if err != nil {
				logger.WithError(err).Error("Shutdown error")
			}
		})
	}

	return eriServer
}

type Server struct {
	server   *http.Server
	listener net.Listener
}

func (s *Server) ServeERI() error {
	return s.server.Serve(s.listener)
}

func (s *Server) Close() error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	defer cancel()

	return s.server.Shutdown(ctx)
}
