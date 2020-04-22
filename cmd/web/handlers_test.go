package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/cmd/web/services"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
	testLog "github.com/sirupsen/logrus/hooks/test"
)

func TestNewAutoCompleteHandler(t *testing.T) {
	logger, hook := testLog.NewNullLogger()
	_ = hook

	refs := []string{
		"a", "b", "c", "d",
		// Testing for > 5 matches
		"exam", "example", "examination", "excalibur", "exceptional", "extra",
	}

	myFinder, err := finder.New(refs, finder.WithAlgorithm(finder.NewJaroWinklerDefaults()))
	if err != nil {
		t.Errorf("Test setup failed, %s", err)
		t.FailNow()
	}

	hitList := hitlist.New(nil, time.Minute*1)

	validRequest := erihttp.AutoCompleteRequest{
		Domain: "ex",
	}

	validRequestBody, err := json.Marshal(&validRequest)
	if err != nil {
		t.Errorf("Test setup failed, %s", err)
		t.FailNow()
	}

	emptyArgumentValidStructureRequest := erihttp.AutoCompleteRequest{}
	emptyArgumentValidStructureRequestBody, err := json.Marshal(&emptyArgumentValidStructureRequest)
	if err != nil {
		t.Errorf("Test setup failed, %s", err)
		t.FailNow()
	}

	expiredContext, c := context.WithCancel(context.Background())
	c()

	type wants struct {
		statusCode int
	}
	tests := []struct {
		name        string
		requestBody io.Reader
		ctx         context.Context
		want        wants
	}{
		{
			name:        "correct POST body",
			requestBody: bytes.NewReader(validRequestBody),
			ctx:         context.Background(),
			want: wants{
				statusCode: 200,
			},
		},
		{
			name:        "malformed POST body",
			requestBody: strings.NewReader("burp"),
			ctx:         context.Background(),
			want: wants{
				statusCode: 400,
			},
		},
		{
			name:        "nil POST body",
			requestBody: nil,
			ctx:         context.Background(),
			want: wants{
				statusCode: 400,
			},
		},
		{
			name:        "Too large POST body",
			requestBody: strings.NewReader(strings.Repeat(".", int(erihttp.MaxBodySize)+1)),
			ctx:         context.Background(),
			want: wants{
				statusCode: 400,
			},
		},
		{
			name:        "Bad JSON",
			requestBody: bytes.NewReader(validRequestBody[0 : len(validRequestBody)-1]), // stripping off the '}'
			ctx:         context.Background(),
			want: wants{
				statusCode: 400,
			},
		},
		{
			name:        "Empty input",
			requestBody: bytes.NewReader(emptyArgumentValidStructureRequestBody),
			ctx:         context.Background(),
			want: wants{
				statusCode: 400,
			},
		},
		{
			name:        "Bad context",
			requestBody: bytes.NewReader(validRequestBody),
			ctx:         expiredContext,
			want: wants{
				statusCode: 400,
			},
		},
	}

	svc := services.NewAutocompleteService(myFinder, hitList, 0, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			hook.Reset()
			handlerFunc := NewAutoCompleteHandler(logger, svc, 10)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", tt.requestBody)
			req = req.WithContext(tt.ctx)
			req.Header.Set("Content-Type", "application/json")

			handlerFunc.ServeHTTP(rec, req)

			if tt.want.statusCode != rec.Code {
				t.Errorf("NewAutoCompleteHandler() = %+v, want %+v", rec, tt.want)

				b, _ := ioutil.ReadAll(rec.Result().Body)
				t.Logf("Body: %s", b)
				for _, l := range hook.AllEntries() {
					t.Logf("Logs: %s", l.Message)
					t.Logf("Meta: %v", l.Data)
				}
			}
		})
	}
}

func TestNewHealthHandler(t *testing.T) {
	type args struct {
		logger logrus.FieldLogger
	}
	tests := []struct {
		name string
		args args
		want http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHealthHandler(tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHealthHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSuggestHandler(t *testing.T) {
	type args struct {
		logger logrus.FieldLogger
		svc    services.SuggestSvc
	}
	tests := []struct {
		name string
		args args
		want http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSuggestHandler(tt.args.logger, tt.args.svc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSuggestHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
