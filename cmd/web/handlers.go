package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/TySug/finder"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/sirupsen/logrus"
)

func NewAutoCompleteHandler(logger logrus.FieldLogger, myFinder *finder.Finder) http.HandlerFunc {

	log := logger.WithField("handler", "auto complete")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.AutoCompleteRequest

		log := log.WithField(handlers.RequestID, r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, log)

		body, err := erihttp.GetBodyFromHTTPRequest(r)
		if err != nil {
			log.WithError(err).Errorf("Error handling request %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			log.WithError(err).Errorf("Error handling request body %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		defer cancel()

		if len(req.Domain) == 0 {
			log.Error("Empty argument")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to lookup by domain"))
			return
		}

		list, err := myFinder.GetMatchingPrefix(ctx, req.Domain, 10)
		if err != nil {
			log.WithError(err).Errorf("Error during lookup %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Request failed, unable to lookup by domain"))
			return
		}

		response, err := json.Marshal(erihttp.AutoCompleteResponse{
			Suggestions: list,
		})
		if err != nil {
			log.WithFields(logrus.Fields{
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		log.WithFields(logrus.Fields{
			"suggestion_amount": len(list),
			"input":             req.Domain,
		}).Debugf("Done performing check")

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)

	}
}

func NewSuggestHandler(logger logrus.FieldLogger, svc services.SuggestSvc) http.HandlerFunc {
	log := logger.WithField("handler", "suggest")
	return func(w http.ResponseWriter, r *http.Request) {
		var sugErr error
		var req erihttp.SuggestRequest

		log := log.WithField(handlers.RequestID, r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, log)

		body, sugErr := erihttp.GetBodyFromHTTPRequest(r)
		if sugErr != nil {
			log.WithError(sugErr).Error("Error handling request")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		sugErr = json.Unmarshal(body, &req)
		if sugErr != nil {
			log.WithError(sugErr).Error("Error handling request body")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		var alts = []string{req.Email}
		result, sugErr := svc.Suggest(ctx, req.Email)
		if sugErr == nil && len(result.Alternatives) > 0 {
			alts = append(alts[0:0], result.Alternatives...)
		}

		response, err := json.Marshal(erihttp.SuggestResponse{
			Alternatives:    alts,
			MalformedSyntax: errors.Is(sugErr, validator.ErrEmailAddressSyntax),
		})

		if err != nil {
			log.WithFields(logrus.Fields{
				"response": response,
				"error":    sugErr,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		log.WithFields(logrus.Fields{
			"alternatives": alts,
			"target":       req.Email,
		}).Debugf("Done performing check")

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}

func NewHealthHandler(logger logrus.FieldLogger) http.HandlerFunc {

	log := logger.WithField("handler", "health")
	return func(w http.ResponseWriter, r *http.Request) {

		log := log.WithField(handlers.RequestID, r.Context().Value(handlers.RequestID))

		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.WithError(err).Error("failed to write in health handler")
		}
	}
}
