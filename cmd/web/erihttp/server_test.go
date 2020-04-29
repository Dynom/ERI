package erihttp

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"

	"github.com/Dynom/ERI/cmd/web/config"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestBuildHTTPServer(t *testing.T) {
	type args struct {
		mux      http.Handler
		config   config.Config
		handlers []func(h http.Handler) http.Handler
	}
	tests := []struct {
		name          string
		args          args
		wantLogWriter string
		want          *http.Server
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logWriter := &bytes.Buffer{}
			l, _ := test.NewNullLogger()

			got := NewServer(tt.args.mux, tt.args.config, l, logWriter, nil, tt.args.handlers...)
			if gotLogWriter := logWriter.String(); gotLogWriter != tt.wantLogWriter {
				t.Errorf("NewServer() gotLogWriter = %v, want %v", gotLogWriter, tt.wantLogWriter)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServer() = %v, want %v", got, tt.want)
			}
		})
	}
}
