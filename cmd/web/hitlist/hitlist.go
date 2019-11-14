package hitlist

import (
	"encoding/hex"
	"errors"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dynom/ERI/types"
)

var (
	ErrNotPresent      = errors.New("value not present")
	ErrNotAValidDomain = errors.New("argument doesn't appear to be a valid domain name")
)

func NewHitList(h hash.Hash) HitList {
	return HitList{
		Set:  make(map[string]domain),
		lock: sync.RWMutex{},
		h:    h,
	}
}

// HitList is an opinionated nearly-flat tree structured type that captures e-mail address validity on two levels
type HitList struct {
	Set  map[string]domain
	lock sync.RWMutex
	h    hash.Hash
}

type Hit struct {
	types.Validations           // The type of validations performed (bit mask)
	ValidUntil        time.Time // The TTL
}

type domain struct {
	types.Validations
	RCPTs map[RCPT]Hit
}

type RCPT string

func (rcpt RCPT) String() string {
	return hex.EncodeToString([]byte(rcpt))
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

func (h *HitList) GetForEmail(email string) (Hit, error) {

	parts, err := types.NewEmailParts(email)
	if err != nil {
		return Hit{}, err
	}

	safeLocal := RCPT(h.h.Sum([]byte(parts.Local)))

	h.lock.RLock()
	r, ok := h.Set[parts.Domain].RCPTs[safeLocal]
	h.lock.RUnlock()

	if !ok || r.ValidUntil.Before(time.Now()) {
		// @todo improve error situation
		return Hit{}, ErrNotPresent
	}

	return r, nil
}

// LearnEmailAddress records validations for a particular e-mail address. LearnEmailAddress clears previously seen
// validators if you want to merge, first fetch, merge and pass the resulting Validations to LearnEmailAddress()
func (h *HitList) LearnEmailAddress(address string, validations types.Validations) error {

	parts, err := types.NewEmailParts(address)
	if err != nil {
		return err
	}

	safeLocal := RCPT(h.h.Sum([]byte(parts.Local)))

	if !h.HasDomain(parts.Domain) {
		v := validations
		if !isValidationsForValidDomain(v) {
			v.MarkAsInvalid()
		}

		err := h.LearnDomain(parts.Domain, v)
		if err != nil {
			return err
		}
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.Set[parts.Domain].RCPTs[safeLocal] = Hit{
		// @todo make configurable
		ValidUntil:  time.Now().Add(time.Hour * 60),
		Validations: h.Set[parts.Domain].RCPTs[safeLocal].Validations.MergeWithNext(validations),
	}

	return nil
}

// LearnDomain learns of a domain and it's validity.
func (h *HitList) LearnDomain(d string, validations types.Validations) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if !mightBeAHostOrIP(d) {
		return ErrNotAValidDomain
	}

	if v, ok := h.Set[d]; !ok {
		h.Set[d] = domain{
			RCPTs:       make(map[RCPT]Hit),
			Validations: validations,
		}
	} else {
		v.Validations = v.Validations.MergeWithNext(validations)
		h.Set[d] = v
	}

	return nil
}

// isValidationsForValidDomain checks if a set of validations really marks a domain as valid.
func isValidationsForValidDomain(validations types.Validations) bool {
	// @todo figure out what we consider "valid", perhaps we should drop the notion of "valid" and instead be more explicit .HasValidSyntax, etc.
	return validations&types.VFMXLookup == 1
}

// mightBeAHostOrIP is a very rudimentary check to see if the argument could be either a host name or IP address
// It aims on speed and not for RFC compliance.
//nolint:gocyclo
func mightBeAHostOrIP(h string) bool {

	// Normally we can assume that host names have a tld or consists at least out of 4 characters
	if l := len(h); 4 >= l || l > 255 {
		return false
	}

	for _, c := range h {
		switch {
		case 48 <= c && c <= 57 /* 0-9 */ :
		case 65 <= c && c <= 90 /* A-Z */ :
		case 97 <= c && c <= 122 /* a-z */ :
		case c == 45 /* dash - */ :
		case c == 46 /* dot . */ :
		default:
			return false
		}
	}

	return true
}
