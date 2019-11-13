package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/config"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/types"

	"github.com/Dynom/ERI/inspector"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
)

// Version contains the app version, the value is changed during compile time to the appropriate Git tag
var Version = "dev"

func main() {
	var conf config.Config
	var err error

	conf, err = config.NewConfig("config.toml")
	if err != nil {
		panic(err)
	}

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Out = os.Stdout
	logger.Level, err = logrus.ParseLevel(conf.Server.Log.Level)

	if err != nil {
		panic(err)
	}

	logger.WithFields(logrus.Fields{
		"version": Version,
	}).Info("Starting up...")

	mux := http.NewServeMux()

	checker := inspector.New(inspector.WithValidators(
		inspector.ValidateSyntax(),
		inspector.ValidateMXAndRCPT(inspector.DefaultRecipient),
	))

	cache := hitlist.NewHitList()
	myFinder, err := finder.New(
		cache.GetValidAndUsageSortedDomains(),
		finder.WithLengthTolerance(0.2),
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
	)

	if err != nil {
		panic(err)
	}

	checkSvc := services.NewCheckService(&cache, myFinder, checker)

	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.CheckRequest

		defer func() {
			// Body's can be nil on GET requests
			if r.Body != nil {
				_ = r.Body.Close()
			}
		}()

		body, err := erihttp.GetBodyFromHTTPRequest(r)
		if err != nil {
			logger.WithError(err).Errorf("Error handling request %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			logger.WithError(err).Errorf("Error handling request body %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		// @todo should the timeout be for the entire request, or just Check ?
		ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		defer cancel()

		// -
		email, err := types.NewEmailParts(req.Email)
		if err != nil {
			logger.WithError(err).Errorf("Email address can't be decomposed %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to decompose e-mail address"))
			return
		}

		checkResult, err := checkSvc.HandleCheckRequest(ctx, email, req.Alternatives)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"result": checkResult,
				"error":  err,
			}).Error("Failed to check e-mail address")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		response, err := json.Marshal(erihttp.CheckResponse{
			Valid:       checkResult.Valid,
			Reason:      "",
			Alternative: checkResult.Alternative,
		})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"result":   checkResult,
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		logger.WithFields(logrus.Fields{
			"cache_ttl_sec": int(checkResult.CacheHitTTL.Seconds()),
			"result":        checkResult,
		}).Debugf("Done performing check")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	})

	mux.HandleFunc("/dumphl", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%+v\n", cache.Set)
		_, _ = fmt.Fprintf(w, "%+v\n", cache.GetValidAndUsageSortedDomains())
	})

	mux.HandleFunc("/learn", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.LearnRequest

		defer r.Body.Close()

		body, err := erihttp.GetBodyFromHTTPRequest(r)
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

			err := cache.LearnEmailAddress(toLearn.Value, v)
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

			err := cache.LearnDomain(toLearn.Value, v)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"value": toLearn.Value,
					"error": err.Error(),
				}).Warn("Unable to learn domain")
			}
		}

		logger.Info("Refreshing domains")
		l := cache.GetValidAndUsageSortedDomains()
		myFinder.Refresh(l)
		logger.WithFields(logrus.Fields{"domain_amount": len(l)}).Info("Refreshed domains")
	})

	lw := logger.WriterLevel(logger.Level)
	defer func() {
		_ = lw.Close()
	}()

	s := erihttp.BuildHTTPServer(mux, conf, lw,
		handlers.WithGzipHandler(),
		handlers.WithHeaders(sliceToHTTPHeaders(conf.Server.Headers)),
	)

	logger.WithFields(logrus.Fields{
		"listen_on": conf.Server.ListenOn,
	}).Info("Done, serving requests")
	err = s.ListenAndServe()

	logger.Errorf("HTTP server stopped %s", err)
}
