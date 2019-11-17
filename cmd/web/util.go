package main

import (
	"net/http"
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
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Out = os.Stdout
	logger.Level, err = logrus.ParseLevel(conf.Server.Log.Level)

	return logger, err
}
