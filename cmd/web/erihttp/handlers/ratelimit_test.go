package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	testLog "github.com/sirupsen/logrus/hooks/test"
)

func TestNewRateLimitHandler(t *testing.T) {
	type args struct {
		b TakeMaxDuration
	}
	tests := []struct {
		name               string
		args               args
		wantHTTPStatusCode int
		wantLogMessage     string
	}{
		{
			name: "All good",
			args: args{
				b: &takeMaxDurationStub{
					withinThreshold: true,
				},
			},
			wantHTTPStatusCode: http.StatusOK,
		},
		{
			name: "Rate limited, within threshold",
			args: args{
				b: &takeMaxDurationStub{
					delay:           time.Nanosecond,
					withinThreshold: true,
				},
			},
			wantHTTPStatusCode: http.StatusOK,
			wantLogMessage:     logRateLimitThrottled,
		},
		{
			name: "Rate limited, outside threshold",
			args: args{
				b: &takeMaxDurationStub{
					withinThreshold: false,
				},
			},
			wantHTTPStatusCode: http.StatusTooManyRequests,
			wantLogMessage:     logRateLimitAboveMaxDelay,
		},
		{
			name: "nil rate limiter",
			args: args{
				b: nil,
			},
			wantLogMessage:     logRateLimiterDisabled,
			wantHTTPStatusCode: http.StatusOK,
		},
	}

	logger, hook := testLog.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()

			rlh := WithRateLimiter(logger, tt.args.b, time.Nanosecond)
			mockResponse := httptest.NewRecorder()
			mockRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

			rlh(mux).ServeHTTP(mockResponse, mockRequest)

			if mockResponse.Code != tt.wantHTTPStatusCode {
				t.Errorf("WithRateLimiter() = %v, want %v", mockResponse.Code, tt.wantHTTPStatusCode)
			}

			if tt.wantLogMessage != "" {
				le := hook.LastEntry()
				if le == nil {
					t.Errorf("Expected a log entry, but none was generated.")
					return
				}

				if le.Message != tt.wantLogMessage {
					t.Errorf("Expected the message %q, but instead I got %q", tt.wantLogMessage, le.Message)
				}
			}
		})
	}

	t.Run("Untyped nil arg", func(t *testing.T) {
		mux := http.NewServeMux()

		logger, hook := testLog.NewNullLogger()
		h := WithRateLimiter(logger, nil, time.Hour*1)

		mockResponse := httptest.NewRecorder()
		mockRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		h(mux).ServeHTTP(mockResponse, mockRequest)

		entry := hook.LastEntry()
		if entry == nil {
			t.Errorf("Expected a log being generated, for noop wrapping")
			return
		}

		if got := entry.Message; got != logRateLimiterDisabled {
			t.Errorf("Expected the message %q, but instead I got %q", logRateLimiterDisabled, got)
		}
	})
}

type takeMaxDurationStub struct {
	delay           time.Duration
	withinThreshold bool
}

func (t *takeMaxDurationStub) TakeMaxDuration(_ int64, _ time.Duration) (time.Duration, bool) {
	return t.delay, t.withinThreshold
}
