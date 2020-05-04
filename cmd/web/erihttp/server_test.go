package erihttp

import (
	"bytes"
	"net"
	"net/http"
	"testing"

	"github.com/Dynom/ERI/cmd/web/config"
	"github.com/Dynom/ERI/runtimer"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestNewServer(t *testing.T) {
	mux := http.NewServeMux()
	type args struct {
		config   config.Config
		rt       *runtimer.SignalHandler
		handlers []func(h http.Handler) http.Handler
	}

	duration := config.Duration{}
	duration.Set("1s")

	basicServerCfg := config.Config{}
	basicServerCfg.Server.NetTTL = duration
	basicServerCfg.Server.ListenOn = "127.0.0.4321"
	basicServerCfg.Server.ConnectionLimit = 1

	tests := []struct {
		name          string
		args          args
		wantLogWriter string
		want          *Server
	}{
		{
			name: "happy flow",
			args: args{
				config:   basicServerCfg,
				rt:       nil,
				handlers: nil,
			},
			want: &Server{
				server:   &http.Server{},
				listener: &net.TCPListener{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			logWriter := &bytes.Buffer{}

			got := NewServer(mux, tt.args.config, logger, logWriter, tt.args.rt, tt.args.handlers...)
			if gotLogWriter := logWriter.String(); gotLogWriter != tt.wantLogWriter {
				t.Errorf("NewServer() gotLogWriter = %v, want %v", gotLogWriter, tt.wantLogWriter)
			}

			if got.server.Addr != tt.args.config.Server.ListenOn {
				t.Errorf("Bad config propagation, expected: %q got: %q\ndetails: %+v", tt.args.config.Server.ListenOn, got.server.Addr, got.server)
			}
		})
	}
}
