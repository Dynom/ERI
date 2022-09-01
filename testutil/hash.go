package testutil

type MockHasherReverse struct {
	MockHasher
}

func (s MockHasherReverse) Sum(p []byte) []byte {
	r := make([]byte, len(p))
	for i := range p {
		r[len(r)-1-i], r[i] = p[i], p[len(p)-1-i]
	}

	return r
}

type MockHasher struct {
	v []byte
}

func (s *MockHasher) Write(p []byte) (int, error) {
	s.v = p
	return len(p), nil
}

func (s MockHasher) Sum(p []byte) []byte {
	return p
}

func (s MockHasher) Reset() {
}

func (s MockHasher) Size() int {
	return len(s.v)
}

func (s MockHasher) BlockSize() int {
	return 128
}
