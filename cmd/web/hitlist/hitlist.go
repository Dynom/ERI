package hitlist

import (
	"encoding/hex"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/Dynom/ERI/types"
)

func New(h hash.Hash, ttl time.Duration, options ...Option) *HitList {
	l := HitList{
		set:        make(map[string]domainHit),
		lock:       sync.RWMutex{},
		notifyLock: sync.RWMutex{},
		h:          h,
		ttl:        ttl,
		toNotify:   make([]OnChangeFn, 0),
	}

	for _, o := range options {
		o(&l)
	}

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
}

type stats struct {
	domain string
	usage  uint
}

type ChangeType uint
type OnChangeFn func(recipient Recipient, domain string, vr validator.Result, change ChangeType)

// GetValidAndUsageSortedDomains returns the used domains, sorted by their associated recipients (high>low)
func (h *HitList) GetValidAndUsageSortedDomains() []string {

	h.lock.RLock()
	var domains = getValidAndUsedFromSet(h.set)
	h.lock.RUnlock()

	sort.Slice(domains, func(i, j int) bool {
		return domains[i].usage < domains[j].usage
	})

	// @todo Could probably be a object pool
	var result = make([]string, 0, len(domains))
	for _, stats := range domains {
		result = append(result, stats.domain)
	}

	return result
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

	h.lock.Lock()
	defer h.lock.Unlock()

	var now = time.Now()
	dh, ok := h.set[domain]

	if !ok {
		recipients := make(map[Recipient]Hit, 1)
		recipients[safeLocal] = Hit{
			ValidUntil:       now.Add(duration),
			ValidationResult: vr,
		}

		dh = domainHit{
			RCPTs:            recipients,
			ValidationResult: vr,
		}

		h.set[domain] = dh

		return nil
	}

	if _, ok := dh.RCPTs[safeLocal]; !ok {
		dh.RCPTs[safeLocal] = Hit{
			ValidUntil:       now.Add(duration),
			ValidationResult: vr,
		}
	}

	dh.ValidationResult.Validations = dh.ValidationResult.Validations.MergeWithNext(vr.Validations)
	dh.ValidationResult.Steps = dh.ValidationResult.Steps.MergeWithNext(vr.Steps)

	h.set[domain] = dh

	return nil
}

// AddEmailAddress records validations for a particular e-mail address. AddEmailAddress clears previously seen
// validators if you want to merge, first fetch, merge and pass the resulting Validations to AddEmailAddress()
func (h *HitList) AddEmailAddress(email string, vr validator.Result) error {
	return h.AddEmailAddressDeadline(email, vr, h.ttl)
}

// AddDomain learns of a domain and it's validity. It overwrites the existing validations, when applicable for a domain
func (h *HitList) AddDomain(domain string, vr validator.Result) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	domain = strings.ToLower(domain)

	hit, ok := h.set[domain]
	if !ok {
		h.set[domain] = domainHit{
			RCPTs:            make(map[Recipient]Hit),
			ValidationResult: vr,
		}

		return nil
	}

	hit.ValidationResult.Validations = hit.ValidationResult.Validations.MergeWithNext(vr.Validations)
	hit.ValidationResult.Steps = hit.ValidationResult.Steps.MergeWithNext(vr.Steps)
	h.set[domain] = hit

	return nil
}

// getValidAndUsedFromSet returns domains which are valid
func getValidAndUsedFromSet(set map[string]domainHit) []stats {
	var result = make([]stats, len(set))

	var now = time.Now()
	var index int
	for domain, details := range set {

		if !details.ValidationResult.Validations.IsValidationsForValidDomain() {
			continue
		}

		result[index] = stats{
			domain: domain,
			usage:  calculateValidRCPTUsage(details.RCPTs, now), // @todo get out of the hot path
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
