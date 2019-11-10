package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Dynom/ERI/types"

	"github.com/Dynom/ERI/inspector"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
)

// Version contains the app version, the value is changed during compile time to the appropriate Git tag
var Version = "dev"

func main() {
	var err error

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Out = os.Stdout
	logger.Level = logrus.DebugLevel

	logger.WithFields(logrus.Fields{
		"version": Version,
	}).Info("Starting up...")

	mux := http.NewServeMux()

	checker := inspector.New(inspector.WithValidators(
		inspector.ValidateSyntax(),
		inspector.ValidateMXAndRCPT(inspector.DefaultRecipient),
	))

	hl := types.NewHitList()
	myFinder, err := finder.New(
		hl.GetValidAndUsageSortedDomains(),
		finder.WithLengthTolerance(0.2),
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
	)

	if err != nil {
		panic(err)
	}

	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req checkRequest

		defer r.Body.Close()

		body, err := getBodyFromHTTPRequest(r)
		if err != nil {
			logger.WithFields(logrus.Fields{"error": err}).Errorf("Error handling request %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			logger.WithFields(logrus.Fields{"error": err}).Errorf("Error handling request body %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		// @todo should the timeout be for the entire request, or just Check ?
		ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		defer cancel()

		l, err := hl.GetForEmail(req.Email)
		if err == types.ErrNotPresent {
			// @todo perform domain check
		}

		var result inspector.Result
		var cached bool
		if err == nil {
			cached = true
			result = inspector.Result{
				Error:       nil,
				Timings:     nil,
				Validations: l.Validations,
			}
		} else {
			result = checker.Check(ctx, req.Email)

			// @todo Learn from this lookup, if configured as such
			// @todo Only learn when it's worth learning. A simple syntax-validation shouldn't be enough
			err = hl.LearnEmailAddress(req.Email, result.Validations)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"result": result,
					"error":  err,
				}).Error("Failed to learn about the check")
			}
		}

		if !cached && result.IsValid() {
			// @todo add "update" thing to finder
			myFinder, err = finder.New(
				hl.GetValidAndUsageSortedDomains(),
				finder.WithLengthTolerance(0.2),
				finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
			)

			// @todo fix
			if err != nil {
				panic(err)
			}
		}

		var res = checkResponse{
			Valid: result.Validations.IsValid(),
		}

		if req.Alternatives {
			// @todo context might've expired, but alt's were requested. Split timeouts
			ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
			defer cancel()

			// @todo provide alternatives
			local, domain, err := splitLocalAndDomain(req.Email)
			if err == nil {
				alt, score, _ := myFinder.FindCtx(ctx, domain)
				alt = local + "@" + alt

				logger.WithFields(logrus.Fields{
					"alternative": alt,
					"original":    req.Email,
					"score":       score,
				}).Debug("Alternative search result")

				if score > finder.WorstScoreValue {
					res.Alternative = alt
				}
			}
		}

		logger.WithFields(logrus.Fields{
			"result":    result,
			"request":   req,
			"cache_hit": cached,
			"lookup":    l,
		}).Debugf("Sent a reply for %q", req.Email)

		response, err := json.Marshal(res)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"result":   result,
				"response": res,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	})

	mux.HandleFunc("/dumphl", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%+v\n", hl.Set)
		_, _ = fmt.Fprintf(w, "%+v\n", hl.GetValidAndUsageSortedDomains())
	})

	mux.HandleFunc("/learn", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req learnRequest

		defer r.Body.Close()

		body, err := getBodyFromHTTPRequest(r)
		if err != nil {
			logger.WithFields(logrus.Fields{"error": err}).Errorf("Error handling request %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			logger.WithFields(logrus.Fields{"error": err}).Errorf("Error handling request body %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		for _, toLearn := range req.Emails {
			var v types.Validations
			if toLearn.Valid {
				v.MarkAsValid()
			}

			err := hl.LearnEmailAddress(toLearn.Value, v)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"value": toLearn.Value,
					"error": err.Error(),
				}).Warn("Unable to learn e-mail address")
			}
		}

		for _, toLearn := range req.Domains {
			var v types.Validations
			if toLearn.Valid {
				v.MarkAsValid()
			}

			err := hl.LearnDomain(toLearn.Value, v)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"value": toLearn.Value,
					"error": err.Error(),
				}).Warn("Unable to learn domain")
			}
		}

		logger.Info("Refreshing domains")
		l := hl.GetValidAndUsageSortedDomains()
		myFinder.Refresh(l)
		logger.WithFields(logrus.Fields{"domain_amount": len(l)}).Info("Refreshed domains")
	})

	server := &http.Server{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second, // Is overridden, when the profiler is enabled.
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 19, // 512 kb
		Handler:           mux,
		Addr:              "localhost:1338",
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

//{"valid": false, "reason": "bad_domain",         "alternative": "john.doe@gmail.com"}
type checkResponse struct {
	Valid       bool   `json:"valid"`
	Reason      string `json:"reason,omitempty"`
	Alternative string `json:"alternative,omitempty"`
}

type checkRequest struct {
	Email        string `json:"email"`
	Alternatives bool   `json:"with_alternatives"`
}

type learnRequest struct {
	Emails  []ToLearn `json:"emails"`
	Domains []ToLearn `json:"domains"`
}

type ToLearn struct {
	Value string `json:"value"`
	Valid bool   `json:"valid"`
}

var (
	ErrMissingBody    = errors.New("missing body")
	ErrInvalidRequest = errors.New("request is invalid")
	ErrBodyTooLarge   = errors.New("request body too large")
)

func getBodyFromHTTPRequest(r *http.Request) ([]byte, error) {
	var empty []byte
	const maxSizePlusOne int64 = 1<<20 + 1

	if r.Body == nil {
		return empty, ErrMissingBody
	}

	b, err := ioutil.ReadAll(io.LimitReader(r.Body, maxSizePlusOne))
	if err != nil {
		if err == io.EOF {
			return empty, ErrMissingBody
		}
		return empty, ErrInvalidRequest
	}

	if int64(len(b)) == maxSizePlusOne {
		return empty, ErrBodyTooLarge
	}

	return b, nil
}
