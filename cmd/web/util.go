package main

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/config"
)

func sliceToHTTPHeaders(slice []config.Header) http.Header {
	headers := http.Header{}
	for _, h := range slice {
		headers.Add(h.Name, h.Value)
	}

	return headers
}

func newLogger(conf config.Config) (*logrus.Logger, error) {
	var err error
	logger := logrus.New()
	//logger.Formatter = &logrus.JSONFormatter{}
	logger.Formatter = &logrus.TextFormatter{}
	logger.Out = os.Stdout
	logger.Level, err = logrus.ParseLevel(conf.Server.Log.Level)

	return logger, err
}

func configureProfiler(mux *http.ServeMux, conf config.Config) {
	var prefix string
	if conf.Server.Profiler.Prefix != "" {
		prefix = conf.Server.Profiler.Prefix
	} else {
		prefix = "debug"
	}

	mux.HandleFunc(`/`+prefix+`/pprof/`, pprof.Index)
	mux.HandleFunc(`/`+prefix+`/pprof/cmdline`, pprof.Cmdline)
	mux.HandleFunc(`/`+prefix+`/pprof/profile`, pprof.Profile)
	mux.HandleFunc(`/`+prefix+`/pprof/symbol`, pprof.Symbol)
	mux.HandleFunc(`/`+prefix+`/pprof/trace`, pprof.Trace)
}

func setCustomResolver(dialer *net.Dialer, host string) {
	if dialer.Resolver == nil {
		dialer.Resolver = &net.Resolver{
			PreferGo: true,
		}
	}

	dialer.Resolver.Dial = func(ctx context.Context, network, address string) (conn net.Conn, e error) {
		d := net.Dialer{}
		return d.DialContext(ctx, network, net.JoinHostPort(host, `53`))
	}
}
