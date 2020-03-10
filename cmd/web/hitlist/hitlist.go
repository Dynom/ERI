package hitlist

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/types"
)

var (
	ErrNotPresent = errors.New("value not present")
)

const (
	_         ChangeType = iota
	ChangeAdd            = iota
)

func New(h hash.Hash, ttl time.Duration, options ...Option) *HitList {
	l := HitList{
		set:        make(map[string]domainHit),
		lock:       sync.RWMutex{},
		notifyLock: sync.RWMutex{},
		h:          h,
		ttl:        ttl,
		toNotify:   make([]OnChangeFn, 0),
		semLimit:   10,
	}

	for _, o := range options {
		o(&l)
	}

	l.sem = make(chan struct{}, l.semLimit)

	return &l
}

// HitList is an opinionated nearly-flat tree structured type that captures e-mail address validity on two levels
type HitList struct {
	set        map[string]domainHit
	ttl        time.Duration // The default TTL for when Learning about a new e-mail address
	lock       sync.RWMutex
	notifyLock sync.RWMutex
	h          hash.Hash
	toNotify   []OnChangeFn
	sem        chan struct{}
	semLimit   uint
}

type Hit struct {
	ValidationResult validator.Result
	ValidUntil       time.Time // The TTL
	Domain           domainHit
}

func (h Hit) TTL() time.Duration {
	return time.Until(h.ValidUntil)
}

type Recipient string

func (rcpt Recipient) String() string {
	return hex.EncodeToString([]byte(rcpt))
}

func (rcpt Recipient) IsEmpty() bool {
	return rcpt == ""
}

type Recipients map[Recipient]Hit

type DomainHit struct {
	ValidationResult validator.Result
}

type domainHit struct {
	ValidationResult validator.Result
	RCPTs            Recipients
	validRCTPTs      uint // An approximate number of recipients
}

type stats struct {
	domain string
	usage  uint
}

type ChangeType uint
type OnChangeFn func(recipient Recipient, domain string, vr validator.Result, change ChangeType)

// RegisterOnChange accepts functions which are invoked, in unspecified order, whenever a mutation happens
func (h *HitList) RegisterOnChange(fn ...OnChangeFn) {
	h.notifyLock.Lock()
	h.toNotify = append(h.toNotify, fn...)
	h.notifyLock.Unlock()
}

func (h *HitList) notify(recipient Recipient, domain string, vr validator.Result, changeType ChangeType) {

	h.notifyLock.RLock()
	defer h.notifyLock.RUnlock()

	for _, notify := range h.toNotify {
		h.sem <- struct{}{}

		go func(fn OnChangeFn) {
			fn(recipient, domain, vr, changeType)
			<-h.sem
		}(notify)
	}
}

// GetValidAndUsageSortedDomains returns the used domains, sorted by their associated recipients (high>low)
func (h *HitList) GetValidAndUsageSortedDomains() []string {

	h.lock.RLock()
	var domains = getValidAndUsedFromSet(h.set)
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
	_, ok := h.set[domain]
	h.lock.RUnlock()

	return ok
}

// GetRCPTsForDomain performs a string-to-lower on the argument and returns hashed "Recipients" for a given domain.
func (h *HitList) GetRCPTsForDomain(domain string) ([]Recipient, error) {
	domain = strings.ToLower(domain)

	h.lock.RLock()
	defer h.lock.RUnlock()

	_, ok := h.set[domain]
	if !ok {
		return []Recipient{}, ErrNotPresent
	}

	var recipients = make([]Recipient, 0, len(h.set[domain].RCPTs))
	for recipient := range h.set[domain].RCPTs {
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

	hit := Hit{}
	h.lock.RLock()
	defer h.lock.RUnlock()

	if d, ok := h.set[domain]; ok {
		if h, ok := d.RCPTs[safeLocal]; ok && h.ValidUntil.Before(time.Now()) {
			hit = h
		}
		hit.Domain = d
	} else {
		return Hit{}, ErrNotPresent
	}

	return hit, nil
}

// GetForDomain performs a string-to-lower on the argument and returns it's corresponding Hit, if a match was found
func (h *HitList) GetForDomain(domain string) (Hit, error) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	domain = strings.ToLower(domain)
	if r, ok := h.set[domain]; ok {
		return Hit{
			ValidationResult: r.ValidationResult,
		}, nil
	}

	return Hit{}, ErrNotPresent
}

