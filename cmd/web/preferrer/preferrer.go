package preferrer

import "github.com/Dynom/ERI/types"

type HasPreferred interface {
	HasPreferred(parts types.EmailParts) (string, bool)
}

type Mapping map[string]string

func New(mapping Mapping) *Preferrer {
	return &Preferrer{
		m: mapping,
	}
}

type Preferrer struct {
	m Mapping
}

// HasPreferred returns the input when there isn't a match or a preferred result if it has. The second return argument
// should be used to discriminate between the two.
func (p *Preferrer) HasPreferred(parts types.EmailParts) (string, bool) {
	if l, ok := p.m[parts.Domain]; ok {
		return l, true
	}

	return parts.Domain, false
}
