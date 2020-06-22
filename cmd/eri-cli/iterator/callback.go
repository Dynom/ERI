package iterator

// NewCallbackIterator provides an iterator interface based on closure callbacks
func NewCallbackIterator(next func() bool, value func() (string, error), close func() error) *CallbackIterator {
	return &CallbackIterator{
		next:  next,
		value: value,
		close: close,
	}
}

type CallbackIterator struct {
	next  func() bool
	value func() (string, error)
	close func() error
}

// Next returns true if we have more iterations pending
func (i *CallbackIterator) Next() bool {
	return i.next()
}

// Value returns the current value, and/or an error
func (i *CallbackIterator) Value() (string, error) {
	return i.value()
}

// Close performs any cleanups. It may be used to return the last error
func (i *CallbackIterator) Close() error {
	return i.close()
}
