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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dynom/ERI/inspector"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
)

// Version contains the app version, the value is changed during compile time to the appropriate Git tag
var Version = "dev"
var ErrNotPresent = errors.New("value not present")

type lookup struct {
	inspector.Validations           // The type of validations performed (bit mask)
	ValidUntil            time.Time // The TTL
}

func newHitList() HitList {
	return HitList{
		Set:  make(map[string]Domain),
		lock: sync.RWMutex{},
	}
}

type HitList struct {
	Set  map[string]Domain
	lock sync.RWMutex
}

type Domain struct {
	inspector.Validations
	RCPTs map[string]lookup
}

func (HitList) splitParts(email string) (string, string, error) {
	i := strings.LastIndex(email, "@")
	if i <= 0 || i >= len(email) {

		// @todo improve error situation
		return "", "", errors.New("argument is not an e-mail address")
	}

	email = strings.ToLower(email)
	return email[:i], email[i+1:], nil
}

// GetValidAndUsageSortedDomains returns the used domains, sorted by their usage
func (h *HitList) GetValidAndUsageSortedDomains() []string {
	var now = time.Now()

	type stats struct {
		domain string
		usage  int
	}

	h.lock.RLock()
	defer h.lock.RUnlock()

	var d = make([]stats, len(h.Set))

	var index int
	for domain, details := range h.Set {
		var usage int

		// Count "valid" usage. If a domain has 0 valid leafs, the domain isn't valid
		for _, leaf := range details.RCPTs {
			if leaf.ValidUntil.After(now) && leaf.Validations.IsValid() {
				usage++
			}
		}

		if usage > 0 {
			d[index] = stats{
				domain: domain,
				usage:  usage,
			}
		}

		index++
	}

	sort.Slice(d, func(i, j int) bool {
		return d[i].usage < d[j].usage
	})

	var result = make([]string, 0, len(d))
	for _, stats := range d {
		result = append(result, stats.domain)
	}

	return result
}

func (h *HitList) HasDomain(d string) bool {
	d = strings.ToLower(d)

	h.lock.RLock()
	_, ok := h.Set[d]
	h.lock.RUnlock()

	return ok
}

func (h *HitList) GetForEmail(email string) (lookup, error) {

	rcpt, domain, err := h.splitParts(email)
	if err != nil {
		return lookup{}, err
	}

	h.lock.RLock()
	r, ok := h.Set[domain].RCPTs[rcpt]
	h.lock.RUnlock()

	if !ok || time.Since(r.ValidUntil) > 0 {
		// @todo improve error situation
		return lookup{}, ErrNotPresent
	}

	return r, nil
}

// LearnEmailAddress records validations for a particular e-mail address. LearnEmailAddress clears previously seen
// validators if you want to merge, first fetch, merge and pass the resulting Validations to LearnEmailAddress()
func (h *HitList) LearnEmailAddress(address string, validations inspector.Validations) error {

	rcpt, domain, err := h.splitParts(address)
	if err != nil {
		return err
	}

	if !h.HasDomain(domain) {
		v := validations
		if !isValidationsForValidDomain(v) {
			v.MarkAsInvalid()
		}

		err := h.LearnDomain(domain, v)
		if err != nil {
			return err
		}
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.Set[domain].RCPTs[rcpt] = lookup{
		ValidUntil:  time.Now().Add(time.Second * 60),
		Validations: h.Set[domain].RCPTs[rcpt].Validations.Merge(validations),
	}

	return nil
}

// LearnDomain learns of a domain and it's validity.
func (h *HitList) LearnDomain(domain string, validations inspector.Validations) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if v, ok := h.Set[domain]; !ok {
		h.Set[domain] = Domain{
			RCPTs:       make(map[string]lookup),
			Validations: validations,
		}
	} else {
		v.Validations = v.Validations.Merge(validations)
		h.Set[domain] = v
	}

	return nil
}

// isValidationsForValidDomain checks if a set of validations really marks a domain as valid.
func isValidationsForValidDomain(validations inspector.Validations) bool {
	// @todo figure out what we consider "valid", perhaps we should drop the notion of "valid" and instead be more explicit .HasValidSyntax, etc.
	return validations&inspector.VFMXLookup == 1
}

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

	hl := newHitList()
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
		if err == ErrNotPresent {
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
			var v inspector.Validations
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
			var v inspector.Validations
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
