package handlers

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	testLog "github.com/sirupsen/logrus/hooks/test"
)

func Test_normalizeSlashes(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "All OK", path: "/eri", want: "/eri"},
		{name: "Fixing Suffix", path: "/eri/", want: "/eri"},
		{name: "Fixing Prefix", path: "eri/", want: "/eri"},
		{name: "Fixing Both", path: "eri", want: "/eri"},
	}

	t.Run("Logs", func(t *testing.T) {
		t.Run("Prefix", func(t *testing.T) {
			logger, hook := testLog.NewNullLogger()
			_ = normalizeSlashes(logger, "foo")

			if len(hook.Entries) != 1 {
				t.Errorf("Expected a log message, instead I got %d %+v", len(hook.Entries), hook.Entries)
				return
			}

			if hook.Entries[0].Level != logrus.WarnLevel {
				t.Errorf("Expected warning level messages")
			}
		})

		t.Run("Suffix", func(t *testing.T) {
			logger, hook := testLog.NewNullLogger()
			_ = normalizeSlashes(logger, "/foo/")

			if len(hook.Entries) != 1 {
				t.Errorf("Expected a log message, instead I got %d %+v", len(hook.Entries), hook.Entries)
				return
			}

			if hook.Entries[0].Level != logrus.WarnLevel {
				t.Errorf("Expected warning level messages")
			}
		})
	})

	logger, _ := testLog.NewNullLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSlashes(logger, tt.path); got != tt.want {
				t.Errorf("normalizeSlashes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithPathStrip(t *testing.T) {
	type args struct {
		logger logrus.FieldLogger
		path   string
	}
	tests := []struct {
		name string
		args args
		want func(h http.Handler) http.Handler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithPathStrip(tt.args.logger, tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithPathStrip() = %v, want %v", got, tt.want)
			}
		})
	}
}
