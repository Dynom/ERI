package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/TySug/finder"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/types"
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

		ctx, cancel := context.WithCancel(r.Context())
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

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)

	}
}
func NewCheckHandler(logger logrus.FieldLogger, svc services.CheckSvc) http.HandlerFunc {

	/*********************************************************************************************************************
			Expected API:
			URL /validate
			Responsibility: Validate an argument with the specified validators, doesn't persist the result

			-->
				email:string
				validators:[]string
		  <--
		  	{
					...result
				}


			URL /check
			Responsibility: Give a brief summary of an input, calls learn on newly found addresses

			--->
				email:string
				include_alternatives:bool

			<---
				{
					"validations_passed": ["structure", "lookup", "connect"], // 0 or more, unique list
					"alternatives": [] // 0 or more, unique string
				}


			URL /suggest
			Responsibility: Return a list of suggestions for the input.
			Notes:
			-	The input is returned if it's the best suggestion


			--->
				email:string

			<--- 200
				{
					"alternatives": [] // 0 or more, unique string
				}

			<-- 400
				{
					"error": "Address couldn't be validated",
					"code": "validation_structure"
				}
	*********************************************************************************************************************/

	log := logger.WithField("handler", "check")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.CheckRequest

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

		// @todo should the timeout be for the entire request, or just Check ?
		//ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// -
		email, err := types.NewEmailParts(req.Email)
		if err != nil {
			log.WithError(err).Errorf("Email address can't be decomposed %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to decompose e-mail address"))
			return
		}

		checkResult, err := svc.HandleCheckRequest(ctx, email, req.Alternatives)
		if err != nil {
			log.WithFields(logrus.Fields{
				"result":  checkResult,
				"error":   err,
				"ctx_err": ctx.Err(),
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
			log.WithFields(logrus.Fields{
				"result":   checkResult,
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		log.WithFields(logrus.Fields{
			"cache_ttl_sec": int(checkResult.CacheHitTTL.Seconds()),
			"result":        checkResult,
			"target":        email.Address,
		}).Debugf("Done performing check")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}

func NewSuggestHandler(logger logrus.FieldLogger, svc services.SuggestSvc) http.HandlerFunc {
	log := logger.WithField("handler", "suggest")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.SuggestRequest

		log := log.WithField(handlers.RequestID, r.Context().Value(handlers.RequestID))

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

		// @todo should the timeout be for the entire request, or just Check ?
		//ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*500)
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		checkResult, err := svc.HandleRequest(ctx, req.Email)
		if err != nil {
			log.WithFields(logrus.Fields{
				"result":  checkResult,
				"error":   err,
				"ctx_err": ctx.Err(),
			}).Error("Failed to validate e-mail address")

			response, rerr := json.Marshal(erihttp.ErrorResponse{
				Error: err,
			})

			if rerr != nil {
				log.WithError(rerr).Error("Unable to marshal error response")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(response)
			return
		}

		response, err := json.Marshal(erihttp.SuggestResponse{
			Alternatives: checkResult.Alternatives,
		})
		if err != nil {
			log.WithFields(logrus.Fields{
				"result":   checkResult,
				"response": response,
				"error":    err,
			}).Error("Failed to marshal the response")

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to produce a response"))
			return
		}

		log.WithFields(logrus.Fields{
			"result": checkResult,
			"target": req.Email,
		}).Debugf("Done performing check")

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

func NewLearnHandler(logger logrus.FieldLogger, svc services.LearnSvc) http.HandlerFunc {

	log := logger.WithField("handler", "learn")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.LearnRequest

		log := log.WithField(handlers.RequestID, r.Context().Value(handlers.RequestID))

		defer deferClose(r.Body, log)

		body, err := erihttp.GetBodyFromHTTPRequest(r)
		if err != nil {
			log.WithFields(logrus.Fields{"error": err}).Errorf("Error handling request %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed"))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			log.WithFields(logrus.Fields{"error": err}).Errorf("Error handling request body %s", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Request failed, unable to parse request body. Did you send JSON?"))
			return
		}

		result := svc.HandleLearnRequest(r.Context(), req)
		log.WithFields(logrus.Fields{
			"domains_added": result.NumDomains - result.DomainErrors,
			"domain_errors": result.DomainErrors,
			"emails_added":  result.NumEmailAddresses - result.EmailAddressErrors,
			"email_errors":  result.EmailAddressErrors,
		}).Debug("Finished refresh request")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("Refreshed %d domain(s) and %d e-mail address(es)", result.NumDomains-result.DomainErrors, result.NumEmailAddresses-result.EmailAddressErrors)))
	}
}
