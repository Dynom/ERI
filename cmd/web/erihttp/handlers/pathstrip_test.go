package handlers

import (
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
		// TODO: Add test cases.
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
