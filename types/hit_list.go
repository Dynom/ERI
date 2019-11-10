package types

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

func NewHitList() HitList {
	return HitList{
		Set:  make(map[string]Domain),
		lock: sync.RWMutex{},
	}
}

type HitList struct {
	Set  map[string]Domain
	lock sync.RWMutex
}

type Lookup struct {
	Validations           // The type of validations performed (bit mask)
	ValidUntil  time.Time // The TTL
}

type Domain struct {
	Validations
	RCPTs map[string]Lookup
}

var ErrNotPresent = errors.New("value not present")

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

func (h *HitList) GetForEmail(email string) (Lookup, error) {

	rcpt, domain, err := splitParts(email)
	if err != nil {
		return Lookup{}, err
	}

	h.lock.RLock()
	r, ok := h.Set[domain].RCPTs[rcpt]
	h.lock.RUnlock()

	if !ok || time.Since(r.ValidUntil) > 0 {
		// @todo improve error situation
		return Lookup{}, ErrNotPresent
	}

	return r, nil
}

// LearnEmailAddress records validations for a particular e-mail address. LearnEmailAddress clears previously seen
// validators if you want to merge, first fetch, merge and pass the resulting Validations to LearnEmailAddress()
func (h *HitList) LearnEmailAddress(address string, validations Validations) error {

	rcpt, domain, err := splitParts(address)
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

	h.Set[domain].RCPTs[rcpt] = Lookup{
		ValidUntil:  time.Now().Add(time.Second * 60),
		Validations: h.Set[domain].RCPTs[rcpt].Validations.Merge(validations),
	}

	return nil
}

// LearnDomain learns of a domain and it's validity.
func (h *HitList) LearnDomain(domain string, validations Validations) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if v, ok := h.Set[domain]; !ok {
		h.Set[domain] = Domain{
			RCPTs:       make(map[string]Lookup),
			Validations: validations,
		}
	} else {
		v.Validations = v.Validations.Merge(validations)
		h.Set[domain] = v
	}

	return nil
}

// isValidationsForValidDomain checks if a set of validations really marks a domain as valid.
func isValidationsForValidDomain(validations Validations) bool {
	// @todo figure out what we consider "valid", perhaps we should drop the notion of "valid" and instead be more explicit .HasValidSyntax, etc.
	return validations&VFMXLookup == 1
}

func splitParts(email string) (string, string, error) {
	i := strings.LastIndex(email, "@")
	if i <= 0 || i >= len(email) {

		// @todo improve error situation
		return "", "", errors.New("argument is not an e-mail address")
	}

	email = strings.ToLower(email)
	return email[:i], email[i+1:], nil
}
