package hitlist

import (
	"errors"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
)

var (
	ErrInvalidDomainSyntax = errors.New("invalid domain syntax")
	ErrInvalidSyntax       = errors.New("invalid syntax")
)

type (
	Hits   map[Domain]Hit
	Domain string
	Hit    struct {
		Recipients       map[rcpt]struct{}
		ValidUntil       time.Time
		ValidationResult validator.Result
	}
)

type (
	Recipient []byte
	rcpt      string
)

func New(h hash.Hash, ttl time.Duration) *HitList {
	l := HitList{
		hits: make(Hits),
		lock: sync.RWMutex{},
		h:    h,
		ttl:  ttl,
	}

	return &l
}

type HitList struct {
	hits Hits
	ttl  time.Duration
	lock sync.RWMutex
	h    hash.Hash
}

// Has returns true if HitList knows about (part of) the argument
func (hl *HitList) Has(parts types.EmailParts) (domain, local bool) {
	var hit Hit

	if parts.Domain == "" {
		return
	}

	inputDomain := Domain(strings.ToLower(parts.Domain))

	hl.lock.RLock()
	defer hl.lock.RUnlock()

	if hit, domain = hl.hits[inputDomain]; domain {
		if parts.Local == "" {
			return
		}

		inputLocal := strings.ToLower(parts.Local)
		recipient := rcpt(hl.h.Sum([]byte(inputLocal)))

		_, local = hit.Recipients[recipient]
	}

	return
}

// CreateInternalTypes returns the Recipient and Domain types for an Email Type Parts. It's stateless, and solely
// works on the input. The input is not allowed to have empty parts. Typical use-case is when wanting to persist the
// "safe" value, to later re-add to a HitList
func (hl *HitList) CreateInternalTypes(p types.EmailParts) (domain Domain, recipient Recipient, err error) {
	if len(p.Domain) == 0 || len(p.Local) == 0 {
		recipient = Recipient("")
		err = ErrInvalidSyntax
		return
	}

	domain = Domain(strings.ToLower(p.Domain))
	recipient = hl.h.Sum([]byte(strings.ToLower(p.Local)))
	return
}

func (hl *HitList) GetDomainValidationDetails(d Domain) (validator.Details, bool) {
	hl.lock.RLock()
	hit, ok := hl.hits[d]
	hl.lock.RUnlock()

	if ok {
		return validator.Details{
			Result:     hit.ValidationResult,
			ValidUntil: hit.ValidUntil,
		}, ok
	}

	return validator.Details{}, ok
}

// GetValidAndUsageSortedDomains returns the used domains, sorted by their associated recipients (high>low)
func (hl *HitList) GetValidAndUsageSortedDomains() []string {
	hl.lock.RLock()
	domains := getValidDomains(hl.hits)
	hl.lock.RUnlock()

	return domains
}

// GetRecipientCount returns the amount of recipients known for a domain
func (hl *HitList) GetRecipientCount(d Domain) (amount uint64) {
	hl.lock.RLock()
	if hit, exists := hl.hits[d]; exists {
		amount = uint64(len(hit.Recipients))
	}
	hl.lock.RUnlock()

	return
}

// AddInternalParts adds values considered "safe". Typically you would only use this on provisioning HitList from a storage layer
func (hl *HitList) AddInternalParts(domain Domain, recipient Recipient, vr validator.Result) error {
	return hl.AddInternalPartsDuration(domain, recipient, vr, hl.ttl)
}

// AddInternalPartsDuration adds values considered "safe". Has an extra duration option which shouldn't be negative
func (hl *HitList) AddInternalPartsDuration(domain Domain, recipient Recipient, vr validator.Result, duration time.Duration) error {
	hl.lock.Lock()
	defer hl.lock.Unlock()

	now := time.Now()
	dh, ok := hl.hits[domain]

	if !ok {

		hl.hits[domain] = Hit{
			Recipients:       map[rcpt]struct{}{rcpt(recipient): {}},
			ValidUntil:       now.Add(duration),
			ValidationResult: vr,
		}

		return nil
	}

	dh.ValidationResult.Validations = dh.ValidationResult.Validations.MergeWithNext(vr.Validations)
	dh.ValidationResult.Steps = dh.ValidationResult.Steps.MergeWithNext(vr.Steps)
	dh.ValidUntil = now.Add(duration)
	dh.Recipients[rcpt(recipient)] = struct{}{}

	hl.hits[domain] = dh

	return nil
}

func (hl *HitList) Add(parts types.EmailParts, vr validator.Result) error {
	return hl.AddDeadline(parts, vr, hl.ttl)
}

// AddDeadline is similar to Add, but allows for custom TTL. Duration shouldn't be negative.
func (hl *HitList) AddDeadline(parts types.EmailParts, vr validator.Result, duration time.Duration) error {
	if parts.Local == "" {
		return hl.AddDomain(parts.Domain, vr)
	}

	domain, safeLocal, err := hl.CreateInternalTypes(parts)
	if err != nil {
		return err
	}

	return hl.AddInternalPartsDuration(domain, safeLocal, vr, duration)
}

// AddEmailAddress records validations for a particular e-mail address.
func (hl *HitList) AddEmailAddress(email string, vr validator.Result) error {
	parts, err := types.NewEmailParts(email)
	if err != nil {
		return err
	}

	return hl.AddDeadline(parts, vr, hl.ttl)
}

// AddDomain learns of a domain and it's validity.
func (hl *HitList) AddDomain(d string, vr validator.Result) error {
	domain := Domain(strings.ToLower(d))

	if len(domain) == 0 {
		return ErrInvalidDomainSyntax
	}

	hl.lock.Lock()
	defer hl.lock.Unlock()

	hit, ok := hl.hits[domain]
	if !ok {
		hl.hits[domain] = Hit{
			Recipients:       map[rcpt]struct{}{},
			ValidUntil:       time.Now().Add(hl.ttl),
			ValidationResult: vr,
		}

		return nil
	}

	hit.ValidationResult.Validations = hit.ValidationResult.Validations.MergeWithNext(vr.Validations)
	hit.ValidationResult.Steps = hit.ValidationResult.Steps.MergeWithNext(vr.Steps)
	hl.hits[domain] = hit

	return nil
}

// getValidDomains returns domains which are valid, sorted by their recipients in descending order
func getValidDomains(hits Hits) []string {
	type stats struct {
		Domain     string
		Recipients int64
	}

	sortStats := make([]stats, 0, len(hits))
	for domain, details := range hits {

		if !details.ValidationResult.Validations.IsValidationsForValidDomain() {
			continue
		}

		sortStats = append(sortStats, stats{
			Domain:     string(domain),
			Recipients: int64(len(details.Recipients)),
		})
	}

	// Sorting on recipient count in Descending order
	sort.Slice(sortStats, func(i, j int) bool {
		return sortStats[i].Recipients > sortStats[j].Recipients
	})

	// @todo Could probably be an object pool, could relieve the GC
	result := make([]string, 0, len(sortStats))
	for _, stats := range sortStats {
		result = append(result, stats.Domain)
	}

	return result
}