// GetHit is a fairly low-level function that returns the hit based on two knows, the RCPT and the domain
// It performs a string-to-lower on the domain
func (h *HitList) GetHit(domain string, rcpt Recipient) (Hit, error) {
	domain = strings.ToLower(domain)

	h.lock.RLock()
	r, ok := h.set[domain].RCPTs[rcpt]
	h.lock.RUnlock()

	if !ok {
		// @todo improve error situation
		return Hit{}, ErrNotPresent
	}

	return r, nil
}

func (h *HitList) getDomainHit(domain string) (domainHit, error) {
	domain = strings.ToLower(domain)

	h.lock.RLock()
	r, ok := h.set[domain]
	h.lock.RUnlock()

	if !ok {
		// @todo improve error situation
		return domainHit{}, ErrNotPresent
	}

	return r, nil
}

// AddEmailAddressDeadline Same as AddEmailAddress, but allows for custom TTL. Duration shouldn't be negative.
func (h *HitList) AddEmailAddressDeadline(email string, vr validator.Result, duration time.Duration) error {
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
		err := h.addDomain(domain, vr)
		if err != nil {
			return err
		}
	}

	// @todo possible race-condition
	domainHit, err := h.getDomainHit(domain)
	if err != nil {
		return fmt.Errorf("domain doesn't exist, this is unexpected %w", err)
	}

	var now = time.Now()
	hit := Hit{
		ValidUntil:       now.Add(duration),
		ValidationResult: vr,
	}

	// @todo
	//hit.ValidationResult = domainHit.RCPTs[safeLocal].ValidationResult.Validations.MergeWithNext(validations)

	domainHit.RCPTs[safeLocal] = hit

	if vr.Validations.IsValid() {
		u := calculateValidRCPTUsage(domainHit.RCPTs, now)
		domainHit.validRCTPTs = u + 1
	}

	h.lock.Lock()
	h.set[domain] = domainHit
	h.lock.Unlock()

	h.notify(safeLocal, domain, hit.ValidationResult, ChangeAdd)

	return nil
}

// AddEmailAddress records validations for a particular e-mail address. AddEmailAddress clears previously seen
// validators if you want to merge, first fetch, merge and pass the resulting Validations to AddEmailAddress()
func (h *HitList) AddEmailAddress(email string, vr validator.Result) error {
	return h.AddEmailAddressDeadline(email, vr, h.ttl)
}

// AddDomain learns of a domain and it's validity. It overwrites the existing validations, when applicable for a domain
func (h *HitList) AddDomain(domain string, vr validator.Result) error {
	err := h.addDomain(domain, vr)
	if err == nil {
		h.notify("", domain, vr, ChangeAdd)
	}

	return err
}

// AddDomain learns of a domain and it's validity. It overwrites the existing validations, when applicable for a domain
func (h *HitList) addDomain(domain string, vr validator.Result) error {
	var now = time.Now()

	if !validator.MightBeAHostOrIP(domain) {
		return errors.New("argument isn't considered a valid domain")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	if hit, ok := h.set[domain]; !ok {
		h.set[domain] = domainHit{
			RCPTs:            make(map[Recipient]Hit),
			ValidationResult: vr,
		}
	} else {
		hit.ValidationResult.Validations = hit.ValidationResult.Validations.MergeWithNext(vr.Validations)
		hit.validRCTPTs = calculateValidRCPTUsage(hit.RCPTs, now)

		h.set[domain] = hit
	}

	return nil
}

// getValidAndUsedFromSet returns domains which are valid
func getValidAndUsedFromSet(set map[string]domainHit) []stats {
	var result = make([]stats, len(set))

	var index int
	for domain, details := range set {

		if !details.ValidationResult.Validations.IsValidationsForValidDomain() {
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
		if !recipient.ValidationResult.Validations.IsValid() {
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
