package services

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Dynom/ERI/cmd/web/preferrer"
	"github.com/sirupsen/logrus"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/TySug/finder"
)

func createMockValidator(v, s validations.Flag) validator.CheckFn {
	return func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
		return validator.Result{
			Validations: validations.Validations(v),
			Steps:       validations.Steps(s),
		}
	}
}

func TestSuggestSvc_Suggest(t *testing.T) {
	finderOptions := []finder.Option{
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
		finder.WithLengthTolerance(0.2),
	}

	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name        string
		email       string
		want        SuggestResult
		wantErr     bool
		validator   validator.CheckFn
		finderList  []string
		logContains string
		preferMap   preferrer.Mapping
	}{
		{
			name:       "All good",
			email:      "john.doe@example.org",
			want:       SuggestResult{Alternatives: []string{"john.doe@example.org"}},
			wantErr:    false,
			validator:  createMockValidator(validations.FSyntax|validations.FValid, validations.FSyntax|validations.FValid),
			finderList: []string{},
		},
		{
			name:       "Including preferred",
			email:      "john.doe@example.com",
			want:       SuggestResult{Alternatives: []string{"john.doe@example.org", "john.doe@example.com"}},
			wantErr:    false,
			validator:  createMockValidator(validations.FSyntax|validations.FValid, validations.FSyntax|validations.FValid),
			finderList: []string{"example.com", "example.org"},
			preferMap:  preferrer.Mapping{"example.com": "example.org"},
		},
		{
			name:       "Invalid domain, should fall back on finder",
			email:      "john.doe@example.or",
			want:       SuggestResult{Alternatives: []string{"john.doe@example.org"}},
			wantErr:    false,
			validator:  createMockValidator(validations.FSyntax, validations.FSyntax),
			finderList: []string{"example.org"},
		},
		{
			name:       "Invalid domain, should fall back on finder and be corrected by preferrer",
			email:      "john.doe@example.cm",
			want:       SuggestResult{Alternatives: []string{"john.doe@example.org"}},
			wantErr:    false,
			validator:  createMockValidator(validations.FSyntax, validations.FSyntax),
			finderList: []string{"example.org"},
			preferMap:  preferrer.Mapping{"example.com": "example.org"},
		},
		{
			name:       "Invalid domain, finder has no alternative",
			email:      "john.doe@example.or",
			want:       SuggestResult{Alternatives: []string{"john.doe@example.or"}},
			wantErr:    false,
			validator:  createMockValidator(validations.FSyntax, validations.FSyntax),
			finderList: []string{"be"}, // Note: Violates the finder.WithLengthTolerance filter, so won't be used
		},
		{
			name:        "Malformed",
			email:       " john.doe@example.org", // leading space
			want:        SuggestResult{Alternatives: []string{" john.doe@example.org"}},
			wantErr:     true,
			validator:   createMockValidator(0, validations.FSyntax),
			finderList:  []string{},
			logContains: "Input doesn't have a valid structure",
		},
		{
			name:        "Malformed",
			email:       "john.doe#example.org", // Missing @, fails a sanity check, earlier than validator
			want:        SuggestResult{Alternatives: []string{"john.doe#example.org"}},
			wantErr:     true,
			validator:   nil, // Validator should never be reached
			finderList:  []string{},
			logContains: "Unable to split input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()

			f, err := finder.New(tt.finderList, finderOptions...)
			if err != nil {
				t.Errorf("Unable to prepare for tests %q", err)
				return
			}

			p := preferrer.New(tt.preferMap)

			svc := NewSuggestService(f, tt.validator, p, logger)
			got, err := svc.Suggest(context.Background(), tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("Suggest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Suggest() got = %v, want %v", got, tt.want)
			}

			if !containsLogWhileExpected(tt.logContains, hook.Entries) {
				t.Errorf("Expected the log message %q to have been generated, it wasn't %+v", tt.logContains, hook.Entries)
			}
		})
	}
}

// containsLogWhileExpected returns false when a log was expected, but not found in any of the entries.
func containsLogWhileExpected(expected string, entries []logrus.Entry) (found bool) {
	if expected == "" {
		return true
	}

	for _, e := range entries {
		if strings.Contains(e.Message, expected) {
			found = true
			return
		}
	}

	return
}

func Test_didDeadlineExpire(t *testing.T) {
	ctx := context.Background()

	ctxExpired, _ := context.WithDeadline(ctx, time.Now())
	ctxCanceled, c := context.WithCancel(ctx)
	c()

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "basic", ctx: ctx},

		{want: true, name: "expired", ctx: ctxExpired},
		{want: true, name: "canceled", ctx: ctxCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := didDeadlineExpire(tt.ctx); got != tt.want {
				t.Errorf("didDeadlineExpire() = %v, want %v", got, tt.want)
			}
		})
	}
}
