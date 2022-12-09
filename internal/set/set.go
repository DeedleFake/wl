package set

type Set[T comparable] map[T]struct{}

func New[T comparable](vals ...T) Set[T] {
	s := make(Set[T], len(vals))
	for _, v := range vals {
		s.Add(v)
	}
	return s
}

func (s Set[T]) Add(v T) {
	s[v] = struct{}{}
}

func (s Set[T]) Has(v T) bool {
	_, ok := s[v]
	return ok
}
