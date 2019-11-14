package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/types"
	"github.com/sirupsen/logrus"
)

func NewCheckHandler(logger *logrus.Logger, svc services.CheckSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		checkResult, err := svc.HandleCheckRequest(ctx, email, req.Alternatives)
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
			"target":        email.Address,
		}).Debugf("Done performing check")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}

func NewLearnHandler(logger *logrus.Logger, svc services.LearnSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.LearnRequest

		defer func() {
			// Body's can be nil on GET requests
			if r.Body != nil {
				_ = r.Body.Close()
			}
		}()

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

		var result services.LearnResult
		result, err = svc.HandleLearnRequest(r.Context(), req)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("Unable to handle learn request")
		}

		logger.WithFields(logrus.Fields{
			"domains": result.NumDomains,
			"emails":  result.NumEmailAddresses,
		}).Debug("Finished refresh request")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("Refreshed %d domain(s) and %d e-mail address(es)", result.NumDomains, result.NumEmailAddresses)))
	}
}
