package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/sirupsen/logrus"
)

func NewAutoCompleteHandler(logger logrus.FieldLogger, svc *services.AutocompleteSvc, maxSuggestions uint64) http.HandlerFunc {

	const (
		FailedRequestError      = "Request failed, unable to parse request body. Expected JSON."
		DomainLookupFailedError = "Request failed, unable to lookup by domain."
		FailedResponseError     = "Generating response failed."
	)

	logger = logger.WithField("handler", "auto complete")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.AutoCompleteRequest

		logger := logger.WithField(handlers.RequestID.String(), r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, logger)

		body, err := erihttp.GetBodyFromHTTPRequest(r)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":          err,
				"content_length": r.ContentLength,
			}).Errorf("Error handling request %s", err)

			w.WriteHeader(http.StatusBadRequest)

			// err is expected to be safe to expose to the client
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: err.Error()})
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			logger.WithError(err).Errorf("Error handling request body %s", err)

			w.WriteHeader(http.StatusBadRequest)
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: FailedRequestError})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		defer cancel()

		if req.Domain == "" {
			logger.Debug("Empty argument")
			w.WriteHeader(http.StatusBadRequest)
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: DomainLookupFailedError})
			return
		}

		result, err := svc.Autocomplete(ctx, req.Domain, maxSuggestions)
		if err != nil {
			logger.WithError(err).Warn("Error during lookup")
			w.WriteHeader(http.StatusBadRequest)

			// @todo Is this always a safe error to "leak" ?
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: DomainLookupFailedError})
			return
		}

		response, err := json.Marshal(erihttp.AutoCompleteResponse{
			Suggestions: result.Suggestions,
		})

		if err != nil {
			logger.WithFields(logrus.Fields{
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: FailedResponseError})
			return
		}

		logger.WithFields(logrus.Fields{
			"suggestions": len(result.Suggestions),
			"input":       req.Domain,
		}).Debugf("Autocomplete result")

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}

// NewSuggestHandler constructs a HTTP handler that deals with suggestion requests
func NewSuggestHandler(logger logrus.FieldLogger, svc services.SuggestSvc) http.HandlerFunc {
	log := logger.WithField("handler", "suggest")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.SuggestRequest

		log := log.WithField(handlers.RequestID.String(), r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, log)

		body, err := erihttp.GetBodyFromHTTPRequest(r)
		if err != nil {
			log.WithError(err).Error("Error handling request")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			log.WithError(err).Error("Error handling request body")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		var alts = []string{req.Email}
		var sugErr error
		{
			var result services.SuggestResult
			result, sugErr = svc.Suggest(ctx, req.Email)
			if sugErr == nil && len(result.Alternatives) > 0 {
				alts = append(alts[0:0], result.Alternatives...)
			}
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

		log := log.WithField(handlers.RequestID.String(), r.Context().Value(handlers.RequestID))

		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.WithError(err).Error("failed to write in health handler")
		}
	}
}
