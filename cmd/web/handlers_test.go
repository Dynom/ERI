package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/cmd/web/services"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
	"github.com/Dynom/TySug/finder"
	testLog "github.com/sirupsen/logrus/hooks/test"
)

func TestNewAutoCompleteHandler(t *testing.T) {
	const maxBodySize = 1024
	logger, hook := testLog.NewNullLogger()

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

	tooLongArgumentValidStructureRequest := erihttp.AutoCompleteRequest{
		Domain: strings.Repeat("a", 255),
	}

	tooLongArgumentValidStructureRequestBody, err := json.Marshal(&tooLongArgumentValidStructureRequest)
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
			requestBody: strings.NewReader(strings.Repeat(".", int(maxBodySize)+1)),
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
			name:        "Input too long",
			requestBody: bytes.NewReader(tooLongArgumentValidStructureRequestBody),
			ctx:         context.Background(),
			want: wants{
				statusCode: 400,
			},
		},
		{
			name:        "Expired context",
			requestBody: bytes.NewReader(validRequestBody),
			ctx:         expiredContext,
			want: wants{
				statusCode: 200,
			},
		},
	}

	svc := services.NewAutocompleteService(myFinder, hitList, 0, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			hook.Reset()
			handlerFunc := NewAutoCompleteHandler(logger, svc, 10, maxBodySize)

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
	logger, hook := testLog.NewNullLogger()

	t.Run("simple GET", func(t *testing.T) {

		hook.Reset()
		handlerFunc := NewHealthHandler(logger)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		handlerFunc.ServeHTTP(rec, req)

		want := 200
		if rec.Code != want {
			t.Errorf("NewHealthHandler() = %+v, want %+v", rec, want)
		}
	})
}

func TestNewSuggestHandler(t *testing.T) {
	const maxBodySize = 1024
	logger, hook := testLog.NewNullLogger()

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

	validRequest := erihttp.SuggestRequest{
		Email: "john@example.org",
	}

	validRequestBody, err := json.Marshal(&validRequest)
	if err != nil {
		t.Errorf("Test setup failed, %s", err)
		t.FailNow()
	}

	emptyArgumentValidStructureRequest := erihttp.SuggestRequest{}
	emptyArgumentValidStructureRequestBody, err := json.Marshal(&emptyArgumentValidStructureRequest)
	if err != nil {
		t.Errorf("Test setup failed, %s", err)
		t.FailNow()
	}

	expiredContext, c := context.WithCancel(context.Background())
	c()

	t.Run("HTTP interaction", func(t *testing.T) {
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
				requestBody: strings.NewReader(strings.Repeat(".", int(maxBodySize)+1)),
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
					statusCode: 200,
				},
			},
			{
				name:        "Expired context",
				requestBody: bytes.NewReader(validRequestBody),
				ctx:         expiredContext,
				want: wants{
					statusCode: 200,
				},
			},
		}

		var val validator.CheckFn = func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
			return validator.Result{
				Validations: validations.Validations(validations.FValid | validations.FSyntax | validations.FMXLookup),
				Steps:       validations.Steps(validations.FValid | validations.FSyntax | validations.FMXLookup),
			}
		}

		svc := services.NewSuggestService(myFinder, val, nil, logger)

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				hook.Reset()
				handlerFunc := NewSuggestHandler(logger, svc, maxBodySize)

				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/", tt.requestBody)
				req = req.WithContext(tt.ctx)
				req.Header.Set("Content-Type", "application/json")

				handlerFunc.ServeHTTP(rec, req)

				if tt.want.statusCode != rec.Code {
					t.Errorf("NewSuggestHandler() = %+v, want %+v", rec, tt.want)

					b, _ := ioutil.ReadAll(rec.Result().Body)
					t.Logf("Body: %s", b)
					for _, l := range hook.AllEntries() {
						t.Logf("Logs: %s", l.Message)
						t.Logf("Meta: %v", l.Data)
					}
				}
			})
		}
	})

	t.Run("Functional", func(t *testing.T) {

		// Setup
		refs := []string{"gmail.com", "example.org", "mail.com"}
		myFinder, err := finder.New(refs, finder.WithAlgorithm(finder.NewJaroWinklerDefaults()))
		if err != nil {
			t.Errorf("Test setup failed, %s", err)
			t.FailNow()
		}

		t.Run("Run finder on malformed input as well", func(t *testing.T) {

			// resetting logger
			hook.Reset()

			// Stub a malformed input
			var val validator.CheckFn = func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
				return validator.Result{
					Validations: validations.Validations(0),
					Steps:       validations.Steps(validations.FSyntax | validations.FMXLookup),
				}
			}

			// Building the service
			svc := services.NewSuggestService(myFinder, val, nil, logger)
			handlerFunc := NewSuggestHandler(logger, svc, maxBodySize)

			// Setting up the request
			req := httptest.NewRequest(http.MethodPost, "/", createSuggestRequestBytesReader(t, "nonexisting@exampleorg"))
			req.Header.Set("Content-Type", "application/json")

			// Recording
			rec := httptest.NewRecorder()
			handlerFunc.ServeHTTP(rec, req)

			response := restoreSuggestResponse(t, rec.Result().Body)

			// Assert
			if response.Alternatives[0] != "nonexisting@example.org" {
				t.Errorf("Expected Finder to correct 'exampleorg' to 'example.org', instead we got: %+v", response.Alternatives)
			}

			if response.MalformedSyntax != true {
				t.Errorf("Expected the response to reflect that the input was erroneous, instead we got: %+v", response)
			}
		})
	})
}

func restoreSuggestResponse(t *testing.T, r io.Reader) erihttp.SuggestResponse {
	responseStr, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	var responseStructure erihttp.SuggestResponse
	err = json.Unmarshal(responseStr, &responseStructure)
	if err != nil {
		t.Fatal(err)
	}

	return responseStructure
}

func createSuggestRequestBytesReader(t *testing.T, email string) io.Reader {
	requestStructure := erihttp.SuggestRequest{
		Email: email,
	}
	requestStr, err := json.Marshal(requestStructure)
	if err != nil {
		t.Fatal(err)
	}

	return bytes.NewReader(requestStr)
}
