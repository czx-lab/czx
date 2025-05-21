package xslices

// Search returns the index of the first element in s that satisfies f, or -1 if no such element exists.
// It returns a boolean indicating whether the element was found.
func Search[T any](s []T, f func(T) bool) (int, bool) {
	for i, v := range s {
		if f(v) {
			return i, true
		}
	}
	return -1, false
}

// Reverse reverses the order of elements in a slice.
// It creates a new slice and returns it, leaving the original slice unchanged.
func Reverse[T any](s []T) []T {
	reversed := make([]T, len(s))
	copy(reversed, s)
	for i := range len(reversed) / 2 {
		j := len(reversed) - 1 - i
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return reversed
}
