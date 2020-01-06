package hitlist

import (
	"encoding/hex"
	"errors"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/Dynom/ERI/types"
)

var (
	ErrNotPresent = errors.New("value not present")
)

func New(h hash.Hash, ttl time.Duration) *HitList {
	return &HitList{
		Set:  make(map[string]domainHit),
		lock: sync.RWMutex{},
		h:    h,
		ttl:  ttl,
	}
}

// HitList is an opinionated nearly-flat tree structured type that captures e-mail address validity on two levels
type HitList struct {
	Set  map[string]domainHit
	ttl  time.Duration // The default TTL for when Learning about a new e-mail address
	lock sync.RWMutex
	h    hash.Hash
}

type Hit struct {
	Validations validations.Validations // The type of validations performed (bit mask)
	ValidUntil  time.Time               // The TTL
}

func (h Hit) TTL() time.Duration {
	return time.Until(h.ValidUntil)
}

type Recipient string

func (rcpt Recipient) String() string {
	return hex.EncodeToString([]byte(rcpt))
}

type Recipients map[Recipient]Hit

type domainHit struct {
	validations.Validations
	RCPTs       Recipients
	validRCTPTs uint // An approximate number of recipients
}

type stats struct {
	domain string
	usage  uint
}

// GetValidAndUsageSortedDomains returns the used domains, sorted by their associated recipients (high>low)
func (h *HitList) GetValidAndUsageSortedDomains() []string {

	h.lock.RLock()
	var domains = getValidAndUsedFromSet(h.Set)
	h.lock.RUnlock()

	sort.Slice(domains, func(i, j int) bool {
		return domains[i].usage < domains[j].usage
	})

	var result = make([]string, 0, len(domains))
	for _, stats := range domains {
		result = append(result, stats.domain)
	}

	return result
}

// HasDomain performs a string-to-lower on the argument and returns true if there is a match
func (h *HitList) HasDomain(domain string) bool {
	domain = strings.ToLower(domain)

	h.lock.RLock()
	_, ok := h.Set[domain]
	h.lock.RUnlock()

	return ok
}

// GetRCPTsForDomain performs a string-to-lower on the argument and returns hashed "Recipients" for a given domain.
func (h *HitList) GetRCPTsForDomain(domain string) ([]Recipient, error) {
	domain = strings.ToLower(domain)

	h.lock.RLock()
	defer h.lock.RUnlock()

	_, ok := h.Set[domain]
	if !ok {
		return []Recipient{}, ErrNotPresent
	}

	var recipients = make([]Recipient, 0, len(h.Set[domain].RCPTs))
	for recipient := range h.Set[domain].RCPTs {
		recipients = append(recipients, recipient)
	}

	return recipients, nil
}

// GetForEmail performs a string-to-lower on the argument and returns it's corresponding Hit, if a match was found
func (h *HitList) GetForEmail(email string) (Hit, error) {
	var domain string
	var safeLocal Recipient

	{
		email = strings.ToLower(email)
		parts, err := types.NewEmailParts(email)
		if err != nil {
			return Hit{}, err
		}

		safeLocal = Recipient(h.h.Sum([]byte(parts.Local)))
		domain = parts.Domain
	}

	// @todo -- Since most typos appear to be at the end of a domain, does it make sense to reverse the domain name?

	h.lock.RLock()
	r, ok := h.Set[domain].RCPTs[safeLocal]
	h.lock.RUnlock()

	if !ok || r.ValidUntil.Before(time.Now()) {
		// @todo improve error situation
		return Hit{}, ErrNotPresent
	}

	return r, nil
}

// GetHit is a fairly low-level function that returns the hit based on two knows, the RCPT and the domain
// It performs a string-to-lower on the domain
func (h *HitList) GetHit(domain string, rcpt Recipient) (Hit, error) {
	domain = strings.ToLower(domain)

	h.lock.RLock()
	r, ok := h.Set[domain].RCPTs[rcpt]
	h.lock.RUnlock()

	if !ok {
		// @todo improve error situation
		return Hit{}, ErrNotPresent
	}

	return r, nil
}

// AddEmailAddressDeadline Same as AddEmailAddress, but allows for custom TTL. Duration shouldn't be negative.
func (h *HitList) AddEmailAddressDeadline(email string, validations validations.Validations, duration time.Duration) error {
	var domain string
	var safeLocal Recipient
	{
		email = strings.ToLower(email)
		parts, err := types.NewEmailParts(email)
		if err != nil {
			return err
		}

		safeLocal = Recipient(h.h.Sum([]byte(parts.Local)))
		domain = parts.Domain
	}

	if !h.HasDomain(domain) {
		err := h.AddDomain(domain, validations)
		if err != nil {
			return err
		}
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	v, ok := h.Set[domain]
	if !ok {
		return errors.New("domain doesn't appear in set, this is... unexpected")
	}

	var now = time.Now()
	v.RCPTs[safeLocal] = Hit{
		ValidUntil:  now.Add(duration),
		Validations: v.RCPTs[safeLocal].Validations.MergeWithNext(validations),
	}

	if validations.IsValid() {
		u := calculateValidRCPTUsage(v.RCPTs, now)
		v.validRCTPTs = u + 1
	}

	h.Set[domain] = v

	return nil
}

// AddEmailAddress records validations for a particular e-mail address. AddEmailAddress clears previously seen
// validators if you want to merge, first fetch, merge and pass the resulting Validations to AddEmailAddress()
func (h *HitList) AddEmailAddress(email string, validations validations.Validations) error {
	return h.AddEmailAddressDeadline(email, validations, h.ttl)
}

// AddDomain learns of a domain and it's validity. It overwrites the existing validations, when applicable for a domain
func (h *HitList) AddDomain(domain string, validations validations.Validations) error {
	var now = time.Now()

	if validator.MightBeAHostOrIP(domain) && (validations.IsValid() || validations.IsValidationsForValidDomain()) {
		validations.MarkAsValid()
	} else {
		validations.MarkAsInvalid()
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	if hit, ok := h.Set[domain]; !ok {
		h.Set[domain] = domainHit{
			RCPTs:       make(map[Recipient]Hit),
			Validations: validations,
		}
	} else {
		usage := calculateValidRCPTUsage(hit.RCPTs, now)

		hit.Validations = hit.Validations.MergeWithNext(validations)
		hit.validRCTPTs = usage

		h.Set[domain] = hit
	}

	return nil
}

// getValidAndUsedFromSet returns domains which are valid
func getValidAndUsedFromSet(set map[string]domainHit) []stats {
	var result = make([]stats, len(set))

	var index int
	for domain, details := range set {

		if !details.Validations.IsValid() {
			continue
		}

		result[index] = stats{
			domain: domain,
			usage:  details.validRCTPTs,
		}

		index++
	}

	return result
}

// calculateValidRCPTUsage calculates the usage of valid, and the first-to-expire valid recipients
func calculateValidRCPTUsage(recipients map[Recipient]Hit, referenceTime time.Time) (usage uint) {

	for _, recipient := range recipients {
		if !recipient.Validations.IsValid() {
			continue
		}

		if referenceTime.IsZero() || recipient.ValidUntil.IsZero() {
			continue
		}

		// The recipient's validity expired, we won't consider it for "oldest"
		if recipient.ValidUntil.Before(referenceTime) {
			continue
		}

		usage++
	}

	return
}
