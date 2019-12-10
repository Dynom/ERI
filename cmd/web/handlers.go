package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/types"
	"github.com/sirupsen/logrus"
)

func NewCheckHandler(logger *logrus.Logger, svc services.CheckSvc) http.HandlerFunc {

	log := logger.WithField("handler", "health")
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

func NewHealthHandler(logger *logrus.Logger) http.HandlerFunc {

	log := logger.WithField("handler", "health")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.WithError(err).Error("failed to write in health handler")
		}
	}
}

func NewLearnHandler(logger *logrus.Logger, svc services.LearnSvc) http.HandlerFunc {

	log := logger.WithField("handler", "learn")
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var req erihttp.LearnRequest

		log = log.WithContext(r.Context())

		defer func() {
			// Body's can be nil on GET requests
			if r.Body != nil {
				_ = r.Body.Close()
			}
		}()

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

		go func() {
			for status := range svc.ResultStream {
				e := log.WithFields(logrus.Fields{
					"validations": status.Validations,
					"value":       status.Value,
					"type":        status.Type,
				})

				if status.Error != nil {
					e = e.WithError(status.Error)
				}

				e.Debug("Learn status")
			}
		}()

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

func NewDebugHandler(cache *hitlist.HitList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var domains = make([]string, 0, len(cache.Set))
		for d := range cache.Set {
			domains = append(domains, d)
		}

		sort.Strings(domains)
		for _, domain := range domains {
			_, _ = fmt.Fprintf(w, "%016b | %s \n", cache.Set[domain].Validations, domain)

			recipients, err := cache.GetRCPTsForDomain(domain)
			if err != nil {
				_, _ = fmt.Fprintf(w, "err: %s\n", err)
				continue
			}

			if len(recipients) > 0 {
				sort.Slice(recipients, func(i, j int) bool {
					return recipients[i] < recipients[j]
				})
				_, _ = fmt.Fprint(w, "\tValidations      | cache ttl                 | recipient \n")

				for _, rcpt := range recipients {
					hit, err := cache.GetHit(domain, rcpt)
					if err != nil {
						_, _ = fmt.Fprintf(w, "err: %s\n", err)
						continue
					}
					_, _ = fmt.Fprintf(w, "\t%016b | %25s | %s \n", hit.Validations, time.Now().Add(hit.TTL()).Format(time.RFC3339), rcpt)
				}
			}
		}

		_, _ = fmt.Fprintf(w, "%+v\n", cache.GetValidAndUsageSortedDomains())
	}
}
