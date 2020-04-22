package handlers

import (
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewRateLimitHandler(t *testing.T) {
	type args struct {
		logger   logrus.FieldLogger
		b        TakeMaxDuration
		maxDelay time.Duration
	}
	tests := []struct {
		name string
		args args
		want Middleware
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRateLimitHandler(tt.args.logger, tt.args.b, tt.args.maxDelay); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRateLimitHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
