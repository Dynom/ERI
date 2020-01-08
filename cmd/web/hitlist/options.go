package hitlist

type Option func(list *HitList)

// WithMaxCallBackConcurrency sets the concurrency level of change callbacks.
func WithMaxCallBackConcurrency(max uint) Option {
	return func(list *HitList) {
		list.semLimit = max
	}
}
