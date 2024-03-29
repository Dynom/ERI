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

const (
	failedRequestError      = "Request failed, unable to parse request body. Expected JSON."
	domainLookupFailedError = "Request failed, unable to lookup by domain."
	failedResponseError     = "Generating response failed."
)

type marshalFn func(v interface{}) ([]byte, error)

func NewAutoCompleteHandler(logger logrus.FieldLogger, svc *services.AutocompleteSvc, maxSuggestions, maxBodySize uint64, jsonMarshaller marshalFn) http.HandlerFunc {
	if jsonMarshaller == nil {
		jsonMarshaller = json.Marshal
	}

	logger = logger.WithField("handler", "auto complete")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.AutoCompleteRequest

		logger := logger.WithField(handlers.RequestID.String(), r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, logger)

		body, err := erihttp.GetBodyFromHTTPRequest(r, int64(maxBodySize))
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
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: failedRequestError})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		defer cancel()

		if req.Domain == "" {
			logger.Debug("Empty argument")
			w.WriteHeader(http.StatusBadRequest)
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: domainLookupFailedError})
			return
		}

		result, err := svc.Autocomplete(ctx, req.Domain, maxSuggestions)
		if err != nil {
			logger.WithError(err).Warn("Error during lookup")

			if err != ctx.Err() {
				// When the context is canceled, we're not going to consider it a bad request
				w.WriteHeader(http.StatusBadRequest)
			}

			// @todo is this a safe error to leak?
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: err.Error()})
			return
		}

		response, err := jsonMarshaller(erihttp.AutoCompleteResponse{
			Suggestions: result.Suggestions,
		})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			writeErrorJSONResponse(logger, w, &erihttp.AutoCompleteResponse{Error: failedResponseError})
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

// NewSuggestHandler constructs an HTTP handler that deals with suggestion requests
func NewSuggestHandler(logger logrus.FieldLogger, svc *services.SuggestSvc, maxBodySize uint64, jsonMarshaller marshalFn) http.HandlerFunc {
	if jsonMarshaller == nil {
		jsonMarshaller = json.Marshal
	}

	log := logger.WithField("handler", "suggest")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.SuggestRequest

		log := log.WithField(handlers.RequestID.String(), r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, log)

		body, err := erihttp.GetBodyFromHTTPRequest(r, int64(maxBodySize))
		if err != nil {
			log.WithError(err).Error("Error handling request")
			w.WriteHeader(http.StatusBadRequest)

			writeErrorJSONResponse(logger, w, &erihttp.SuggestResponse{Error: err.Error()})
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			log.WithError(err).Error("Error handling request body")
			w.WriteHeader(http.StatusBadRequest)
			writeErrorJSONResponse(logger, w, &erihttp.SuggestResponse{Error: failedRequestError})
			return
		}

		alts := []string{req.Email}
		result, sugErr := svc.Suggest(r.Context(), req.Email)
		if len(result.Alternatives) > 0 {
			alts = append(alts[0:0], result.Alternatives...)
		}

		sr := erihttp.SuggestResponse{
			Alternatives:    alts,
			MalformedSyntax: errors.Is(sugErr, validator.ErrEmailAddressSyntax),
			MisconfiguredMX: !result.HasValidMX,
		}

		if sugErr != nil {
			log.WithFields(logrus.Fields{
				"suggest_response": sr,
				"error":            sugErr,
				"input":            req.Email,
			}).Warn("Suggest error")
			sr.Error = sugErr.Error()
		}

		response, err := jsonMarshaller(sr)
		if err != nil {
			log.WithFields(logrus.Fields{
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			writeErrorJSONResponse(logger, w, &erihttp.SuggestResponse{Error: failedResponseError})
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
	ok := []byte("OK")

	logger = logger.WithField("handler", "health")
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logger.WithField(handlers.RequestID.String(), r.Context().Value(handlers.RequestID))

		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write(ok)
		if err != nil {
			logger.WithError(err).Error("failed to write to http.ResponseWriter")
		}
	}
}
