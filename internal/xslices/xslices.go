package xslices

func Filter[T any, S ~[]T](s S, f func(T) bool) (r S) {
	r = make(S, 0, len(s))
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}
